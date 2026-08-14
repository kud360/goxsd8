package conformance

import (
	"path/filepath"

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
// # The only two outcomes this slice can DECIDE
//
// validate.Validator.Assess (#712) charges exactly two rules, both about the
// ·validation root· and nothing below it:
//
//  1. cvc-assess-elt (§3.3.4.6), when the root has no top-level element
//     declaration AND carries no xsi:type attribute. It then determines neither
//     a ·governing element declaration· nor a ·governing type definition·, so
//     key-sva clause 1 fails and §3.3.5.1's e-validity gives it notKnown rather
//     than valid — which under §5.2 ·strict wildcard validation·, the mode this
//     processor committed to in #712, is what "the invoking process ... will
//     otherwise report an error to its environment" names. The document is NOT
//     VALID, whatever else it holds.
//  2. cvc-elt (§3.3.4.3) clause 2, when the root's declaration was found and its
//     {abstract} is true. The root is ·strictly assessed· and locally INVALID,
//     so e-validity clause 1.1.1.1 fails and its [validity] is invalid. The
//     document is NOT VALID, whatever else it holds.
//
// Both are unconditional: neither verdict can be overturned by anything in the
// rest of the document, which is what makes them decidable while the engine
// assesses no attribute, no child and no type.
//
// Case 2 fires through an ASSEMBLED schema: producer.produceElement maps
// {abstract} from the top-level <element>'s abstract attribute (§3.3.2.1
// dcl.elt.common, #761), so a document declaring an abstract root reaches the
// cvc-elt charge here.
//
// # Why an EMPTY Result is not evidence of validity
//
// Everything else — a root that is declared and not abstract, so no violation
// is charged — is UNDECIDABLE IN BOTH DIRECTIONS and DECLINES. e-validity is a
// conjunction: local validity, AND no [[children]] or [[attributes]] whose
// [validity] is invalid, AND none attributed to a strict ·wildcard particle·
// and left notKnown. Being ·strictly assessed· at all (key-sva, §3.3.4.6) is
// itself a three-clause definition whose clauses 2 and 3 dispatch assessment
// recursively into every attribute and child — which Assess does not do: its
// recursive walk visits them to log and charges nothing. The spec has no
// category for "this processor did not implement that check" stronger than
// notKnown, so an empty Result licenses no "valid" claim; equally it licenses
// no "invalid" one, so an expected-invalid case in this shape declines exactly
// as an expected-valid one does. There is no narrower reading available: no
// attribute, child or type-checking capability exists anywhere in validate to
// base a content-shape heuristic on.
//
// The one shape that looks like case 1 and is not: a root with an xsi:type
// attribute but no top-level declaration. Assess DETECTS that attribute and
// never ·resolves· it (cvc-resolve-instance, §3.17.6.3, is unimplemented), so
// it withholds the cvc-assess-elt charge and returns no violation — landing the
// case in the undecidable bucket above, which is where it belongs until #716
// resolves the type for real.
//
// # Why no false pass is possible
//
// Every "not valid" observation this lane emits comes from one of the two
// charges above, each of which is unconditional. It never emits a "valid"
// observation at all: an empty Result declines. So the lane can record a
// still-failing gap for a suite-invalid case it cannot see the defect in, and
// for a suite-valid case whose root is undeclared or abstract, but it cannot
// score a pass on a document it did not really reject.
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
// violation and a truncated walk. And a violation count other than one, or a rule
// outside the two enumerated, declines rather than being read as a verdict a
// later slice's wider Assess might charge under an approximation.

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
	v, err := validate.New(schema)
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

// ruleCvcAssessElt and ruleCvcElt are the two rules validate.Validator.Assess
// charges, and the whole of what this lane may read as a verdict. Both are
// catalog IDs in their BARE form: cvc-elt's charged clause lives in its message
// text, not in a dotted rule ID, so matching the rule alone is the only stable
// match — and it is the right one, since a root failing ANY cvc-elt clause is
// not locally valid and so not valid (§3.3.5.1 e-validity clause 1.1.1.1).
const (
	ruleCvcAssessElt xsderr.Rule = "cvc-assess-elt"
	ruleCvcElt       xsderr.Rule = "cvc-elt"
)

// decidedNotValid reports whether the violations one assessment charged
// establish that the document is not valid, unconditionally and whatever the
// unassessed rest of it holds.
//
// Exactly one violation, carrying one of the two enumerated rules, is that
// evidence. Everything else is a decline: none at all leaves e-validity's
// conjunction unevaluated (see the file comment), and any other count or rule is
// a shape this slice was not written against — a later, wider Assess may charge
// under a fail-open approximation, and reading such a charge as a verdict would
// be trusting it sight unseen.
func decidedNotValid(violations []*xsderr.Error) bool {
	if len(violations) != 1 {
		return false
	}
	rule := violations[0].Rule
	return rule == ruleCvcAssessElt || rule == ruleCvcElt
}
