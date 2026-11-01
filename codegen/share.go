package codegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/xsderr"
)

// unit is one target's package as phase 1 builds it: its name, its Dir made
// absolute, its import path (Target.Dir), its declarations in emission order
// with what each one's code names (refs, parallel to decls), the documents it
// imports from earlier targets (Generate's doc), and its source.
type unit struct {
	pkg, base, path string
	decls           []declaration
	refs            []analysis
	imports         []imported
	src             []byte
}

// analysis is what one declaration's code names: refs, the indices, ascending,
// of the package's other declarations whose type names its code names or
// whose marker method relates to it — an arm's interface, or an interface's
// arms — and usesBig, whether it names a math/big type.
type analysis struct {
	refs    []int
	usesBig bool
}

// imported is one document a unit imports: the URI of the Loc each of its
// declarations' headers renders, and the slice index of the target that
// defines it.
type imported struct {
	uri   string
	owner int
}

// newUnit returns the unit of the package pkg whose Dir made absolute is base,
// whose import path is path and whose declarations are decls, its source the
// file Generate writes for it when it imports nothing.
func newUnit(pkg, base, path string, decls []declaration) (*unit, error) {
	refs, err := analyze(decls)
	if err != nil {
		return nil, err
	}
	u := &unit{pkg: pkg, base: base, path: path, decls: decls, refs: refs}
	u.src, err = u.render(nil)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// analyze returns each declaration's analysis, in decls' order, by parsing
// its code: the identifiers in type positions, never a field's or a method's
// name, that name another declaration, the selectors qualified by big, and the
// marker methods relating an arm to its interface — an interface's method
// names and the name of the method a declaration's code declares.
func analyze(decls []declaration) ([]analysis, error) {
	shapes := make([]codeShape, len(decls))
	// index, interfaces and arms are lookup tables only and never iterated
	// (STYLE D2); arms' slices hold indices in decls' order.
	index := make(map[string]int, len(decls))
	interfaces := map[string]int{}
	arms := map[string][]int{}
	for i, d := range decls {
		s, err := parseShape(d)
		if err != nil {
			return nil, err
		}
		shapes[i] = s
		index[d.name] = i
		for _, m := range s.markers {
			if s.isInterface {
				interfaces[m] = i
				continue
			}
			arms[m] = append(arms[m], i)
		}
	}
	out := make([]analysis, len(decls))
	for i, s := range shapes {
		var refs []int
		for _, name := range s.idents {
			if j, ok := index[name]; ok && j != i {
				refs = append(refs, j)
			}
		}
		for _, m := range s.markers {
			if s.isInterface {
				refs = append(refs, arms[m]...)
				continue
			}
			if j, ok := interfaces[m]; ok {
				refs = append(refs, j)
			}
		}
		slices.Sort(refs)
		out[i] = analysis{refs: slices.Compact(refs), usesBig: s.usesBig}
	}
	return out, nil
}

// codeShape is what parseShape reads from one declaration's code.
type codeShape struct {
	idents      []string
	usesBig     bool
	markers     []string
	isInterface bool
}

// parseShape parses d's code as the one file of a package and reads its
// shape (analyze).
func parseShape(d declaration) (codeShape, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "", "package p\n"+d.code, parser.SkipObjectResolution)
	if err != nil {
		return codeShape{}, fmt.Errorf("parsing the declaration of %s: %w", d.name, err)
	}
	var s codeShape
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if ts.TypeParams != nil {
					s.collect(ts.TypeParams)
				}
				s.collect(ts.Type)
				iface, ok := ts.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				s.isInterface = true
				for _, m := range iface.Methods.List {
					for _, name := range m.Names {
						s.markers = append(s.markers, name.Name)
					}
				}
			}
		case *ast.FuncDecl:
			s.markers = append(s.markers, decl.Name.Name)
			if decl.Recv != nil {
				s.collect(decl.Recv)
			}
			s.collect(decl.Type)
		}
	}
	return s, nil
}

// collect records in s the identifiers n names in type positions, and
// whether it names a selector qualified by big.
func (s *codeShape) collect(n ast.Node) {
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Field:
			s.collect(n.Type)
			return false
		case *ast.SelectorExpr:
			if x, ok := n.X.(*ast.Ident); ok && x.Name == "big" {
				s.usesBig = true
			}
			return false
		case *ast.Ident:
			s.idents = append(s.idents, n.Name)
		}
		return true
	})
}

// docs returns the URIs of the Locs u's headers render, each once, in the
// order u's emission first attributes a declaration to each.
func (u *unit) docs() []string {
	var uris []string
	for _, d := range u.decls {
		if d.loc != nil && !slices.Contains(uris, d.loc.URI) {
			uris = append(uris, d.loc.URI)
		}
	}
	return uris
}

// set returns the indices, ascending, of u's declarations attributed to the
// document uri.
func (u *unit) set(uri string) []int {
	var set []int
	for i, d := range u.decls {
		if d.loc != nil && d.loc.URI == uri {
			set = append(set, i)
		}
	}
	return set
}

// closure returns the indices, ascending, of the declarations from reaches
// through refs, from's own included.
func (u *unit) closure(from []int) []int {
	reached := slices.Clone(from)
	for next := 0; next < len(reached); next++ {
		for _, j := range u.refs[reached[next]].refs {
			if !slices.Contains(reached, j) {
				reached = append(reached, j)
			}
		}
	}
	slices.Sort(reached)
	return reached
}

// key returns the texts of the declarations at indices, each with its Loc as
// the Loc carries it, independent of any Dir, sorted and joined.
func (u *unit) key(indices []int) string {
	texts := make([]string, len(indices))
	for k, i := range indices {
		texts[k] = u.decls[i].text(xsderr.Loc.String)
	}
	slices.Sort(texts)
	return strings.Join(texts, "\x00")
}

// definer returns the slice index of the target that defines the document
// uri for the unit at index self, whose imports are imports: the owner it
// imports uri from, or self.
func definer(self int, imports []imported, uri string) int {
	if owner, ok := ownerOf(imports, uri); ok {
		return owner
	}
	return self
}

// ownerOf returns the slice index of the target imports takes the document
// uri from, false when imports holds no such document.
func ownerOf(imports []imported, uri string) (int, bool) {
	i := slices.IndexFunc(imports, func(im imported) bool { return im.uri == uri })
	if i < 0 {
		return 0, false
	}
	return imports[i].owner, true
}

// link computes the imports of units[i] against the units before it, by the
// rule Generate's doc states, and renders its source again when it imports
// any document. The units before it are already linked.
func link(units []*unit, i int) error {
	u := units[i]
	var imports []imported
	for _, uri := range u.docs() {
		set := u.set(uri)
		setKey, closureKey := u.key(set), u.key(u.closure(set))
		for o, owner := range units[:i] {
			if definer(o, owner.imports, uri) != o {
				continue
			}
			ownerSet := owner.set(uri)
			if len(ownerSet) == 0 || owner.key(ownerSet) != setKey || owner.key(owner.closure(ownerSet)) != closureKey {
				continue
			}
			imports = append(imports, imported{uri: uri, owner: o})
			break
		}
	}
	for {
		kept := make([]imported, 0, len(imports))
		for _, im := range imports {
			if u.closureHolds(units, i, imports, im) {
				kept = append(kept, im)
			}
		}
		if len(kept) == len(imports) {
			break
		}
		imports = kept
	}
	if len(imports) == 0 {
		return nil
	}
	u.imports = imports
	src, err := u.render(units)
	if err != nil {
		return err
	}
	u.src = src
	return nil
}

// closureHolds reports whether the unit u at index self, importing imports,
// takes every document its closure of im's document reaches from the package
// that im's owner takes it from (Generate's doc).
func (u *unit) closureHolds(units []*unit, self int, imports []imported, im imported) bool {
	owner := units[im.owner]
	var reached []string
	for _, i := range u.closure(u.set(im.uri)) {
		if loc := u.decls[i].loc; loc != nil && !slices.Contains(reached, loc.URI) {
			reached = append(reached, loc.URI)
		}
	}
	for _, uri := range reached {
		if definer(self, imports, uri) != definer(im.owner, owner.imports, uri) {
			return false
		}
	}
	return true
}

// render returns u's gofmt-formatted source: every declaration as emitted
// when u imports nothing, and otherwise, units being every target's unit,
// each declaration of an imported document an alias of its owner's type, and
// a declaration attributed to no document only when one of u's own
// definitions reaches it.
func (u *unit) render(units []*unit) ([]byte, error) {
	aliased := make([]bool, len(u.decls))
	for i, d := range u.decls {
		if d.loc != nil {
			_, aliased[i] = ownerOf(u.imports, d.loc.URI)
		}
	}
	kept := make([]bool, len(u.decls))
	for i, d := range u.decls {
		kept[i] = len(u.imports) == 0 || d.loc != nil
	}
	if len(u.imports) > 0 {
		var own []int
		for i, d := range u.decls {
			if d.loc != nil && !aliased[i] {
				own = append(own, i)
			}
		}
		for next := 0; next < len(own); next++ {
			for _, j := range u.refs[own[next]].refs {
				if !kept[j] {
					kept[j] = true
					own = append(own, j)
				}
			}
		}
	}
	names := u.importNames(units, kept)
	usesBig := false
	for i := range u.decls {
		usesBig = usesBig || kept[i] && !aliased[i] && u.refs[i].usesBig
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s\n\npackage %s\n", generatedHeader, u.pkg)
	var specs []string
	if usesBig {
		specs = append(specs, `"math/big"`)
	}
	for k, o := range u.owners() {
		specs = append(specs, names[k]+" "+strconv.Quote(units[o].path))
	}
	switch len(specs) {
	case 0:
	case 1:
		fmt.Fprintf(&b, "\nimport %s\n", specs[0])
	default:
		fmt.Fprintf(&b, "\nimport (\n\t%s\n)\n", strings.Join(specs, "\n\t"))
	}
	renderLoc := func(loc xsderr.Loc) string { return headerLoc(u.base, loc) }
	for i, d := range u.decls {
		if !kept[i] {
			continue
		}
		b.WriteString("\n")
		if aliased[i] {
			owner, _ := ownerOf(u.imports, d.loc.URI)
			k := slices.Index(u.owners(), owner)
			fmt.Fprintf(&b, "%stype %s = %s.%s\n", d.comment(renderLoc), d.name, names[k], d.name)
			continue
		}
		b.WriteString(d.text(renderLoc))
	}
	src, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting the emitted source: %w", err)
	}
	return src, nil
}

// owners returns the slice indices of the targets u imports from, each once,
// ascending.
func (u *unit) owners() []int {
	var owners []int
	for _, im := range u.imports {
		if !slices.Contains(owners, im.owner) {
			owners = append(owners, im.owner)
		}
	}
	slices.Sort(owners)
	return owners
}

// importNames returns the name u's file imports each of its owners under, in
// owners' order: the owner's package name unless it is taken, and otherwise
// that name followed by the smallest numeric suffix from 2 that is not. A
// name is taken when it is a Go predeclared identifier, big, the name of one
// of u's declarations kept marks, an alias's included, or the name given to an
// earlier owner.
func (u *unit) importNames(units []*unit, kept []bool) []string {
	owners := u.owners()
	if len(owners) == 0 {
		return nil
	}
	// taken is a lookup table only and never iterated (STYLE D2).
	taken := map[string]bool{"big": true}
	for _, name := range types.Universe.Names() {
		taken[name] = true
	}
	for i, d := range u.decls {
		if kept[i] {
			taken[d.name] = true
		}
	}
	names := make([]string, len(owners))
	for k, o := range owners {
		pkg := units[o].pkg
		name := pkg
		for n := 2; taken[name]; n++ {
			name = pkg + strconv.Itoa(n)
		}
		taken[name] = true
		names[k] = name
	}
	return names
}
