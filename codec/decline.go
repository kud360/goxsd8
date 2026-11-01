package codec

import (
	"errors"
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// A DeclineError is a construct [Decode] does not decode; Decode's doc states
// when it is returned.
type DeclineError struct {
	construct construct
	loc       xsderr.Loc
	// detail names the declining component and its schema Loc.
	detail string
}

// Loc returns the instance Loc of the element the decline is charged to: the
// document element for a construct of its declaration or its type, and the
// element carrying it for xsi:type.
func (e *DeclineError) Loc() xsderr.Loc { return e.loc }

// Error names the instance Loc, the construct, and the declining component
// with its schema Loc.
func (e *DeclineError) Error() string {
	return e.loc.String() + ": codec does not decode " + e.construct.name() + ": " + e.detail +
		"; a decoder limitation, not a schema or instance fault"
}

// Unwrap returns errors.ErrUnsupported.
func (e *DeclineError) Unwrap() error { return errors.ErrUnsupported }

// construct is the closed set of constructs Decode declines, one per
// declining site. The zero value is invalid (STYLE T1).
type construct uint8

// The construct values, in the order Decode finds them: the document
// element's declaration, its type, its content, each particle in order, its
// attribute uses in order, its attribute wildcard, and the instance.
const (
	constructTypeTable construct = iota + 1
	constructNillable
	constructElementDefault
	constructElementFixed
	constructSimpleRoot
	constructEmptyContent
	constructSimpleContent
	constructMixedContent
	constructOpenContent
	constructGroupRefContent
	constructChoiceContent
	constructAllContent
	constructSequenceOccurs
	constructNestedGroup
	constructWildcardParticle
	constructElementRef
	constructComplexElement
	constructSharedName
	constructList
	constructUnion
	constructSpecial
	constructAnyURI
	constructOtherPrimitive
	constructXsiAttribute
	constructAttributeDefault
	constructAttributeFixed
	constructAttributeWildcard
	constructXsiType
)

// constructNames holds, indexed by construct, each construct's name.
var constructNames = [...]string{
	constructTypeTable:         "an element declaration with a {type table}",
	constructNillable:          "an element declaration with {nillable} true",
	constructElementDefault:    "an element declaration with a default {value constraint}",
	constructElementFixed:      "an element declaration with a fixed {value constraint}",
	constructSimpleRoot:        "a document element of simple type",
	constructEmptyContent:      "empty content",
	constructSimpleContent:     "simple content",
	constructMixedContent:      "mixed content",
	constructOpenContent:       "{open content}",
	constructGroupRefContent:   "a <group ref> content particle",
	constructChoiceContent:     "choice content",
	constructAllContent:        "all content",
	constructSequenceOccurs:    "a content sequence occurring other than exactly once",
	constructNestedGroup:       "a nested model group particle",
	constructWildcardParticle:  "a wildcard particle",
	constructElementRef:        "an <element ref> particle",
	constructComplexElement:    "a local element declaration of complex type",
	constructSharedName:        "two particles sharing an expanded name",
	constructList:              "a list simple type",
	constructUnion:             "a union simple type",
	constructSpecial:           "a special simple type",
	constructAnyURI:            "an xs:anyURI simple type",
	constructOtherPrimitive:    "an atomic simple type whose primitive is not xs:string",
	constructXsiAttribute:      "an attribute use in the xsi namespace",
	constructAttributeDefault:  "an attribute use with a default value constraint",
	constructAttributeFixed:    "an attribute use with a fixed value constraint",
	constructAttributeWildcard: "an {attribute wildcard}",
	constructXsiType:           "an element carrying xsi:type",
}

// name returns the construct's name.
func (c construct) name() string {
	if c == 0 || int(c) >= len(constructNames) {
		return "construct(" + strconv.Itoa(int(c)) + ")"
	}
	return constructNames[c]
}

// decline returns the *DeclineError for construct c at the instance Loc loc,
// detail naming the declining component and its schema Loc. No schema or
// instance rule is violated, so it carries no xsderr rule (STYLE E2).
func decline(loc xsderr.Loc, c construct, detail string) *DeclineError {
	return &DeclineError{construct: c, loc: loc, detail: detail}
}
