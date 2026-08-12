package validate

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The test double: a literal infoset tree implementing Element, Attribute,
// Text and Children, standing in for the source adapters (validate/xmlsrc
// and its siblings) that do not exist yet. It is the in-package consumer
// proving the seam is implementable from outside the engine.

// testElement is an element information item built from literals.
type testElement struct {
	name     xsd.QName
	attrs    []Attribute
	kids     []Child
	kidsErr  error // reported by the cursor after the last kid, if non-nil
	bindings map[string]string
	loc      xsderr.Loc
}

func (e *testElement) Name() xsd.QName         { return e.name }
func (e *testElement) Attributes() []Attribute { return e.attrs }
func (e *testElement) Children() Children      { return &testChildren{kids: e.kids, err: e.kidsErr} }
func (e *testElement) Loc() xsderr.Loc         { return e.loc }

func (e *testElement) LookupPrefix(prefix string) (string, bool) {
	uri, ok := e.bindings[prefix]
	return uri, ok
}

// testChildren is a cursor over a literal slice of children, optionally
// faulting once the slice is drained.
type testChildren struct {
	kids []Child
	at   int
	err  error
}

func (c *testChildren) Next() (Child, bool) {
	if c.at >= len(c.kids) {
		return Child{}, false
	}
	kid := c.kids[c.at]
	c.at++
	return kid, true
}

func (c *testChildren) Err() error { return c.err }

// testAttribute is an attribute information item built from literals.
type testAttribute struct {
	name  xsd.QName
	value string
	loc   xsderr.Loc
}

func (a *testAttribute) Name() xsd.QName { return a.name }
func (a *testAttribute) Value() string   { return a.value }
func (a *testAttribute) Loc() xsderr.Loc { return a.loc }

// testText is a run of character information items built from literals.
type testText struct {
	data string
	loc  xsderr.Loc
}

func (t *testText) Data() string    { return t.data }
func (t *testText) Loc() xsderr.Loc { return t.loc }

var (
	_ Element   = (*testElement)(nil)
	_ Children  = (*testChildren)(nil)
	_ Attribute = (*testAttribute)(nil)
	_ Text      = (*testText)(nil)
)

func TestChildArms(t *testing.T) {
	elem := &testElement{name: xsd.QName{Local: "e"}}
	text := &testText{data: "x"}

	ec := ElementChild(elem)
	if got, ok := ec.Element(); !ok || got != Element(elem) {
		t.Errorf("ElementChild(e).Element() = (%v,%v), want (e,true)", got, ok)
	}
	if got, ok := ec.Text(); ok || got != nil {
		t.Errorf("ElementChild(e).Text() = (%v,%v), want (nil,false)", got, ok)
	}

	tc := TextChild(text)
	if got, ok := tc.Text(); !ok || got != Text(text) {
		t.Errorf("TextChild(t).Text() = (%v,%v), want (t,true)", got, ok)
	}
	if got, ok := tc.Element(); ok || got != nil {
		t.Errorf("TextChild(t).Element() = (%v,%v), want (nil,false)", got, ok)
	}

	// The zero Child holds neither arm; the walk panics on one
	// (TestAssessPanicsOnZeroChild), it is not silently skipped.
	var zero Child
	if _, ok := zero.Element(); ok {
		t.Error("zero Child reports an element arm")
	}
	if _, ok := zero.Text(); ok {
		t.Error("zero Child reports a text arm")
	}
}

func TestChildConstructorsRejectNil(t *testing.T) {
	t.Run("element", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("ElementChild(nil) did not panic")
			}
		}()
		ElementChild(nil)
	})
	t.Run("text", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("TextChild(nil) did not panic")
			}
		}()
		TextChild(nil)
	})
}
