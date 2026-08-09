// Command surface prints the module's exported identifier surface, and,
// given -base, diffs it against the surface at another git ref.
//
// STYLE T5 requires every new exported identifier to carry a justification,
// so the arbiter, the warden and the steward all need the same answer: what
// did this change add to or remove from the surface. Computing it is
// mechanical, so it is a tool (PRINCIPLES 27) rather than a `go doc` read by
// eye.
//
// Usage:
//
//	go run ./tools/surface                    # print the current surface
//	go run ./tools/surface -base origin/main   # diff current against a ref
//
// With no flags it walks the module's packages and prints one line per
// exported identifier, sorted and deterministic:
//
//	<import path> func|method|type|const|var|field <name>
//
// Methods print their receiver type, and struct fields print
// "<TypeName>.<FieldName>". `conformance`, `tools/` and `internal/` are
// skipped: none is published surface.
//
// With -base <ref>, surface also computes the same listing at <ref> and
// prints only what changed: lines added (prefixed "+") and removed
// (prefixed "-"), followed by a one-line summary ("surface: +3 -1" or
// "surface: unchanged"). The ref's tree is materialized in a temporary
// detached git worktree (`git worktree add --detach`), read, and removed
// again (`git worktree remove --force`) so the caller's checkout and
// branch are never touched — docs/WORKFLOW.md's "one writer per checkout"
// forbids stashing or checking out in place.
//
// Exit codes: 0 on success (with or without a diff), 2 on an operational
// error (an unparseable go.mod, an unwalkable directory, a failed git
// worktree command, or the like).
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	base := flag.String("base", "", "git ref to diff the current exported surface against")
	flag.Parse()

	if err := run(*base); err != nil {
		fmt.Fprintf(os.Stderr, "surface: %v\n", err)
		os.Exit(2)
	}
}

// run is main's testable body: compute the current surface, and either
// print it (base == "") or diff it against base and print the diff.
func run(base string) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	modulePath, err := readModulePath(root)
	if err != nil {
		return err
	}
	cur, err := Surface(root, modulePath)
	if err != nil {
		return fmt.Errorf("computing current surface: %w", err)
	}

	if base == "" {
		for _, line := range cur {
			fmt.Println(line)
		}
		return nil
	}
	return runDiff(root, base, cur)
}

// runDiff materializes base in a temporary detached worktree, computes its
// surface there, diffs it against cur, and prints the result. The worktree
// is always removed before returning, including on an error path.
func runDiff(root, base string, cur []string) error {
	wtDir, cleanup, err := worktreeAt(root, base)
	if err != nil {
		return err
	}
	defer cleanup()

	baseModulePath, err := readModulePath(wtDir)
	if err != nil {
		return err
	}
	baseSurface, err := Surface(wtDir, baseModulePath)
	if err != nil {
		return fmt.Errorf("computing surface at %s: %w", base, err)
	}

	added, removed := Diff(baseSurface, cur)
	for _, line := range removed {
		fmt.Println("- " + line)
	}
	for _, line := range added {
		fmt.Println("+ " + line)
	}
	fmt.Println(summary(added, removed))
	return nil
}

// summary renders the one-line diff summary runDiff prints after the
// added/removed lines: how many lines moved each direction, or
// "unchanged" when the two surfaces are identical.
func summary(added, removed []string) string {
	if len(added) == 0 && len(removed) == 0 {
		return "surface: unchanged"
	}
	return fmt.Sprintf("surface: +%d -%d", len(added), len(removed))
}

// Surface walks the Go source tree rooted at dir and returns the sorted,
// deduplicated exported-identifier surface it finds there: one line per
// func, method, type, const, var, and exported struct field on an exported
// type. modulePath prefixes every import path (the root package gets
// modulePath itself; a subdirectory gets modulePath + "/" + its
// slash-separated path relative to dir). It performs no git and no process
// calls, so a test can point it at a fixture tree built with t.TempDir()
// and get the same deterministic answer a real module checkout would.
func Surface(dir, modulePath string) ([]string, error) {
	seen := make(map[string]bool)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("relativizing %s to %s: %w", path, dir, err)
		}
		if skipDir(rel) {
			return filepath.SkipDir
		}
		lines, err := packageSurface(path, importPath(modulePath, rel))
		if err != nil {
			return err
		}
		for _, line := range lines {
			seen[line] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", dir, err)
	}

	out := make([]string, 0, len(seen))
	for line := range seen {
		out = append(out, line)
	}
	sort.Strings(out)
	return out, nil
}

// Diff compares two surface listings, such as those Surface produces, and
// reports what changed between them: lines present in cur but not base
// ("added") and lines present in base but not cur ("removed"), each
// deduplicated and sorted. It performs no git and no process calls, so a
// test can hand it fixture slices directly regardless of their input
// order.
func Diff(base, cur []string) (added, removed []string) {
	baseSet := toSet(base)
	curSet := toSet(cur)
	return setDiff(curSet, baseSet), setDiff(baseSet, curSet)
}

// toSet builds a lookup index (STYLE D2's permitted use of a map) out of a
// surface listing, for setDiff to test membership against.
func toSet(lines []string) map[string]bool {
	set := make(map[string]bool, len(lines))
	for _, line := range lines {
		set[line] = true
	}
	return set
}

// setDiff returns the sorted lines of a that are absent from b.
func setDiff(a, b map[string]bool) []string {
	var out []string
	for line := range a {
		if !b[line] {
			out = append(out, line)
		}
	}
	sort.Strings(out)
	return out
}

// skipDir reports whether the directory named by rel — a dir path already
// relativized to the tree root Surface is walking — is pruned from the walk.
// `conformance` and `tools` are repo infrastructure rather than library, so
// neither is published surface (docs/ARCHITECTURE.md's two tiers); the rest
// are never surface at all: internal packages, vendored or fixture trees, and
// the hidden and underscore-prefixed names `go build` itself ignores.
func skipDir(rel string) bool {
	if rel == "." {
		return false
	}
	name := filepath.Base(rel)
	switch name {
	case "conformance", "tools", "internal", "testdata", "vendor":
		return true
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return true
	}
	return false
}

// importPath renders the import path of the package living in the
// directory rel is relative to modulePath's module root.
func importPath(modulePath, rel string) string {
	if rel == "." {
		return modulePath
	}
	return modulePath + "/" + filepath.ToSlash(rel)
}

// packageSurface parses every non-test .go file directly inside dir (not
// its subdirectories — those are separate packages Surface's walk visits
// on their own) and returns the exported-identifier lines they declare,
// labeled with importPath.
func packageSurface(dir, importPath string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading dir %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	var lines []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		lines = append(lines, fileSurface(importPath, file)...)
	}
	return lines, nil
}

// fileSurface returns the exported-identifier lines declared at the top
// level of a single parsed file.
func fileSurface(importPath string, file *ast.File) []string {
	var lines []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			lines = append(lines, funcSurface(importPath, d)...)
		case *ast.GenDecl:
			lines = append(lines, genDeclSurface(importPath, d)...)
		}
	}
	return lines
}

// funcSurface returns a func's or method's surface line, or nil when its
// name is unexported.
func funcSurface(importPath string, d *ast.FuncDecl) []string {
	if !ast.IsExported(d.Name.Name) {
		return nil
	}
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return []string{fmt.Sprintf("%s func %s", importPath, d.Name.Name)}
	}
	recv := receiverType(d.Recv.List[0].Type)
	return []string{fmt.Sprintf("%s method (%s) %s", importPath, recv, d.Name.Name)}
}

// receiverType renders a method receiver's type expression back to the
// name funcSurface's output line uses: "Foo" or "*Foo", stripping any
// generic type parameters.
func receiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return "*" + receiverType(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverType(t.X)
	case *ast.IndexListExpr:
		return receiverType(t.X)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// genDeclSurface dispatches a top-level `type`, `const`, or `var` block to
// its surface extractor; any other token (e.g. `import`) contributes no
// surface.
func genDeclSurface(importPath string, d *ast.GenDecl) []string {
	switch d.Tok {
	case token.TYPE:
		return typeSurface(importPath, d)
	case token.CONST:
		return valueSurface(importPath, "const", d)
	case token.VAR:
		return valueSurface(importPath, "var", d)
	default:
		return nil
	}
}

// typeSurface returns an exported type's own surface line plus, for a
// struct type, one line per exported field (STYLE T5's "exported struct
// fields on exported types").
func typeSurface(importPath string, d *ast.GenDecl) []string {
	var lines []string
	for _, spec := range d.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok || !ast.IsExported(ts.Name.Name) {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s type %s", importPath, ts.Name.Name))

		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			continue
		}
		for _, field := range st.Fields.List {
			lines = append(lines, fieldSurface(importPath, ts.Name.Name, field)...)
		}
	}
	return lines
}

// fieldSurface returns a struct field's surface line(s): one per exported
// name in a named field group, or the embedded type's own name for an
// embedded (anonymous) field.
func fieldSurface(importPath, typeName string, field *ast.Field) []string {
	var lines []string
	for _, n := range field.Names {
		if !ast.IsExported(n.Name) {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s field %s.%s", importPath, typeName, n.Name))
	}
	if len(field.Names) > 0 {
		return lines
	}
	name := embeddedName(field.Type)
	if name == "" || !ast.IsExported(name) {
		return lines
	}
	return append(lines, fmt.Sprintf("%s field %s.%s", importPath, typeName, name))
}

// embeddedName extracts the type name an anonymous struct field embeds,
// unwrapping a leading pointer or package-qualified selector.
func embeddedName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return embeddedName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

// valueSurface returns one line per exported name in a `const` or `var`
// block, labeled with kind ("const" or "var").
func valueSurface(importPath, kind string, d *ast.GenDecl) []string {
	var lines []string
	for _, spec := range d.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, n := range vs.Names {
			if !ast.IsExported(n.Name) {
				continue
			}
			lines = append(lines, fmt.Sprintf("%s %s %s", importPath, kind, n.Name))
		}
	}
	return lines
}

// moduleRoot walks up from the current working directory until it finds
// the directory containing go.mod, so the tool works the same way whether
// it is run from the module root or from a package directory beneath it
// (the same approach tools/rulecat uses for the same reason).
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from working directory")
		}
		dir = parent
	}
}

// readModulePath extracts the `module` directive from the go.mod file at
// root, without a full go.mod parser: a hand-rolled tool has no need for
// the replace/require directives a module graph resolver would.
func readModulePath(root string) (string, error) {
	path := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		mod, ok := strings.CutPrefix(strings.TrimSpace(line), "module ")
		if !ok {
			continue
		}
		return strings.TrimSpace(mod), nil
	}
	return "", fmt.Errorf("no module directive found in %s", path)
}

// worktreeAt materializes ref as a temporary detached git worktree of the
// repository rooted at repoRoot, returning its path and a cleanup func the
// caller must defer. It never touches repoRoot's own checked-out branch:
// `git worktree add --detach` leaves HEAD there untouched.
func worktreeAt(repoRoot, ref string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "goxsd8-surface-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir for %s worktree: %w", ref, err)
	}

	add := exec.Command("git", "worktree", "add", "--detach", dir, ref)
	add.Dir = repoRoot
	if out, err := add.CombinedOutput(); err != nil {
		_ = os.RemoveAll(dir) // best effort: add failed, nothing was checked out into dir
		return "", nil, fmt.Errorf("git worktree add --detach %s %s: %w\n%s", dir, ref, err, out)
	}

	cleanup := func() {
		remove := exec.Command("git", "worktree", "remove", "--force", dir)
		remove.Dir = repoRoot
		if out, err := remove.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "surface: removing temp worktree %s: %v\n%s\n", dir, err, out)
		}
	}
	return dir, cleanup, nil
}
