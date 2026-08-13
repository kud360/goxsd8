package xmlsrc

import (
	"fmt"

	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The engine meets this package only through these four views.
//
// *xmltree.CharData is validate.Text as it stands: Data reports the decoded
// characters and Loc the run's position, which is the whole interface. A
// forwarding wrapper would restate both methods and could only drift from
// them (STYLE T4), so a character-data node is handed over as it is. It
// couples nothing: validate/imports_test.go bans github.com/kud360/goxsd8/parser
// by prefix over validate's whole transitive closure, so the engine can
// never name the concrete type, let alone switch on it.
var (
	_ validate.Element   = (*element)(nil)
	_ validate.Attribute = attribute{}
	_ validate.Children  = (*children)(nil)
	_ validate.Text      = (*xmltree.CharData)(nil)
)

// qname converts a resolved xmltree name to the QName the schema side
// carries. Both are the (namespace name, local name) pair Datatypes §3.3.18
// defines, so the conversion is a copy and never a lookup.
func qname(n xmltree.Name) xsd.QName {
	return xsd.QName{Space: n.Space(), Local: n.Local()}
}

// element is one element information item: the start tag xmltree resolved,
// over the shared walk its [[children]] are still to be pulled from.
type element struct {
	w     *walker
	start *xmltree.StartElement
	// depth is the walk depth this element's [[children]] occupy — one
	// deeper than the depth the element itself was opened at.
	depth int
	// n is the walker's token count when this element was yielded, which is
	// the only position its [[children]] are still the next tokens at (see
	// Children).
	n int
}

func (e *element) Name() xsd.QName { return qname(e.start.Name()) }

// Attributes converts the start tag's attributes. xmltree already excludes
// namespace declarations and already reports document order, which is
// exactly this method's obligation: neither is redone here.
func (e *element) Attributes() []validate.Attribute {
	attrs := e.start.Attributes()
	if len(attrs) == 0 {
		return nil
	}
	out := make([]validate.Attribute, 0, len(attrs))
	for _, a := range attrs {
		out = append(out, attribute{a: a})
	}
	return out
}

// Children opens a cursor over the shared token stream at this element's
// child depth. It is not a second decode: the tokens it yields are the ones
// the walk has not pulled yet, which are this element's [[children]] only
// while the stream still stands where this element was yielded.
//
// It panics once anything else has pulled from the walk — the cursor that
// yielded this element advanced past it, discarding the subtree, or this
// element's own cursor already ran. The tokens at this depth are then a
// later element's children, and a cursor keyed on depth alone would report
// them as this element's. A caller that descends too late is an
// adapter-contract violation like a Child holding neither arm, not a fault
// in the source, so it panics rather than reaching [validate.Result.Err].
func (e *element) Children() validate.Children {
	if e.n != e.w.n {
		panic(fmt.Sprintf("validate/xmlsrc: Children called after the walk left %s", e.Name()))
	}
	return &children{w: e.w, depth: e.depth}
}

func (e *element) LookupPrefix(prefix string) (string, bool) {
	return e.start.LookupPrefix(prefix)
}

func (e *element) Loc() xsderr.Loc { return e.start.Loc() }

// attribute is one attribute information item. xsi:type, xsi:nil,
// xsi:schemaLocation and xsi:noNamespaceSchemaLocation arrive through it
// like any other attribute: §2.7's note makes them attribute information
// items identified by namespace and local name, and cvc-complex-type clause
// 2's "excepting" carve-out is the engine's to apply.
type attribute struct {
	a xmltree.Attribute
}

func (a attribute) Name() xsd.QName { return qname(a.a.Name()) }

// Value reports A.[[normalized value]] as the source produced it. XML 1.0
// §3.3.3 normalization is an infoset precondition (Appendix D), already
// applied upstream, and this layer applies nothing further.
func (a attribute) Value() string { return a.a.Value() }

func (a attribute) Loc() xsderr.Loc { return a.a.Loc() }

// children is a cursor over one element's [[children]], reading the walk's
// single token stream at the depth those children occupy.
//
// The cursor does not trust the engine to descend into every element it is
// handed: an element child the engine takes and never calls Children on (a
// processContents="skip" wildcard match) leaves its whole subtree in the
// stream, and those tokens are discarded here rather than reported to this
// element as further children.
type children struct {
	w     *walker
	depth int
	// done latches this cursor's own end tag, so a second Next after
	// exhaustion reports false instead of taking the next sibling of the
	// element this cursor belongs to.
	done bool
}

func (c *children) Next() (validate.Child, bool) {
	if c.done || c.w.err != nil {
		return validate.Child{}, false
	}
	for {
		node, at, err := c.w.next()
		if err != nil {
			c.w.err = err
			return validate.Child{}, false
		}
		if at > c.depth {
			// A token of a deeper subtree the engine did not descend into.
			continue
		}
		switch n := node.(type) {
		case *xmltree.StartElement:
			return validate.ElementChild(&element{w: c.w, start: n, depth: at + 1, n: c.w.n}), true
		case *xmltree.CharData:
			// One Text per source run: text, CDATA and entity-expanded text
			// stay separate runs, and the engine concatenates them into the
			// ·initial value·.
			return validate.TextChild(n), true
		case *xmltree.EndElement:
			if at == c.depth {
				// A child element of this one closed.
				continue
			}
			c.done = true
			return validate.Child{}, false
		default:
			// Unreachable: xmltree.Reader.Token yields only these three node
			// kinds, having dropped comments, PIs and directives itself.
			continue
		}
	}
}

// Err reports the walk's one latched source fault, which every cursor over
// the same walk reports (STYLE D3): the stream that faulted is shared, so
// the fault is one fact and is stored once.
func (c *children) Err() error { return c.w.err }
