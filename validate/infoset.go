package validate

import (
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// Element is the element information item the engine assesses, as a source
// adapter presents it. It carries the Appendix D properties a cvc- rule
// actually reads off an element — expanded name, [[attributes]],
// [[children]], [[in-scope namespaces]] — plus the location diagnostics
// need, and nothing a concrete decoder would add (PRINCIPLES 8).
//
// Implementations live outside this package, so this interface is complete
// as it stands: a property a later rule needs arrives as a separate
// capability interface the engine narrows to, never as a method added here
// (see the package doc).
type Element interface {
	// Name reports the element's ·expanded name·, [[namespace name]] plus
	// [[local name]], in the same [xsd.QName] the declaration side carries,
	// so cvc-elt clause 1 is a comparison and not a second encoding of
	// expanded names. A zero QName — an empty Local — is an implementer
	// bug: no information item has an empty local name.
	Name() xsd.QName

	// Attributes reports E.[[attributes]] in SOURCE order, EXCLUDING
	// namespace declarations. Appendix D holds xmlns and xmlns:p in the
	// separate [[namespace attributes]] property and cvc-complex-type
	// clause 2 iterates [[attributes]] alone, so an adapter that leaves
	// them in this slice rejects every namespaced document for undeclared
	// attributes. Namespace bindings reach the engine through LookupPrefix
	// and nowhere else.
	//
	// Source order is the implementer's obligation, not a courtesy: the
	// engine reports violations in the order it meets them, so an order
	// that varies per run varies the diagnostics (STYLE D1/D2). No element
	// of the slice is nil.
	Attributes() []Attribute

	// Children reports E.[[children]] — its element and character
	// information items — as a pull cursor in document order, which is the
	// order cvc-complex-type clause 1.4 matches them in ("taken in order").
	// It never returns nil: an element with no children returns a cursor
	// whose first Next reports false.
	Children() Children

	// LookupPrefix resolves prefix against E.[[in-scope namespaces]], so a
	// QName-valued lexical occurring here (xsi:type="p:foo") resolves in
	// the context it was written in (Datatypes §3.3.18). The empty prefix
	// yields the default namespace; ok is false for an unbound prefix.
	LookupPrefix(prefix string) (uri string, ok bool)

	// Loc reports where the element begins in its source document.
	Loc() xsderr.Loc
}

// Attribute is one attribute information item of an [Element], as a source
// adapter presents it. Namespace declarations are not attributes and never
// appear as one (see [Element]'s Attributes).
type Attribute interface {
	// Name reports the attribute's ·expanded name·, on the same terms as
	// [Element]'s Name.
	Name() xsd.QName

	// Value reports A.[[normalized value]]: attribute-value normalization
	// per XML 1.0 §3.3.3 is ALREADY applied by the source, and
	// whiteSpace-facet normalization is NOT — that one belongs to the type
	// the attribute is assessed against, and applying it here would erase
	// the difference between xs:string and xs:token before any facet sees
	// the value.
	Value() string

	// Loc reports where the attribute begins in its source document.
	Loc() xsderr.Loc
}

// Text is a run of character information items in an [Element]'s
// [[children]], as a source adapter presents it. The engine concatenates an
// element's text runs to form the ·initial value· cvc-type clause 3.1.3
// tests; a run is never normalized on its own.
type Text interface {
	// Data reports the run's characters with entity and character
	// references already resolved and CDATA sections already unwrapped —
	// the [[character code]] sequence, not the source spelling.
	Data() string

	// Loc reports where the run begins in its source document.
	Loc() xsderr.Loc
}

// Children is a cursor over one [Element]'s [[children]], in document
// order. It is a pull cursor rather than a slice so a streaming source
// yields one child at a time and no walk ever holds a document (STYLE P4);
// its shape is bufio.Scanner's and parser/xmltree's Reader.Token's (STYLE
// T4).
type Children interface {
	// Next reports the next child and true, or the zero [Child] and false
	// once the children are exhausted or a fault in the source stopped
	// them — Err tells those two apart. Every child it yields is one built
	// by [ElementChild] or [TextChild]; a zero Child stops the assessment
	// with an Err, since it names no item.
	Next() (Child, bool)

	// Err reports the fault in the source that ended the children early,
	// or nil when Next reported false at the true end of them.
	Err() error
}

// Child is one item of an [Element]'s [[children]]: exactly one of an
// element or a run of text. The item kinds are closed by Appendix D, so
// this is a sum (STYLE T2) — but a struct sum rather than an interface
// sealed by an unexported method, because the adapters that construct one
// live outside this package and could not implement a sealed interface.
// [ElementChild] and [TextChild] are the only way to build one and the two
// accessors the only way to read one, so no third case exists for the
// engine to handle.
//
// Comment and processing-instruction information items are Appendix D items
// that no cvc- rule reads, and this sum deliberately does not model them: an
// adapter DROPS them rather than yielding them. Adding an arm later is
// additive, since only this package reads a Child.
type Child struct {
	elem Element
	text Text
}

// ElementChild returns the [Child] holding e. It panics if e is nil: an
// absent child is not yielded at all, so a nil one is an adapter bug, and
// catching it here beats a nil dereference mid-walk.
func ElementChild(e Element) Child {
	if e == nil {
		panic("validate: ElementChild: nil Element")
	}
	return Child{elem: e}
}

// TextChild returns the [Child] holding t. It panics if t is nil, on the
// same grounds as [ElementChild].
func TextChild(t Text) Child {
	if t == nil {
		panic("validate: TextChild: nil Text")
	}
	return Child{text: t}
}

// Element reports the child's element and true, or nil and false when the
// child is a run of text.
func (c Child) Element() (Element, bool) { return c.elem, c.elem != nil }

// Text reports the child's run of text and true, or nil and false when the
// child is an element.
func (c Child) Text() (Text, bool) { return c.text, c.text != nil }
