package conformance

import (
	"path/filepath"
	"slices"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// This file activates the instance lane (issue #713) by giving the instance
// entry of defaultLanes a real executor, on the precedent schema.go set for the
// schema lane (#175). It touches nothing else in the runner (the #6 seam, STYLE
// T2): the lane's selector stays selectsKind(kindInstance), so the executor is
// handed EVERY instanceTest case and either decides it or honestly DECLINES it
// (records a Fail gap) — a case it cannot decide for the right reason never
// flips to pass. It is package-internal conformance support: it exports nothing
// and no library code imports it.
//
// # What an instanceTest asserts, and against which schema
//
// An instanceTest asks: is THIS instance document valid against the schema of
// its test group? The catalog names only the instance document, so discovery
// carries the group's schema documents alongside it (caseSpec.schemaDoc and
// schemaExtraDocs, groupSchemaDocs in conformance/runner.go) — the
// <schemaDocument> list of the group's sibling schemaTest. A group declaring any
// number of schemaTests other than exactly one yields no schema reference at all
// and DECLINES, which is not a hypothetical shape: 55 groups of the pinned suite
// declare NONE, so 55 instance cases decline for that reason alone.
//
// The schema is assembled by assembleCase, the very gate the schema lane
// decides its own cases with, and this lane declines wherever that gate does —
// plus one condition the schema lane does not have: it declines a schema
// document set the assembly REJECTED. A schema the parser found schema-invalid
// is not the schema the suite declared, so an assessment against whatever
// partial components survived would decide a different question. (The suite's
// own metadata is not consulted for this: the assembly's verdict is the fact,
// and a group whose schemaTest declares a non-valid expectation therefore
// declines through the same test as any other failed assembly.)
//
// # The only outcomes this slice can DECIDE
//
// validate.Validator.Assess charges exactly eight rules, about the ·validation
// root· and about any descendant the descent reaches (see "Charges at depth"
// below):
//
//  1. cvc-assess-elt (§3.3.4.6), when the root has no top-level element
//     declaration AND no xsi:type ·resolving· to a top-level type definition
//     (#716). It then determines neither a ·governing element declaration· nor
//     a ·governing type definition·, so key-sva clause 1 fails and §3.3.5.1's
//     e-validity gives it notKnown rather than valid — which under §5.2 ·strict
//     wildcard validation·, the mode this processor committed to in #712, is
//     what "the invoking process ... will otherwise report an error to its
//     environment" names. The document is NOT VALID, whatever else it holds.
//  2. cvc-elt (§3.3.4.3) clause 2, when the root's declaration was found and its
//     {abstract} is true; clause 3, when it carries an xsi:nil its {nillable}
//     forbids (3.1), an xsi:nil with no ·actual value· (3.2), or is ·nilled· and
//     carries [[children]] (3.2.3.1) or a fixed {value constraint} (3.2.3.2);
//     clause 4, when its xsi:type ·resolves· to a type definition that does not
//     ·override· the ·selected type definition·; and clause 5.2.2, when its
//     declaration's fixed {value constraint} disagrees with the [[children]] it
//     has (#716). The root is ·strictly assessed· and locally INVALID, so
//     e-validity clause 1.1.1.1 fails and its [validity] is invalid. The
//     document is NOT VALID, whatever else it holds.
//  3. cvc-complex-type (§3.4.4.2) clause 1, clause 2, clause 3 or clause 4, when
//     the root's ·governing type definition· was determinable and complex and
//     one of its [[attributes]] matches neither an attribute use nor an
//     {attribute wildcard} (#714), a {required} use has no attribute at all
//     (#714), a ·defaulted attribute·'s own {lexical form} is not datatype-valid
//     (#766), its [[children]] hold a character or element information item the
//     {content type}.{variety} admits none of — clauses 1.1, 1.2 and 1.3
//     (#715) — or its ·initial value· under a simple {content type} is not
//     ·valid· per String Valid against that {content type}'s {simple type
//     definition}, the other half of clause 1.2 (#775). The root is then not
//     locally ·valid· with respect to that type, so cvc-type clause 3.2 fails, so
//     cvc-elt clause 5 fails, and e-validity clause 1.1.1.1 gives the root
//     [validity] invalid exactly as case 2 does.
//  4. cvc-attribute (§3.2.4.1) clause 3 or clause 4, when one of the root's
//     [[attributes]] matched an attribute use and its lexical is not ·valid· per
//     String Valid (§3.16.4) against the declaration's {type definition}, or its
//     ·actual value· disagrees with a fixed {value constraint} on that
//     declaration (#766). Such an attribute's [validity] is invalid, and
//     e-validity's conjunction (§3.3.5.1 clause 1.1.1.2) fails for the root on
//     an invalid attribute of its own whatever else holds.
//  5. cvc-au (§3.5.4), when a matched attribute's ·actual value· disagrees with
//     a fixed {value constraint} on the attribute USE — a different property
//     from case 4's, which is why both can be charged for one attribute (#766).
//     cvc-complex-type clause 2.1 reads it, so the root is not locally ·valid·
//     and case 3's chain applies unchanged.
//  6. cvc-complex-content (§3.4.4.3) clause 1, when the root's ·governing type
//     definition· was determinable and complex, its {content type} holds a
//     particle, and the sequence of element information items in its
//     [[children]] is not ·accepted· by that particle — an item no particle
//     admits at its position, or a sequence that ends before a {min occurs} is
//     met (#715). cvc-complex-type clause 1.4 reads it, so case 3's chain
//     applies unchanged.
//  7. cvc-identity-constraint (§3.11.4), when a key, unique or keyref declared
//     on an element the descent typed is not satisfied over that element's
//     subtree — a field selecting more than one valued node (clause 3), two
//     ·qualified node set· members sharing a ·key-sequence· (clauses 4.1 and
//     4.2.2), a key whose ·target node set· is wider than its ·qualified node
//     set· (clause 4.2.1), a key-sequence element member from a {nillable}
//     declaration (clause 4.2.3), or a keyref matching no entry of the node
//     table its {referenced key} has in that element's own [identity-constraint
//     table] (clause 4.3, #718). cvc-elt clause 6 reads it, so case 3's chain
//     applies unchanged.
//  8. cvc-id (§3.3.4.5), when the ·validation root·'s [ID/IDREF table] holds a
//     binding with more than one member (clause 2, a multiply-defined ID) or —
//     only where no item of the subtree was declined — an empty one (clause 1,
//     a reference to an undefined ID) (#718). cvc-elt clause 7 reads it AT THE
//     ROOT ALONE, and it makes the root not locally ·valid· exactly as case 2
//     does.
//
// All eight are unconditional: no verdict here can be overturned by anything in
// the rest of the document, which is what makes them decidable while the engine
// leaves most of the document undecided.
//
// # Charges at depth
//
// Cases 2 to 7 are charged against a DESCENDANT on the same terms as against
// the root (#790), and stay unconditional there. §3.3.4.6 clause 3.1 has a
// child assessed with respect to the ·governing element declaration· the
// parent's content model ·attributed· it to, so a child validate charges is one
// it was ·strictly assessed· against a declaration it really has, and its
// [validity] is invalid (§3.3.5.1 clause 1.1.1). Every ancestor up to the root
// is strictly assessed too — validate types a child only from a parent whose
// own governing type it determined — and clause 1.1.2 makes an ancestor with an
// invalid [[child]] invalid in turn, so the root's [validity] is invalid and the
// document is not valid, whatever the unassessed rest of it holds. A charge
// under a ·laxly assessed· ancestor, which clause 1.1.2 would NOT propagate,
// cannot arise: validate charges nothing at all below an element whose
// governing type it did not determine.
//
// The unconditionality has ONE exception, and it is the parser's, not
// validate's: a declaration whose {type table} parser's typeTableRepresentable
// WITHHELD looks tableless to validate, which then assesses the element against
// its DECLARED type where an <alternative> would ·conditionally select·
// another. A charge at or below such an element can be a false one. The
// exception is enumerable exactly because the withholding is — an <alternative>
// on the inline arm, an anonymous declared type behind a synthesized default, or
// a reference to ·xs:error· — and it shrinks as those shapes are mapped: a
// declaration whose table IS built makes validate decline instead of guess
// (#822, #821).
//
// Cases 3 to 6 rest on conditions validate checks rather than this file
// assuming them: the attribute half of cvc-complex-type is reached only where
// the governing type was determinable — the ·selected type definition·, or the
// ·instance-specified· one that ·overrides· it, and never a {type table}'s —
// only for attributes clause 2 quantifies over (the four xsi: names are
// excepted), and never where an {attribute wildcard} leaves an arm of the rule
// unevaluated. The content half adds its own: an element that is ·nilled·
// (clause 1 applies only where it is not, and cvc-elt clause 3.2.3.1 decides
// its [[children]] instead), and a {content type} whose shape
// xsd.Schema.ContentMatcher declines — {open content}, the nested repetition
// cvc-accept's own Note leaves non-deterministic, an all group holding an all
// group — each of which withholds clause 1.4 entirely rather than matching part
// of a sequence, and a root with no character information item [[child]] at
// all, whose ·initial value· cvc-elt clause 5.1 may take from a {value
// constraint} instead. The value charges add their own: a declaration whose
// {type definition} does not resolve to a simple type, and — the one that would
// otherwise reject every typeless attribute — a value.ValidateLexical error
// that is a fault of the type or of the backend rather than a verdict about the
// lexical (value.IsDatatypeVerdict). Each of those is a DECLINE inside
// validate, and a declined attribute charges nothing at all, so it cannot
// arrive here.
//
// Case 2 fires through an ASSEMBLED schema: producer.produceElement maps
// {abstract} from the top-level <element>'s abstract attribute (§3.3.2.1
// dcl.elt.common, #761), so a document declaring an abstract root reaches the
// cvc-elt charge here. Case 2 and case 3 can be charged for ONE root together —
// the cvc-elt branch keeps walking after charging, so an abstract root whose
// governing type is determinable is assessed for its attributes too — which is
// among the reasons decidedNotValid pins no violation count.
//
// # Why an EMPTY Result is not evidence of validity
//
// Everything else — a root that is declared and not abstract, so no violation
// is charged — is UNDECIDABLE IN BOTH DIRECTIONS and DECLINES. e-validity is a
// conjunction: local validity, AND no [[children]] or [[attributes]] whose
// [validity] is invalid, AND none attributed to a strict ·wildcard particle·
// and left notKnown. Being ·strictly assessed· at all (key-sva, §3.3.4.6) is
// itself a three-clause definition whose clauses 2 and 3 dispatch assessment
// recursively into every attribute and child, which Assess now follows only as
// far as it can type: a descendant is decided against the declaration its
// parent's content model ·attributed· it to, and where that declaration or its
// type is not determinable — a {type table}, a simple or unresolvable {type
// definition}, a name no top-level declaration matches under a wildcard and no
// xsi:type types either, a ·skipped· subtree — the element and everything below
// it is decided against nothing. Every clause of every rule this engine does not
// evaluate at all (cvc-type's simple half, assertions, cvc-elt clause 5.1) is
// undecided at every depth besides. The spec has no category for "this
// processor did not implement that check" stronger than notKnown, so an empty
// Result licenses no "valid" claim; equally it licenses no "invalid" one, so an
// expected-invalid case in this shape declines exactly as an expected-valid one
// does.
//
// The one shape that looks like case 1 and is not: a root with no top-level
// declaration whose xsi:type ·resolves·. Its ·instance-specified type
// definition· is its ·governing type definition· (key-governing-type-elem clause
// 8), so it is ·strictly assessed· against that type, cvc-assess-elt is not
// charged, and it lands in the undecidable bucket above on the same terms as any
// other root that charges nothing.
//
// # Why no false pass is possible
//
// Every "not valid" observation this lane emits comes from one of the eight
// charges above, each of which is unconditional. It never emits a "valid"
// observation at all: an empty Result declines. So the lane can record a
// still-failing gap for a suite-invalid case it cannot see the defect in, and
// for a suite-valid case whose root is undeclared or abstract, but it cannot
// score a pass on a document it did not really reject — at the root or at any
// depth, the charges being the same eight either way.
//
// Case 3's ATTRIBUTE clauses are the ones whose unconditionality depends on a
// schema COMPONENT being complete rather than on the instance alone: an
// under-reported {attribute uses} or {attribute wildcard} would make a valid
// attribute look unmatched (#414). validate declines a governing type whose two
// folds have not run (assess.go's attributePropertiesFolded), so the charge
// cannot reach this lane from such a type — and it is validate that must
// decline it, because assembleCase's decidability gate bounds what THIS lane
// assembles and says nothing about the library's other callers.
//
// Cases 7 and 8 carry their own, and validate declines rather than charging at
// every one: an identity constraint whose {selector} or {fields} fall outside
// the §3.11.6.2/§3.11.6.3 path subset, a field node whose ·governing type
// definition· was not determinable, a ·key-sequence· member pair validated
// against two different simple types, and — for cvc-id clause 1 alone — an
// [ID/IDREF table] any item of the subtree was declined for. Each is a DECLINE
// inside validate, so it cannot arrive here.
//
// Case 3's clause 1 and case 6 do not share that dependency, and are not
// declined with it: no finalize pass folds a {content type}, so a complex
// type's particle is whatever its producer built for it whether the type is
// named or anonymous (validate's governingType records the split).
//
// The cvc-assess-elt charge does carry one hazard of its own: a root is equally
// undeclared when an <import>/<include> the assembly did not follow took the
// declaring components with it, which is not the defect the suite meant to test.
// assembleCase's `Unfollowed() && perr != nil` conjunction bounds that for the
// SCHEMA lane, where the fabricated verdict shows up as a failed parse, and does
// not transfer here, where the parse succeeds and the charge lands anyway.
//
// Three further declines close the ways a NON-verdict could reach that
// comparison. An instance document that will not resolve or read is a recorded
// gap, on ReadDocument's own reasoning in schema.go — a reader limitation is not
// a well-formedness verdict. A validate.Result whose Err is non-nil is a walk
// that STOPPED on a source fault mid-document, so what it did or did not charge
// records how far the walk got and not what the document holds: the abstract-root
// branch keeps walking after charging, so a Result can carry BOTH a decidable
// violation and a truncated walk. And a violation set that is EMPTY, or that
// holds any rule outside the three enumerated, declines rather than being read
// as a verdict a later slice's wider Assess might charge under an
// approximation; the COUNT is not a condition, since one root can honestly
// carry several charges (see decidedNotValid).

// newInstanceExec builds the instance lane's executor. The strict backend is
// built once here, exactly as newSchemaExec does it: it maps all 20 primitives,
// so parser.Parse's internal builtin.Seed precondition holds for every case.
func newInstanceExec() executor {
	backend := strict.New()
	return func(c caseSpec) Status {
		return execInstanceCase(backend, c)
	}
}

// execInstanceCase decides one instanceTest case, or honestly declines it
// (Fail): it assembles the group's schema through the shared gate, assesses the
// instance document against it, and reads the assessment only where the two
// root-dispatch charges make the answer unconditional.
func execInstanceCase(backend value.Backend, c caseSpec) Status {
	if c.schemaDoc == "" {
		// The group declared no single schemaTest to take a schema from
		// (groupSchemaDocs): a case with no stated schema, not a case with an
		// invalid one.
		return Fail()
	}
	schema, decidable, perr := assembleCase(backend, c.schemaDoc, c.schemaExtraDocs)
	if !decidable || perr != nil {
		return Fail()
	}
	v, err := validate.New(schema, backend)
	if err != nil {
		// Only a nil schema reaches here, which a nil perr should have excluded;
		// declining rather than trusting it keeps the lane's verdicts honest.
		return Fail()
	}
	result, ok := assessInstance(v, c.doc)
	if !ok {
		return Fail()
	}
	if !decidedNotValid(result.Violations()) {
		return Fail()
	}
	// The only observation this slice can make is "not valid": an empty Result
	// declined above rather than reaching here as "valid".
	return decideAgreement(false, c.expect.wantsValid())
}

// assessInstance reads the instance document at doc and assesses it against v,
// or declines (ok false). It declines on three conditions, none of which is a
// verdict about the document: a document that will not resolve or that the
// reader rejects for any reason, a caller fault or a document malformed before
// its document element (xmlsrc.Validate's own error channel), and a walk that
// STOPPED on a source fault mid-document (validate.Result.Err), whose empty
// violation list records how far the walk got rather than what the document
// holds.
//
// The resolver is a loader.Dir rooted at the instance document's own directory,
// mirroring assembleCase, so a case fixture is reached the same way whichever
// lane reaches it.
func assessInstance(v *validate.Validator, doc string) (*validate.Result, bool) {
	resolver := loader.Dir(filepath.Dir(doc))
	rc, _, err := resolver.Resolve("", filepath.Base(doc))
	if err != nil {
		return nil, false
	}
	defer func() { _ = rc.Close() }() // read-only handle: close error cannot affect the verdict
	result, err := xmlsrc.Validate(v, rc, xmlsrc.WithURI(doc))
	if err != nil {
		return nil, false
	}
	if result.Err() != nil {
		return nil, false
	}
	return result, true
}

// These are the eight rules validate.Validator.Assess charges, and the whole of
// what this lane may read as a verdict. All eight are catalog IDs in their BARE
// form: the charged clause lives in the message text, not in a dotted rule ID,
// so matching the rule alone is the only stable match — and it is the right
// one, since a root failing ANY clause of any of the eight is not locally valid
// and so not valid (§3.3.5.1 e-validity clause 1.1.1.1).
const (
	ruleCvcAssessElt      xsderr.Rule = "cvc-assess-elt"
	ruleCvcElt            xsderr.Rule = "cvc-elt"
	ruleCvcComplexType    xsderr.Rule = "cvc-complex-type"
	ruleCvcComplexContent xsderr.Rule = "cvc-complex-content"
	ruleCvcAttribute      xsderr.Rule = "cvc-attribute"
	ruleCvcAu             xsderr.Rule = "cvc-au"

	ruleCvcIdentityConstraint xsderr.Rule = "cvc-identity-constraint"
	ruleCvcID                 xsderr.Rule = "cvc-id"
)

// decidableRules collects them for the membership test below, so growing the
// set is one edit and the test reads the same however long it gets — a chain of
// != comparisons silently admits a rule someone forgot to add to it.
var decidableRules = []xsderr.Rule{
	ruleCvcAssessElt, ruleCvcElt, ruleCvcComplexType, ruleCvcComplexContent,
	ruleCvcAttribute, ruleCvcAu, ruleCvcIdentityConstraint, ruleCvcID,
}

// decidedNotValid reports whether the violations one assessment charged
// establish that the document is not valid, unconditionally and whatever the
// unassessed rest of it holds.
//
// At least one violation, with every violation carrying one of the enumerated
// rules, is that evidence. Each enumerated rule is an unconditional "not valid"
// on its own, so a set of them is one too, however many. No violation at all
// leaves e-validity's conjunction unevaluated and declines (see the file
// comment); so does a set holding any OTHER rule — that is a shape this slice
// was not written against, and a later, wider Assess may charge one under a
// fail-open approximation, which reading as a verdict would be trusting it sight
// unseen.
//
// The count is not pinned at one because the clauses quantify independently —
// cvc-complex-type over the attributes present, the uses required and the
// ·defaulted attributes·; cvc-attribute and cvc-au over one attribute from two
// different {value constraint} sources — so one root can carry several charges
// honestly.
func decidedNotValid(violations []*xsderr.Error) bool {
	if len(violations) == 0 {
		return false
	}
	for _, v := range violations {
		if !slices.Contains(decidableRules, v.Rule) {
			return false
		}
	}
	return true
}
