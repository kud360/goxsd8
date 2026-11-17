package codeccensus

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/codegen"
	"github.com/kud360/goxsd8/conformance"
	"github.com/kud360/goxsd8/internal/goname"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/tools/internal/codeccensus/assess"
	"github.com/kud360/goxsd8/tools/internal/suiteroot"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// driverModule is the temporary module's path. It lies under this module's
// tools directory, the parent of the tools/internal holding package assess, so
// that Go's internal rule lets the drivers import that package, and no
// directory of this module's tree holds it, so that no import is ambiguous.
const driverModule = "github.com/kud360/goxsd8/tools/internal/codeccensus/driver"

// batchSize bounds the generated packages one driver links: 3000 were measured
// to build in one binary at grounding (#3292).
const batchSize = 2500

// A Population is what [Walk] counts before any driver runs: the suite root
// as the caller named it, the instance cases it considered, those whose
// schema generates, and of those the cases that decode into the placeholder
// and, of them, the ones whose document element names a top-level
// declaration with {abstract} false, as the package doc's "# Population"
// states each.
type Population struct {
	Root                    string
	Considered, Generates   int
	Placeholder, Unresolved int
}

// Walk runs the census over the suite at root up to its drivers' summaries:
// it reads the catalog, generates and resolves each group, builds and runs
// one driver per batch, and passes each driver's summary to merge, in batch
// order. When root holds no suite index Walk writes the corpus-absent notice,
// naming tool, to stdout and returns present false and the write's error.
func Walk(stdout io.Writer, tool, root string, merge func(assess.Summary)) (pop Population, present bool, err error) {
	index := filepath.Join(root, "suite.xml")
	if _, serr := os.Stat(index); errors.Is(serr, fs.ErrNotExist) {
		return Population{}, false, suiteroot.PrintAbsent(stdout, tool, "no suite index at "+index, "census")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Population{}, true, fmt.Errorf("making the suite root %s absolute: %w", root, err)
	}
	entries, err := conformance.Catalog(abs)
	if err != nil {
		return Population{}, true, fmt.Errorf("reading the catalog: %w", err)
	}
	module, err := moduleRoot(tool)
	if err != nil {
		return Population{}, true, err
	}
	tmp, err := os.MkdirTemp("", tool+"-")
	if err != nil {
		return Population{}, true, fmt.Errorf("creating the driver module: %w", err)
	}
	defer func() {
		if rerr := os.RemoveAll(tmp); rerr != nil {
			err = errors.Join(err, fmt.Errorf("removing the driver module %s: %w", tmp, rerr))
		}
	}()
	if err := writeModule(tmp, module); err != nil {
		return Population{}, true, err
	}
	pop, batches, err := plan(abs, root, instanceCases(entries), tmp)
	if err != nil {
		return Population{}, true, err
	}
	for i, b := range batches {
		sum, err := drive(tmp, i, b)
		if err != nil {
			return Population{}, true, err
		}
		merge(sum)
	}
	return pop, true, nil
}

// instanceCases returns the entries the population holds, in catalog order:
// each instanceTest entry not withheld.
func instanceCases(entries []conformance.CatalogEntry) []conformance.CatalogEntry {
	var cases []conformance.CatalogEntry
	for _, e := range entries {
		if e.Withheld || !isInstanceID(e.ID) {
			continue
		}
		cases = append(cases, e)
	}
	return cases
}

// isInstanceID reports whether id, `<testSet>/<testGroup>/<kind>/<test-name>`,
// names an instanceTest.
func isInstanceID(id string) bool {
	segments := strings.Split(id, "/")
	return len(segments) >= 4 && segments[len(segments)-2] == "instance"
}

// moduleRoot returns the directory of this module's go.mod, as `go env GOMOD`
// names it from the working directory, and its go directive's version; tool
// names the command in the error for a working directory outside it.
func moduleRoot(tool string) (module, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return module{}, fmt.Errorf("running go env GOMOD: %w", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		return module{}, errors.New(tool + " runs inside the github.com/kud360/goxsd8 module, and the working directory lies in no module")
	}
	data, err := os.ReadFile(gomod)
	if err != nil {
		return module{}, fmt.Errorf("reading %s: %w", gomod, err)
	}
	m := module{dir: filepath.Dir(gomod)}
	var path string
	for line := range strings.Lines(string(data)) {
		if p, ok := strings.CutPrefix(line, "module "); ok {
			path = strings.TrimSpace(p)
		}
		if v, ok := strings.CutPrefix(line, "go "); ok {
			m.goVersion = strings.TrimSpace(v)
		}
	}
	if path != "github.com/kud360/goxsd8" {
		return module{}, fmt.Errorf("%s runs inside the github.com/kud360/goxsd8 module, and %s declares %q", tool, gomod, path)
	}
	return m, nil
}

// module is this module's root directory and its go directive's version.
type module struct {
	dir, goVersion string
}

// writeModule writes the driver module's go.mod into dir, requiring this
// module by a replace directive on m's directory, and copies m's go.sum
// beside it.
func writeModule(dir string, m module) error {
	gomod := fmt.Sprintf("module %s\n\ngo %s\n\nrequire github.com/kud360/goxsd8 v0.0.0\n\nreplace github.com/kud360/goxsd8 => %s\n",
		driverModule, m.goVersion, m.dir)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		return fmt.Errorf("writing the driver module's go.mod: %w", err)
	}
	sum, err := os.ReadFile(filepath.Join(m.dir, "go.sum"))
	if err != nil {
		return fmt.Errorf("reading the module's go.sum: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644); err != nil {
		return fmt.Errorf("writing the driver module's go.sum: %w", err)
	}
	return nil
}

// batch is one driver's share of the census: the request it reads, the
// generated packages it imports, and the decoders it compiles, each a
// qualified type, `p<N>.<Name>`, at the index a case's assess.Case.Decoder
// names.
type batch struct {
	req      assess.Request
	packages []genPackage
	decoders []string
}

// genPackage is one generated package a driver imports: its name, p<N>, its
// generated file's sealed interfaces, in file order, and nillable, whether
// the census wrote its nillableOptions (writeNillableOptions).
type genPackage struct {
	name     string
	sealed   []sealedDecl
	nillable bool
}

// bound reports whether p's decoders take an option list: p declares a
// sealed interface or the census bound a Nillable instantiation of it.
func (p genPackage) bound() bool { return len(p.sealed) > 0 || p.nillable }

// sealedDecl is one interface type a generated file declares, by name, and
// its branches, the names of the types the file gives its marker method, in
// file order, and decimal, whether it is the Value interface of codegen's
// PrecisionDecimal, its branches then that struct's four arms.
type sealedDecl struct {
	name     string
	branches []string
	decimal  bool
}

// decimalProvenance is the provenance a header records for the types codegen
// emits for xs:precisionDecimal, its PrecisionDecimal, the Value interface
// and its arms (codegen's package doc, "Naming").
const decimalProvenance = " is generated from datatype {http://www.w3.org/2001/XMLSchema}precisionDecimal."

// plan generates each group's package into dir, resolves each case's type,
// and returns the population with the batches that drive the rest. abs is
// the suite root, absolute, and root as the caller named it.
func plan(abs, root string, cases []conformance.CatalogEntry, dir string) (Population, []batch, error) {
	pop := Population{Root: root, Considered: len(cases)}
	backend := strict.New()
	var batches []batch
	var cur *batch
	// groupOf maps a schema-document list, joined, to its group's position in
	// groups, for lookup only; nothing ranges over it (STYLE D2).
	groupOf := map[string]int{}
	var groups []groupCases
	for _, c := range cases {
		if len(c.SchemaDocs) == 0 {
			continue
		}
		key := strings.Join(c.SchemaDocs, "\x00")
		i, seen := groupOf[key]
		if !seen {
			i = len(groups)
			groupOf[key] = i
			groups = append(groups, groupCases{schema: c.SchemaDocs})
		}
		groups[i].cases = append(groups[i].cases, c)
	}
	for n, g := range groups {
		// A schema that does not assemble or generate is the census's data,
		// its cases counted as not generating, never an operational error:
		// the suite is full of schemas that are invalid on purpose.
		schema, err := assess.Assemble(abs, g.schema, backend)
		if err != nil {
			continue
		}
		pkg := "p" + strconv.Itoa(n)
		pkgDir := filepath.Join(dir, pkg)
		if errs := codegen.Generate([]codegen.Target{{Schema: schema, Dir: pkgDir, Package: pkg, Name: pkg}}); errs != nil {
			continue
		}
		gen, err := parseGenerated(filepath.Join(pkgDir, "xsd_gen.go"))
		if err != nil {
			return Population{}, nil, err
		}
		file := gen.file
		types := generatedTypes(file)
		family := nillableFamilyOf(gen)
		if cur == nil || len(cur.packages) == batchSize {
			batches = append(batches, batch{req: assess.Request{Root: abs}})
			cur = &batches[len(batches)-1]
		}
		ag := assess.Group{Schema: g.schema}
		used := false
		for _, c := range g.cases {
			pop.Generates++
			inst := c.Docs[0]
			name, err := documentElement(filepath.Join(abs, filepath.FromSlash(inst)))
			if err != nil {
				return Population{}, nil, err
			}
			typ, decl, ok := resolveCase(schema, types, name, pkgDir)
			qualified := pkg + "." + typ
			if ok {
				qualified, ok = family.root(schema, types, name, typ, pkg)
			}
			if !ok {
				pop.Placeholder++
				if decl {
					pop.Unresolved++
				}
				ag.Cases = append(ag.Cases, assess.Case{ID: c.ID, Instance: inst, Decoder: -1})
				continue
			}
			used = true
			ag.Cases = append(ag.Cases, assess.Case{ID: c.ID, Instance: inst, Decoder: cur.decoder(qualified)})
		}
		cur.req.Groups = append(cur.req.Groups, ag)
		if !used {
			continue
		}
		nillable := len(family.args) > 0
		if nillable {
			if err := writeNillableOptions(pkgDir, pkg, *family, file.Imports); err != nil {
				return Population{}, nil, err
			}
		}
		cur.packages = append(cur.packages, genPackage{name: pkg, sealed: sealedInterfaces(file), nillable: nillable})
	}
	return pop, batches, nil
}

// groupCases is one schema-document list and the cases of every test group
// declaring it, in catalog order.
type groupCases struct {
	schema []string
	cases  []conformance.CatalogEntry
}

// decoder returns the index of the decoder into typ, a qualified type,
// adding it when b has none.
func (b *batch) decoder(typ string) int {
	if i := slices.Index(b.decoders, typ); i >= 0 {
		return i
	}
	b.decoders = append(b.decoders, typ)
	return len(b.decoders) - 1
}

// resolveCase returns the name of the type ownType finds for the top-level
// element declaration of schema the document element name names, ok false
// when there is none; decl reports, for ok false, that the declaration exists
// with {abstract} false, so that only ownType failed to find a single type.
// A zero name, a document element that did not read, has no declaration.
func resolveCase(schema *xsd.Schema, types []typeDecl, name xsd.QName, dir string) (typ string, decl, ok bool) {
	if name == (xsd.QName{}) {
		return "", false, false
	}
	d, found := schema.Element(name)
	if !found || d.Abstract() {
		return "", false, false
	}
	typ, ok = ownType(types, d, dir)
	return typ, true, ok
}

// documentElement returns the expanded name of the document element of the
// instance at path, or the zero QName when the document is not well-formed
// before it.
func documentElement(path string) (xsd.QName, error) {
	f, err := os.Open(path)
	if err != nil {
		return xsd.QName{}, fmt.Errorf("reading instance %s: %w", path, err)
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot change what was read
	r := xmltree.NewReader(path, f)
	for {
		tok, err := r.Token()
		if err != nil {
			return xsd.QName{}, nil
		}
		if start, ok := tok.(*xmltree.StartElement); ok {
			return xsd.QName{Space: start.Name().Space(), Local: start.Name().Local()}, nil
		}
	}
}

// typeDecl is one type declaration of a generated file: its name, the first
// line of its doc comment, the header codegen's package doc "Naming" states,
// and holds, the type whose attribute fields it holds where its second line
// is the one that section states for a nilled element's attributes struct, ""
// otherwise.
type typeDecl struct {
	name, header, holds string
}

// The text around the type's name on the second header line of a nilled
// element's attributes struct (codegen's package doc, "Naming").
const (
	holdsPrefix = "// It holds the attribute fields of "
	holdsSuffix = ", which a nilled element keeps."
)

// generatedFile is a generated file as parseGenerated read it: its syntax
// tree, the file set positioning it, and its source.
type generatedFile struct {
	file *ast.File
	fset *token.FileSet
	src  []byte
}

// parseGenerated parses the generated file at path, its comments included.
func parseGenerated(path string) (generatedFile, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return generatedFile{}, fmt.Errorf("reading generated file %s: %w", path, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return generatedFile{}, fmt.Errorf("parsing generated file %s: %w", path, err)
	}
	return generatedFile{file: file, fset: fset, src: src}, nil
}

// generatedTypes returns the type declarations of the generated file, in
// file order.
func generatedTypes(file *ast.File) []typeDecl {
	var types []typeDecl
	for _, d := range file.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE || gen.Doc == nil {
			continue
		}
		var holds string
		if len(gen.Doc.List) > 1 {
			if of, ok := strings.CutPrefix(gen.Doc.List[1].Text, holdsPrefix); ok {
				holds, _ = strings.CutSuffix(of, holdsSuffix)
			}
		}
		for _, spec := range gen.Specs {
			types = append(types, typeDecl{name: spec.(*ast.TypeSpec).Name.Name, header: gen.Doc.List[0].Text, holds: holds})
		}
	}
	return types
}

// sealedInterfaces returns each interface type the generated file declares
// with no type parameters, in file order, with its branches. A generic
// interface, such as codegen's Nillable, is no type a WithBranches can name
// uninstantiated: nillableFamilyOf finds it, and writeNillableOptions binds
// each instantiation the census decodes through. The branches are each type B the file
// gives a method of a single value receiver of type B, named "is" plus the interface's
// name, with no parameters or results, the marker codegen's package doc "# Complex
// types" states, in file order; for a union's interface, file order is its variants'
// basic-member order (codegen's package doc, "# Simple types"), the order WithBranches
// takes a union's variants of one Go type in. An interface whose header records
// decimalProvenance is PrecisionDecimal's Value interface, and its branches, in file
// order, are its arms in the order codegen's package doc "# Simple types" states they
// are emitted: numerical, positive infinity, negative infinity, not a number.
func sealedInterfaces(file *ast.File) []sealedDecl {
	var sealed []sealedDecl
	for _, d := range file.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts := spec.(*ast.TypeSpec)
			if _, isInterface := ts.Type.(*ast.InterfaceType); isInterface && ts.TypeParams == nil {
				decimal := gen.Doc != nil && gen.Doc.List[0].Text == "// "+ts.Name.Name+decimalProvenance
				sealed = append(sealed, sealedDecl{name: ts.Name.Name, decimal: decimal})
			}
		}
	}
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 || fn.Type.Params.NumFields() != 0 || fn.Type.Results.NumFields() != 0 {
			continue
		}
		recv, ok := fn.Recv.List[0].Type.(*ast.Ident)
		if !ok {
			continue
		}
		i := slices.IndexFunc(sealed, func(s sealedDecl) bool { return fn.Name.Name == "is"+s.name })
		if i >= 0 {
			sealed[i].branches = append(sealed[i].branches, recv.Name)
		}
	}
	return sealed
}

// ownProvenances are the texts a header renders the declaration a top-level
// element's own type is generated from as, before the element's name: plain,
// and, for a sealed interface over its governing types, through its {type
// table} or its {type definition} (codegen's package doc, "Element
// declarations").
var ownProvenances = []string{"element ", "the {type table} of element ", "the {type definition} of element "}

// ownType returns the name of the type codegen emitted for the top-level
// element declaration decl into dir, among types, ok false unless exactly one
// type matches. A type matches when its name is decl's local name's
// goname.Identifier, alone or with a numeric suffix from 2, codegen's package
// doc "Naming" rule for a type named from an element, and its header records
// decl, by its expanded name and its Loc rendered as that section states for
// dir, as one of ownProvenances. Every other type a header records decl for is
// named by a path or a branch of names that extends the identifier with
// further words, each of which opens with an upper-case letter, so no other
// name has that shape.
func ownType(types []typeDecl, decl xsd.ElementDeclaration, dir string) (string, bool) {
	base := goname.Identifier(decl.Name().Local)
	loc := headerLoc(dir, decl.Loc())
	var found []string
	for _, t := range types {
		if !namedFrom(t.name, base) {
			continue
		}
		for _, p := range ownProvenances {
			if t.header == "// "+t.name+" is generated from "+p+decl.Name().String()+" declared at "+loc+"." {
				found = append(found, t.name)
			}
		}
	}
	if len(found) != 1 {
		return "", false
	}
	return found[0], true
}

// namedFrom reports whether name is base, or base followed by a numeric
// suffix goname.Namer gives a collision: a decimal from 2, with no leading
// zero.
func namedFrom(name, base string) bool {
	suffix, ok := strings.CutPrefix(name, base)
	if !ok {
		return false
	}
	if suffix == "" {
		return true
	}
	n, err := strconv.Atoi(suffix)
	return err == nil && n >= 2 && strconv.Itoa(n) == suffix
}

// headerLoc renders loc as codegen's package doc "Naming" states a header
// renders it for a target whose Dir is dir.
func headerLoc(dir string, loc xsderr.Loc) string {
	if !filepath.IsAbs(loc.URI) {
		return loc.String()
	}
	base, err := filepath.Abs(dir)
	if err != nil {
		return loc.String()
	}
	rel, err := filepath.Rel(base, loc.URI)
	if err != nil {
		return loc.String()
	}
	loc.URI = filepath.ToSlash(rel)
	return loc.String()
}

// drive writes batch b's driver into the module at dir as package b<n>,
// builds it, runs it on b's request, and returns its summary.
func drive(dir string, n int, b batch) (assess.Summary, error) {
	name := "b" + strconv.Itoa(n)
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		return assess.Summary{}, fmt.Errorf("creating driver %s: %w", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name, "main.go"), driverSource(b), 0o644); err != nil {
		return assess.Summary{}, fmt.Errorf("writing driver %s: %w", name, err)
	}
	bin := filepath.Join(dir, name, name)
	build := exec.Command("go", "build", "-o", bin, "./"+name)
	build.Dir = dir
	build.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOWORK=off")
	if out, err := build.CombinedOutput(); err != nil {
		return assess.Summary{}, fmt.Errorf("building driver %s: %w\n%s", name, err, out)
	}
	req, err := json.Marshal(b.req)
	if err != nil {
		return assess.Summary{}, fmt.Errorf("encoding driver %s's request: %w", name, err)
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin)
	cmd.Stdin = bytes.NewReader(req)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return assess.Summary{}, fmt.Errorf("running driver %s: %w\n%s", name, err, stderr.Bytes())
	}
	var sum assess.Summary
	if err := json.Unmarshal(stdout.Bytes(), &sum); err != nil {
		return assess.Summary{}, fmt.Errorf("reading driver %s's summary: %w", name, err)
	}
	return sum, nil
}

// driverSource returns the main package of b's driver: an import of each of
// b's packages; for each package declaring a sealed interface, the option
// list p<N>Branches, one option per interface, each branch's zero value as
// *new(B), so that a branch that is no struct compiles too: a
// codec.WithPrecisionDecimalArms of its four arms in order for a
// PrecisionDecimal's Value interface, and a codec.WithBranches for every
// other; and a decoder per qualified type, in b's order, bound to its
// package's list.
func driverSource(b batch) []byte {
	var s strings.Builder
	s.WriteString("// Code generated by the codec census; DO NOT EDIT.\n\npackage main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\n")
	if slices.ContainsFunc(b.packages, genPackage.bound) {
		s.WriteString("\t\"github.com/kud360/goxsd8/codec\"\n")
	}
	s.WriteString("\t\"github.com/kud360/goxsd8/tools/internal/codeccensus/assess\"\n")
	for _, p := range b.packages {
		fmt.Fprintf(&s, "\t%s %q\n", p.name, path.Join(driverModule, p.name))
	}
	s.WriteString(")\n")
	for _, p := range b.packages {
		if !p.bound() {
			continue
		}
		head, tail := "[]codec.Option{\n", "}\n"
		if p.nillable {
			head, tail = "append([]codec.Option{\n", "}, "+p.name+"."+nillableOptions+"...)\n"
		}
		fmt.Fprintf(&s, "\nvar %sBranches = %s", p.name, head)
		for _, sd := range p.sealed {
			option := "WithBranches"
			if sd.decimal {
				option = "WithPrecisionDecimalArms"
			}
			fmt.Fprintf(&s, "\tcodec.%s[%s.%s](", option, p.name, sd.name)
			for i, br := range sd.branches {
				if i > 0 {
					s.WriteString(", ")
				}
				fmt.Fprintf(&s, "*new(%s.%s)", p.name, br)
			}
			s.WriteString("),\n")
		}
		s.WriteString(tail)
	}
	s.WriteString("\nvar decoders = []assess.Decoder{\n")
	for _, d := range b.decoders {
		pkg, _, _ := strings.Cut(d, ".")
		bound := ""
		if i := slices.IndexFunc(b.packages, func(p genPackage) bool { return p.name == pkg }); i >= 0 && b.packages[i].bound() {
			bound = pkg + "Branches..."
		}
		fmt.Fprintf(&s, "\tassess.DecoderFor[%s](%s),\n", d, bound)
	}
	s.WriteString("}\n\nfunc main() {\n\tif err := assess.Drive(os.Stdin, os.Stdout, decoders); err != nil {\n" +
		"\t\tfmt.Fprintln(os.Stderr, err)\n\t\tos.Exit(1)\n\t}\n}\n")
	return []byte(s.String())
}

// Ranked returns the tallies largest first, then by the construct's String,
// the order each command prints its decline table in.
func Ranked(tallies []assess.Tally) []assess.Tally {
	r := slices.Clone(tallies)
	slices.SortStableFunc(r, func(a, b assess.Tally) int {
		if c := cmp.Compare(len(b.Cases), len(a.Cases)); c != 0 {
			return c
		}
		return strings.Compare(a.Construct, b.Construct)
	})
	return r
}

// MergeTallies adds src, one driver's decline table, to dst and returns it:
// a construct's tallies from two drivers merge by its String, the key
// assess.Drive's doc states across processes, every driver linking the one
// codec, and their case lists concatenate.
func MergeTallies(dst, src []assess.Tally) []assess.Tally {
	for _, t := range src {
		i := slices.IndexFunc(dst, func(u assess.Tally) bool { return u.Construct == t.Construct })
		if i < 0 {
			dst = append(dst, assess.Tally{Construct: t.Construct})
			i = len(dst) - 1
		}
		dst[i].Cases = append(dst[i].Cases, t.Cases...)
	}
	return dst
}

// PrintCases renders one -paths list to w: its heading with the case count,
// then each case ID in byte order. w is a bufio.Writer, which keeps its first
// write error for the caller's Flush to return.
func PrintCases(w *bufio.Writer, heading string, ids []string) {
	_, _ = fmt.Fprintf(w, "%s (%d):\n", heading, len(ids))
	for _, id := range slices.Sorted(slices.Values(ids)) {
		_, _ = fmt.Fprintf(w, "  %s\n", id)
	}
}
