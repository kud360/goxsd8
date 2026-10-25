package codegen

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// A DeclineError is a construct [Generate] declines; Generate's doc states
// how it is returned.
type DeclineError struct {
	construct construct
	loc       xsderr.Loc
	// detail is the decline's text before its Loc: the construct's format
	// over the arguments the declining site passed.
	detail string
}

// Construct returns the decline's Construct, as Generate's doc defines it.
func (e *DeclineError) Construct() Construct { return Construct{e.construct} }

// Loc returns the decline's Loc, as Generate's doc defines it.
func (e *DeclineError) Loc() xsderr.Loc { return e.loc }

// Error names the construct and the Loc; it does not name the target.
func (e *DeclineError) Error() string {
	return e.detail + " at " + e.loc.String() + " is not supported by codegen, a generator limitation, not a schema fault"
}

// Unwrap returns errors.ErrUnsupported.
func (e *DeclineError) Unwrap() error { return errors.ErrUnsupported }

// A Construct is a construct Generate declines: two declines of the same
// construct have equal Constructs; the zero Construct names none.
type Construct struct{ c construct }

// String returns the construct's name, distinct for each Construct; it is not
// the decline's message.
func (c Construct) String() string {
	if c.c == 0 || int(c.c) >= len(constructs) {
		return "Construct(" + strconv.Itoa(int(c.c)) + ")"
	}
	return constructs[c.c].name
}

// construct is the closed set of constructs the generator declines, one per
// declining site. The zero value is invalid (STYLE T1).
type construct uint8

// The construct values, in the order of their declining sites in emit.go.
const (
	constructElementNillable construct = iota + 1
	constructElementAbstract
	constructSubstitutionGroup
	constructElementValueConstraint
	constructTypeTable
	constructAnyTypeElement
	constructSpecialType
	constructUnionType
	constructAbsentVariety
	constructNoPrimitive
	constructUncoveredPrimitive
	constructChoiceAttributeUses
	constructEmptyChoice
	constructEmptiableChoice
	constructDerivation
	constructRestrictionBase
	constructComplexTypeAbstract
	constructAssertions
	constructAttributeWildcard
	constructSimpleContent
	constructMixedContent
	constructOpenContent
	constructContentOccurs
	constructContentGroupRef
	constructAll
	constructVacuousParticle
	constructAttributeValueConstraint
	constructNamedComplexLocal
	constructElementRef
	constructGroupRef
	constructNestedGroup
	constructWildcard
)

// constructs holds, indexed by construct, each construct's name, which
// Construct.String returns, and the format its decline's detail is rendered
// from. Index 0, the invalid construct, holds neither.
var constructs = [...]struct{ name, format string }{
	constructElementNillable:          {"element {nillable}", "%s with {nillable} true"},
	constructElementAbstract:          {"element {abstract}", "%s with {abstract} true"},
	constructSubstitutionGroup:        {"{substitution group affiliations}", "%s with {substitution group affiliations}"},
	constructElementValueConstraint:   {"element {value constraint}", "%s with a {value constraint}"},
	constructTypeTable:                {"{type table}", "%s with a {type table}"},
	constructAnyTypeElement:           {"element of type xs:anyType", "%s with {type definition} xs:anyType"},
	constructSpecialType:              {"special type", "%s, a special type"},
	constructUnionType:                {"union type", "%s, a union type"},
	constructAbsentVariety:            {"absent {variety}", "%s, of absent {variety}"},
	constructNoPrimitive:              {"no {primitive type definition}", "%s, with no {primitive type definition}"},
	constructUncoveredPrimitive:       {"uncovered primitive", "%s, whose {primitive type definition} is %s"},
	constructChoiceAttributeUses:      {"choice with {attribute uses}", "%s whose content is a choice, with {attribute uses}"},
	constructEmptyChoice:              {"choice with no {particles}", "%s whose choice has no {particles}"},
	constructEmptiableChoice:          {"emptiable choice", "%s whose choice is emptiable: its alternative element %s has {min occurs} 0"},
	constructDerivation:               {"derivation other than restriction", "%s, derived by %s"},
	constructRestrictionBase:          {"restriction of a type other than xs:anyType", "%s, a restriction of %s rather than of xs:anyType"},
	constructComplexTypeAbstract:      {"complex type {abstract}", "%s with {abstract} true"},
	constructAssertions:               {"{assertions}", "%s with {assertions}"},
	constructAttributeWildcard:        {"{attribute wildcard}", "%s with an {attribute wildcard}"},
	constructSimpleContent:            {"simple content", "%s with simple content"},
	constructMixedContent:             {"mixed content", "%s with mixed content"},
	constructOpenContent:              {"{open content}", "%s with {open content}"},
	constructContentOccurs:            {"content particle not exactly once", "%s whose content particle occurs %s times rather than exactly once"},
	constructContentGroupRef:          {"content particle <group ref>", "%s whose content particle is a <group ref>"},
	constructAll:                      {"all compositor", "%s with an all compositor"},
	constructVacuousParticle:          {"particle with {max occurs} 0", "a particle of element %s with {max occurs} 0 in %s"},
	constructAttributeValueConstraint: {"attribute effective value constraint", "%s with a %s effective value constraint (§3.5.4 key-evc)"},
	constructNamedComplexLocal:        {"local element of named complex type", "local element %s with named complex {type definition} %s"},
	constructElementRef:               {"<element ref> particle", "an <element ref> particle to %s in %s"},
	constructGroupRef:                 {"<group ref> particle", "a <group ref> particle to %s in %s"},
	constructNestedGroup:              {"nested model group particle", "a nested %s model group particle in %s"},
	constructWildcard:                 {"wildcard particle", "a wildcard particle in %s"},
}

// decline returns the *DeclineError for construct c, which the generator does
// not cover, at loc, its detail c's format over args. No schema rule is
// violated, so it carries no xsderr rule (STYLE E2). Its text already says the
// construct is unsupported, so DeclineError.Error prints none of
// errors.ErrUnsupported's text.
//
// GAP(codegen): every construct outside the package doc's "# Coverage" slice
// declines here; that section lists them under the same marker. #2900 owns
// lifting them.
func decline(loc xsderr.Loc, c construct, args ...any) error {
	return &DeclineError{construct: c, loc: loc, detail: fmt.Sprintf(constructs[c].format, args...)}
}
