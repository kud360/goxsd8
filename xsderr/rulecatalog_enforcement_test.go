// This file enforces, module-wide, the doc.go claim that every [xsderr.Rule]
// constructed anywhere in goxsd8 is a rule the specs actually define. Rule is a
// bare string type, so the Go type system cannot carry that guarantee; a source
// scan can. The test is deliberately in the EXTERNAL test package so it reaches
// the catalog only through the exported IsValidRule — the same door a consumer
// has.
//
// It has two halves, and both are needed:
//
//   - POSITIVE: every Rule-typed constant declared in the module is in the
//     catalog (or is a sentinel). This is what "no invented rule IDs" means.
//   - NEGATIVE: no string literal sits in a Rule POSITION in non-test code —
//     a New/Wrap rule argument, an Error.Rule field, a return from a
//     Rule-returning function, or a Rule(...) conversion. Without this half the
//     positive half is trivially satisfiable by going back to bare literals,
//     which declare no constant at all and so would be scanned by nothing.
package xsderr_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// xsderrImportPath is the import path whose local name each scanned file is
// resolved against, so an aliased import (`e "…/xsderr"`) is followed correctly
// rather than assumed to be spelled "xsderr".
const xsderrImportPath = "github.com/kud360/goxsd8/xsderr"

// The vacuity floors. A scan that silently walks nothing passes every assertion
// it makes, so each half declares the minimum it must see. Both sit safely below
// the counts at the time of writing (287 New/Wrap call sites, 83 Rule-typed
// constants) so ordinary refactoring does not trip them, while a scan that has
// stopped finding files does.
const (
	minRuleCallSites = 250
	minRuleConstants = 65
)

// TestEveryRuleConstantIsInTheCatalog is the POSITIVE half: every Rule-typed
// constant declared anywhere in the module — test files included, since a
// typo'd fixture constant is just as wrong — satisfies IsValidRule.
func TestEveryRuleConstantIsInTheCatalog(t *testing.T) {
	root := moduleRoot(t)
	found := 0
	for _, f := range goFiles(t, root, true) {
		s := newFileScope(t, f)
		if s == nil {
			continue
		}
		for _, c := range s.ruleConstants() {
			found++
			if xsderr.IsValidRule(xsderr.Rule(c.value)) {
				continue
			}
			t.Errorf("%s: const %s = %q is not in the rule catalog and is not a sentinel; either the ID is a typo or the specs in docs/specs/md do not define it (regenerate catalog.go with `go tool rulecat` if the specs changed)",
				s.pos(c.pos), c.name, c.value)
		}
	}
	if found < minRuleConstants {
		t.Errorf("scanned only %d xsderr.Rule constants, want at least %d: the scan is probably not walking the module (root %s)", found, minRuleConstants, root)
	}
}

// TestNoRuleStringLiterals is the NEGATIVE half: in non-test code, a Rule
// position must hold a named constant, never a bare string literal. That is what
// keeps the positive half non-vacuous — and what makes every rule ID in the
// module carry a doc comment citing its spec section.
func TestNoRuleStringLiterals(t *testing.T) {
	root := moduleRoot(t)
	calls := 0
	for _, f := range goFiles(t, root, false) {
		s := newFileScope(t, f)
		if s == nil {
			continue
		}
		n, violations := s.ruleLiterals()
		calls += n
		for _, v := range violations {
			t.Errorf("%s: %s is a string literal; declare a named `xsderr.Rule` constant with a doc comment citing the spec rule and use that instead (STYLE E2)",
				s.pos(v.pos), v.what)
		}
	}
	if calls < minRuleCallSites {
		t.Errorf("scanned only %d xsderr.New/Wrap call sites, want at least %d: the scan is probably not walking the module (root %s)", calls, minRuleCallSites, root)
	}
}

// fileScope is one parsed file plus the name xsderr's exports go by inside it.
type fileScope struct {
	fset *token.FileSet
	file *ast.File
	// qual is the identifier qualifying xsderr's exports (usually "xsderr").
	// It is empty when the file IS package xsderr, which names them unqualified;
	// refers resolves both forms so the module's own error package is scanned
	// like every other.
	qual string
}

// newFileScope parses path and reports how it refers to xsderr, or nil when the
// file cannot see xsderr at all and so has nothing to scan.
func newFileScope(t *testing.T, path string) *fileScope {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Errorf("parsing %s: %v", path, err)
		return nil
	}
	s := &fileScope{fset: fset, file: f}
	if f.Name.Name == "xsderr" {
		return s
	}
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil || p != xsderrImportPath {
			continue
		}
		s.qual = "xsderr"
		if imp.Name != nil {
			s.qual = imp.Name.Name
		}
		return s
	}
	return nil
}

// pos renders a position as file:line, relative to nothing — the absolute path
// is what a reader needs to open the offending line.
func (s *fileScope) pos(p token.Pos) string {
	pos := s.fset.Position(p)
	return fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
}

// refers reports whether e names xsderr's exported name, in whichever form this
// file spells it: the selector qual.name, or the bare identifier when the file
// is package xsderr itself.
func (s *fileScope) refers(e ast.Expr, name string) bool {
	if s.qual == "" {
		id, ok := e.(*ast.Ident)
		return ok && id.Name == name
	}
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == s.qual
}

// ruleConst is one declared Rule constant: its name, its literal string value
// and where it is written.
type ruleConst struct {
	name  string
	value string
	pos   token.Pos
}

// ruleConstants returns every constant in the file declared with type
// xsderr.Rule and a string-literal value, in declaration order. A constant whose
// value is not a literal (an expression over other constants) is skipped: there
// is no value to check without type-checking the package, and no such constant
// exists in the module.
func (s *fileScope) ruleConstants() []ruleConst {
	var out []ruleConst
	for _, decl := range s.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || !s.refers(vs.Type, "Rule") {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					break
				}
				v, ok := stringLit(vs.Values[i])
				if !ok {
					continue
				}
				out = append(out, ruleConst{name: name.Name, value: v, pos: name.Pos()})
			}
		}
	}
	return out
}

// literalViolation is one string literal caught in a Rule position.
type literalViolation struct {
	what string
	pos  token.Pos
}

// ruleLiterals reports the number of xsderr.New/Wrap call sites the file
// contains (the vacuity counter) and every string literal found in a Rule
// position. Violations are collected, never returned on the first hit, so one
// run reports them all (STYLE S3).
func (s *fileScope) ruleLiterals() (calls int, violations []literalViolation) {
	ast.Inspect(s.file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			c, v := s.checkCall(x)
			calls += c
			violations = append(violations, v...)
		case *ast.CompositeLit:
			violations = append(violations, s.checkErrorLiteral(x)...)
		case *ast.FuncDecl:
			violations = append(violations, s.checkReturns(x.Type, x.Body)...)
		case *ast.FuncLit:
			violations = append(violations, s.checkReturns(x.Type, x.Body)...)
		}
		return true
	})
	return calls, violations
}

// checkCall covers positions 1 and 4: the rule argument of New/Wrap, and the
// operand of an explicit xsderr.Rule(...) conversion.
func (s *fileScope) checkCall(call *ast.CallExpr) (calls int, violations []literalViolation) {
	for _, fn := range []string{"New", "Wrap"} {
		if !s.refers(call.Fun, fn) || len(call.Args) == 0 {
			continue
		}
		calls++
		if _, ok := stringLit(call.Args[0]); ok {
			violations = append(violations, literalViolation{
				what: "the rule argument of xsderr." + fn,
				pos:  call.Args[0].Pos(),
			})
		}
	}
	if s.refers(call.Fun, "Rule") && len(call.Args) == 1 {
		if _, ok := stringLit(call.Args[0]); ok {
			violations = append(violations, literalViolation{
				what: "the operand of an xsderr.Rule conversion",
				pos:  call.Args[0].Pos(),
			})
		}
	}
	return calls, violations
}

// checkErrorLiteral covers position 2: the Rule field of an xsderr.Error
// composite literal, the second way to build an *Error without going through
// New/Wrap.
func (s *fileScope) checkErrorLiteral(lit *ast.CompositeLit) []literalViolation {
	if !s.refers(lit.Type, "Error") {
		return nil
	}
	var out []literalViolation
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Rule" {
			continue
		}
		if _, ok := stringLit(kv.Value); ok {
			out = append(out, literalViolation{
				what: "the Rule field of an xsderr.Error literal",
				pos:  kv.Value.Pos(),
			})
		}
	}
	return out
}

// checkReturns covers position 3: a literal returned from a function whose
// result type is xsderr.Rule. This is the position that would otherwise let a
// rule ID hide in a switch arm of a kind→rule mapper, out of sight of every
// other check. Nested function literals are left to their own visit, so a
// closure's returns are judged against the closure's signature, not the
// enclosing function's.
//
// The one literal allowed here is "": the ZERO Rule means "no rule", which is
// what RuleOf returns beside ok=false, not a claimed rule ID. It stays a
// violation in every other position, where an error would be constructed citing
// nothing at all (STYLE E2).
func (s *fileScope) checkReturns(typ *ast.FuncType, body *ast.BlockStmt) []literalViolation {
	if body == nil || typ.Results == nil {
		return nil
	}
	var ruleIdx []int
	total := 0
	for _, field := range typ.Results.List {
		n := max(len(field.Names), 1)
		for i := range n {
			if s.refers(field.Type, "Rule") {
				ruleIdx = append(ruleIdx, total+i)
			}
		}
		total += n
	}
	if len(ruleIdx) == 0 {
		return nil
	}
	var out []literalViolation
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		// A bare `return` (named results) or a `return f()` passthrough has no
		// per-result expression to inspect; only the positional form does.
		if !ok || len(ret.Results) != total {
			return true
		}
		for _, i := range ruleIdx {
			if v, ok := stringLit(ret.Results[i]); ok && v != "" {
				out = append(out, literalViolation{
					what: "a returned xsderr.Rule",
					pos:  ret.Results[i].Pos(),
				})
			}
		}
		return true
	})
	return out
}

// stringLit unquotes e when it is a string literal, and reports false for every
// other expression — an identifier, a selector, a conversion.
func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

// goFiles lists the module's .go files in a stable walk order, skipping dot
// directories (.git, .claude) and every testdata tree — those hold fixtures and
// vendored suites, not module code. withTests keeps _test.go files.
func goFiles(t *testing.T, root string, withTests bool) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !withTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return out
}

// moduleRoot ascends from the working directory to the directory holding
// go.mod, so the scan covers the whole module however the test is invoked. It
// mirrors tools/rulecat's helper of the same name; the two are a dozen lines
// each and share no consumer worth a package.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above working directory")
		}
		dir = parent
	}
}
