package xpath

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// foldComparison gathers, over every {test} CompileAssertionTest compiles
// while this package's tests run, each disagreement between a fold over the
// tree (ctaReadsChild, ctaChildAttributes, ctaKeysOf, ctaReadsContextItem) and
// the per-node method or recorded flag it replaces (#2599). ReadsContextItem's
// recorded flag may over-report where the parse discarded an operand holding
// `.` into ctaTypeError or ctaEmptyValue; those texts are listed apart.
type foldComparison struct {
	compiled      int
	disagreements []string
	overReported  []string
}

var foldCompare foldComparison

// TestMain runs the package's tests with every compile compared, and fails
// where any fold disagrees with its method.
func TestMain(m *testing.M) {
	ctaCompiledAssertion = foldCompare.record
	code := m.Run()
	ctaCompiledAssertion = nil
	fmt.Printf("fold comparison: %d compiled {test}s, %d disagreements, %d over-reported by the recorded flag\n",
		foldCompare.compiled, len(foldCompare.disagreements), len(foldCompare.overReported))
	for _, d := range foldCompare.disagreements {
		fmt.Println("  DISAGREE", d)
	}
	for _, o := range foldCompare.overReported {
		fmt.Println("  OVER-REPORTED", o)
	}
	if code == 0 && len(foldCompare.disagreements) != 0 {
		code = 1
	}
	os.Exit(code)
}

func (c *foldComparison) record(expr string, t AssertionTest) {
	c.compiled++
	root := t.root
	names := append(foldNames(reflect.ValueOf(root), nil), xsd.QName{Local: "zz-none"})
	for _, name := range names {
		if got, want := ctaReadsChild(root, name), root.readsChild(name); got != want {
			c.disagree(expr, "ReadsChild(%v): fold %v, method %v", name, got, want)
		}
		if got, want := ctaChildAttributes(root, name, nil), root.childAttributes(name, nil); !slices.Equal(got, want) {
			c.disagree(expr, "childAttributes(%v): fold %v, method %v", name, got, want)
		}
	}
	got, want := ctaKeysOf(root, nil), root.counted(nil)
	if len(got) != len(want) {
		c.disagree(expr, "counted: fold %d keys %v, method %d keys %v", len(got), got, len(want), want)
	}
	for i := range min(len(got), len(want)) {
		if reflect.TypeOf(got[i]) != reflect.TypeOf(want[i]) || !got[i].same(want[i]) {
			c.disagree(expr, "counted[%d]: fold %#v, method %#v", i, got[i], want[i])
		}
	}
	fold := ctaReadsContextItem(root)
	switch {
	case fold == t.readsContextItem:
	case fold:
		c.disagree(expr, "ReadsContextItem: fold true, recorded false")
	case foldHoldsDiscard(root):
		c.overReported = append(c.overReported, fmt.Sprintf("%q", expr))
	default:
		c.disagree(expr, "ReadsContextItem: recorded true, fold false, and no ctaTypeError or ctaEmptyValue in %#v", root)
	}
}

func (c *foldComparison) disagree(expr, format string, args ...any) {
	c.disagreements = append(c.disagreements, fmt.Sprintf("%q: ", expr)+fmt.Sprintf(format, args...))
}

// foldHoldsDiscard reports whether n holds a node a parse discards an operand
// into: ctaTypeError, or ctaEmptyValue.
func foldHoldsDiscard(n ctaNode) bool {
	switch n.(type) {
	case ctaTypeError, ctaEmptyValue:
		return true
	}
	for o := range n.eachOperand {
		if foldHoldsDiscard(o) {
			return true
		}
	}
	return false
}

var foldQNameType = reflect.TypeFor[xsd.QName]()

// foldNames appends every xsd.QName v holds, through interfaces, structs and
// slices, never through a pointer or a map.
func foldNames(v reflect.Value, into []xsd.QName) []xsd.QName {
	if v.Type() == foldQNameType {
		return append(into, xsd.QName{Space: v.Field(0).String(), Local: v.Field(1).String()})
	}
	kind := v.Kind()
	if kind == reflect.Interface && !v.IsNil() {
		return foldNames(v.Elem(), into)
	}
	if kind == reflect.Struct {
		for i := range v.NumField() {
			into = foldNames(v.Field(i), into)
		}
	}
	if kind == reflect.Slice || kind == reflect.Array {
		for i := range v.Len() {
			into = foldNames(v.Index(i), into)
		}
	}
	return into
}
