package validate

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// recorder is a slog.Handler that appends one line per record, so a test can
// compare the walk's visits against an expected sequence. It qualifies each
// key with the open groups rather than dropping them, so an expectation here
// keeps describing the log the engine actually emits (STYLE L1).
type recorder struct {
	visits *[]string
	prefix string
}

func (h recorder) Enabled(context.Context, slog.Level) bool { return true }

func (h recorder) Handle(_ context.Context, r slog.Record) error {
	parts := []string{r.Message}
	r.Attrs(func(a slog.Attr) bool {
		parts = append(parts, h.prefix+a.Key+"="+a.Value.String())
		return true
	})
	*h.visits = append(*h.visits, strings.Join(parts, " "))
	return nil
}

// WithAttrs panics rather than recording or dropping: the walk calls no
// logger method that reaches it, so the day one does, the expectations here
// must be rewritten to say what those attributes are.
func (h recorder) WithAttrs([]slog.Attr) slog.Handler {
	panic("validate_test: recorder.WithAttrs: the walk does not log with pre-bound attributes")
}

func (h recorder) WithGroup(name string) slog.Handler {
	h.prefix += name + "."
	return h
}

func recordingLogger() (*slog.Logger, *[]string) {
	visits := new([]string)
	return slog.New(recorder{visits: visits}), visits
}

func emptySchema(t *testing.T) *xsd.Schema {
	t.Helper()
	schema, err := xsd.NewSchemaBuilder().Finalize()
	if err != nil {
		t.Fatalf("finalizing an empty schema: %v", err)
	}
	return schema
}

func loc(line, col int) xsderr.Loc {
	return xsderr.Loc{URI: "instance.xml", Line: line, Col: col}
}

// sampleTree is the literal instance every walk test drives, standing for
//
//	<root xmlns:p="urn:p" id="r1" p:lang="en">top
//	  <a>in a</a>between
//	  <b><c/></b>
//	</root>
//
// with one distinct Loc per information item, so a visit is identifiable.
func sampleTree() *testElement {
	c := &testElement{name: xsd.QName{Local: "c"}, loc: loc(5, 6)}
	b := &testElement{
		name: xsd.QName{Local: "b"},
		kids: []Child{ElementChild(c)},
		loc:  loc(5, 3),
	}
	a := &testElement{
		name: xsd.QName{Local: "a"},
		kids: []Child{TextChild(&testText{data: "in a", loc: loc(3, 6)})},
		loc:  loc(3, 3),
	}
	return &testElement{
		name: xsd.QName{Local: "root"},
		attrs: []Attribute{
			&testAttribute{name: xsd.QName{Local: "id"}, value: "r1", loc: loc(1, 23)},
			&testAttribute{name: xsd.QName{Space: "urn:p", Local: "lang"}, value: "en", loc: loc(1, 31)},
		},
		kids: []Child{
			TextChild(&testText{data: "top", loc: loc(2, 3)}),
			ElementChild(a),
			TextChild(&testText{data: "between", loc: loc(4, 3)}),
			ElementChild(b),
		},
		bindings: map[string]string{"p": "urn:p"},
		loc:      loc(1, 1),
	}
}

// wantVisits is sampleTree's every information item, once each, in the order
// cvc-assess-elt (§3.3.4.6) fixes: the element, then its [[attributes]], then
// its [[children]] in document order, recursively. Every key is qualified by
// the "validate" group New installs (STYLE L1).
var wantVisits = []string{
	"assessing element validate.name=root validate.loc=instance.xml:1:1",
	"assessing attribute validate.name=id validate.loc=instance.xml:1:23",
	"assessing attribute validate.name={urn:p}lang validate.loc=instance.xml:1:31",
	"assessing text validate.chars=3 validate.loc=instance.xml:2:3",
	"assessing element validate.name=a validate.loc=instance.xml:3:3",
	"assessing text validate.chars=4 validate.loc=instance.xml:3:6",
	"assessing text validate.chars=7 validate.loc=instance.xml:4:3",
	"assessing element validate.name=b validate.loc=instance.xml:5:3",
	"assessing element validate.name=c validate.loc=instance.xml:5:6",
}

func TestAssessWalksEveryNodeOnceInDocumentOrder(t *testing.T) {
	log, visits := recordingLogger()
	v, err := New(emptySchema(t), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := v.Assess(sampleTree())

	if !slices.Equal(*visits, wantVisits) {
		t.Errorf("walk visited\n\t%s\nwant\n\t%s",
			strings.Join(*visits, "\n\t"), strings.Join(wantVisits, "\n\t"))
	}
	if got := res.Violations(); len(got) != 0 {
		t.Errorf("Violations() = %v, want none", got)
	}
	if err := res.Err(); err != nil {
		t.Errorf("Err() = %v, want nil for a walk that completed", err)
	}
}

// A Validator holds no walk state, so assessing twice must not carry the
// first walk into the second.
func TestValidatorIsReusable(t *testing.T) {
	log, visits := recordingLogger()
	v, err := New(emptySchema(t), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	first := v.Assess(sampleTree())
	firstVisits := slices.Clone(*visits)
	*visits = nil
	second := v.Assess(sampleTree())

	if !slices.Equal(*visits, firstVisits) {
		t.Errorf("second walk visited\n\t%s\nwant the first walk's\n\t%s",
			strings.Join(*visits, "\n\t"), strings.Join(firstVisits, "\n\t"))
	}
	if first == second {
		t.Error("Assess returned the same *Result twice")
	}
	if err := second.Err(); err != nil {
		t.Errorf("second Err() = %v, want nil", err)
	}
}

// A fault the source reports mid-document stops the walk and surfaces as
// Result.Err, and the subtree past it is not assessed.
func TestAssessStopsAtASourceFault(t *testing.T) {
	fault := errors.New("truncated document")
	faulty := &testElement{name: xsd.QName{Local: "a"}, kidsErr: fault, loc: loc(2, 3)}
	unreached := &testElement{name: xsd.QName{Local: "z"}, loc: loc(3, 3)}
	root := &testElement{
		name: xsd.QName{Local: "root"},
		kids: []Child{ElementChild(faulty), ElementChild(unreached)},
		loc:  loc(1, 1),
	}

	log, visits := recordingLogger()
	v, err := New(emptySchema(t), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := v.Assess(root)

	if !errors.Is(res.Err(), fault) {
		t.Errorf("Err() = %v, want it to wrap %v", res.Err(), fault)
	}
	want := []string{
		"assessing element validate.name=root validate.loc=instance.xml:1:1",
		"assessing element validate.name=a validate.loc=instance.xml:2:3",
	}
	if !slices.Equal(*visits, want) {
		t.Errorf("walk visited\n\t%s\nwant\n\t%s",
			strings.Join(*visits, "\n\t"), strings.Join(want, "\n\t"))
	}
}

// A Child built by neither constructor names no information item. That is an
// adapter bug and not a fault in the source, so it panics — Result.Err is
// reserved for the source faults that stop a walk, and routing both there
// would report a bug in an adapter to a user as a broken document.
func TestAssessPanicsOnZeroChild(t *testing.T) {
	root := &testElement{
		name: xsd.QName{Local: "root"},
		kids: []Child{{}, ElementChild(&testElement{name: xsd.QName{Local: "z"}})},
		loc:  loc(1, 1),
	}

	log, visits := recordingLogger()
	v, err := New(emptySchema(t), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Assess did not panic on a Child holding neither arm")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "Child holds neither arm") {
			t.Errorf("Assess panicked with %v, wanted the message to name the empty Child", r)
		}
		// The walk stops at the empty child: root is visited, the sibling
		// past it is not.
		want := []string{"assessing element validate.name=root validate.loc=instance.xml:1:1"}
		if !slices.Equal(*visits, want) {
			t.Errorf("walk visited\n\t%s\nwant\n\t%s",
				strings.Join(*visits, "\n\t"), strings.Join(want, "\n\t"))
		}
	}()

	v.Assess(root)
}

// The reference tree models xmlns:p the way an adapter must: not as an
// attribute (Appendix D's separate [[namespace attributes]] property), and
// reachable only through LookupPrefix.
func TestNamespaceDeclarationsAreNotAttributes(t *testing.T) {
	root := sampleTree()
	for _, a := range root.Attributes() {
		if a.Name().Local == "xmlns" || strings.HasPrefix(a.Name().Local, "xmlns:") {
			t.Errorf("Attributes() reports the namespace declaration %s", a.Name())
		}
	}
	if uri, ok := root.LookupPrefix("p"); !ok || uri != "urn:p" {
		t.Errorf("LookupPrefix(p) = (%q,%v), want (urn:p,true)", uri, ok)
	}
	if _, ok := root.LookupPrefix("unbound"); ok {
		t.Error("LookupPrefix(unbound) resolved")
	}
}

func TestNewRejectsNilSchema(t *testing.T) {
	v, err := New(nil)
	if err == nil {
		t.Fatal("New(nil) = nil error, want one")
	}
	if v != nil {
		t.Errorf("New(nil) returned a %T, want nil", v)
	}
	var xerr *xsderr.Error
	if errors.As(err, &xerr) {
		t.Errorf("New(nil) returned %T: a missing schema is not a validity verdict", xerr)
	}
}

func TestNewDefaultsAndNilLogger(t *testing.T) {
	// The zero option set and an explicit nil logger are both usable: a
	// walk under either runs to completion and logs nothing (STYLE L1).
	for _, opts := range [][]Option{nil, {WithLogger(nil)}} {
		v, err := New(emptySchema(t), opts...)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if res := v.Assess(sampleTree()); res.Err() != nil {
			t.Errorf("Err() = %v, want nil", res.Err())
		}
	}
}

func TestSchemaViewResolvesElements(t *testing.T) {
	schema := emptySchema(t)
	v, err := New(schema)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, ok := v.Schema().Element(xsd.QName{Local: "absent"}); ok {
		t.Error("the empty schema resolved an element declaration")
	}
}

func TestResultViolationsAreCopied(t *testing.T) {
	// Violations is a window, not a handle: mutating what it returns must
	// not reach the Result (xsd.Schema.Elements' convention).
	res := &Result{violations: []*xsderr.Error{xsderr.New("cvc-elt.1", loc(1, 1), "sample")}}
	got := res.Violations()
	got[0] = nil
	if res.Violations()[0] == nil {
		t.Error("Violations() shares its backing array with the Result")
	}
	if (&Result{}).Violations() != nil {
		t.Error("Violations() = non-nil for a Result carrying none")
	}
}

func TestAssessRejectsNilRoot(t *testing.T) {
	v, err := New(emptySchema(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("Assess(nil) did not panic")
		}
	}()
	v.Assess(nil)
}
