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
// validate.Validator.Assess charges exactly nine rules, about the ·validation
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
//  9. cvc-type (§3.3.4.4) clause 3.1, when an element's ·governing type
//     definition· is a Simple Type Definition and it carries an attribute
//     beyond the four xsi: names (3.1.1), an element information item [[child]]
//     (3.1.2), or — where it is not ·nilled· — an ·initial value· that is not
//     ·valid· per String Valid against that type (3.1.3) (#913). The element is
//     then not locally ·valid· with respect to its ·governing type definition·,
//     so case 3's chain applies unchanged.
//
// All nine are unconditional: no verdict here can be overturned by anything in
// the rest of the document, which is what makes them decidable while the engine
// leaves most of the document undecided. They are every "not valid" this lane
// observes; its "valid" is the one gated shape that charges none of them ("Why
// an EMPTY Result is evidence of validity for ONE shape only" below).
//
// # Charges at depth
//
// Cases 2 to 7 and case 9 are charged against a DESCENDANT on the same terms as
// against the root (#790, #913), and stay unconditional there. §3.3.4.6 clause 3.1 has a
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
// Every {type table} a declaration carries IS built (#851), so validate never
// guesses a tableless declaration's type: it ·conditionally selects· through the
// table, and withholds the element's ·governing type definition· only where the
// §3.12.6 required-subset evaluator declines one of the {test}s it had to try
// (validate/cta.go).
//
// Cases 3 to 6 and case 9 rest on conditions validate checks rather than this
// file assuming them: the attribute half of cvc-complex-type is reached only where
// the governing type was determinable — the ·selected type definition·, which
// a {type table} may have ·conditionally selected·, or the ·instance-specified·
// one that ·overrides· it — and only for attributes clause 2 quantifies over
// (the four xsi: names are excepted). The content half adds its own: an element
// that is ·nilled· (clause 1 applies only where it is not, and cvc-elt clause
// 3.2.3.1 decides its [[children]] instead), and a {content type} whose shape
// xsd.Schema.ContentMatcher declines — a nested repetition whose occurrence
// ranges admit more partitions than the walk carries cursors, an all group
// holding an all group — each of which withholds clause 1.4 entirely rather than
// matching part of a sequence. The value charges add their own: a declaration whose
// {type definition} does not resolve to a simple type, and — the one that would
// otherwise reject every value of a type the backend does not map — a
// value.ValidateLexical error that is a fault of the type or of the backend rather than
// a verdict about the lexical (value.IsDatatypeVerdict). Each of those is a DECLINE
// inside validate, and a declined attribute charges nothing at all, so it cannot arrive
// here.
//
// Case 2 fires through an ASSEMBLED schema: producer.produceElement maps
// {abstract} from the top-level <element>'s abstract attribute (§3.3.2.1
// dcl.elt.common, #761), so a document declaring an abstract root reaches the
// cvc-elt charge here. Case 2 and case 3 can be charged for ONE root together —
// the cvc-elt branch keeps walking after charging, so an abstract root whose
// governing type is determinable is assessed for its attributes too — which is
// among the reasons decidedNotValid pins no violation count.
//
// # Why an EMPTY Result is evidence of validity for ONE shape only
//
// In general an empty Result — no violation charged — is UNDECIDABLE IN BOTH
// DIRECTIONS and DECLINES. e-validity is a conjunction: local validity, AND no
// [[children]] or [[attributes]] whose [validity] is invalid, AND none
// attributed to a strict ·wildcard particle· and left notKnown. Being
// ·strictly assessed· at all (key-sva, §3.3.4.6) is itself a three-clause
// definition whose clauses 2 and 3 dispatch assessment recursively into every
// attribute and child, which Assess follows only as far as it can type: where a
// descendant's declaration or type is not determinable — a withheld {type
// table} selection, an unresolvable {type definition}, a name no top-level
// declaration matches under a wildcard and no xsi:type types either, a
// ·skipped· subtree — the element and everything below it is decided against
// nothing, and a few of validate's declines record nothing in
// Result.Unevaluated (validate.Unevaluated's own doc lists them).
// The spec has no category for "this processor did not implement that check"
// stronger than notKnown, so outside the shape below an empty Result licenses
// no "valid" claim; equally it licenses no "invalid" one, so an
// expected-invalid case declines exactly as an expected-valid one does.
//
// The one shape it DOES license "valid" for is an ASSESSED SUBTREE ROOT
// (assessedSubtreeRoot, subtreeroot.go, #1841), with content or without
// (#1855), and only where the walk recorded nothing in Result.Unevaluated.
// key-sva (§3.3.4.6) clauses 2 and 3 dispatch assessment into every attribute
// and child, and sic-e-outcome (§3.3.5.1) clause 1.1 recurses through them, so
// the gate holds EVERY element of the subtree to its conditions, not the root
// alone. It re-reads the instance and re-derives each child's ·attribution·
// with xsd.Schema.ContentMatcher, independently of the walk, and discharges at
// every element each clause the walk does not record deciding:
//
//   - key-sva clause 1 and cvc-elt clause 1: the root's declaration is the
//     top-level one its name resolves to, and every other element's is the
//     one the walk's childGoverning takes too: the {term} of the element
//     particle its parent's {content type} ·attributes· it to, carrying the
//     element's own name — the ·context-determined declaration· (§3.3.4.6
//     key-governing-ed clause 2) — or, for a child ·attributed to· a strict or
//     lax Wildcard or to the {open content}, the top-level declaration its
//     name ·resolves· to (clauses 3 and 4), admitted only where
//     cvc-complex-type clause 5 is vacuous for it (subtreeGate.resolvedChild).
//     A strict ·wildcard particle·'s child resolving to none, with no
//     xsi:type, is admitted with its subtree unread: the walk charges
//     e-validity clause 1.1.3 for its parent, so no empty Result carries it.
//     Every other attribution is refused: a skip Wildcard, whose subtree is
//     ·skipped· (key-sva clause 3.2, #1861); a lax Wildcard's or the {open
//     content}'s child resolving to none, the untyped element whose
//     [[children]] the walk leaves untyped (#1823, #1911); and a
//     ·substitution group· member cvc-accept clause 2.3.2 admitted, whose
//     {substitution group exclusions} and the head's {disallowed
//     substitutions} the walk never reads (#1932). No element of a subtree
//     whose Result is empty is therefore assessed against no type.
//   - cvc-elt clauses 2 to 6, at every element: {abstract} false, no xsi:nil,
//     no {type table} — so the ·selected type definition· is the {type
//     definition}, which the gate resolves itself — and no fixed {value
//     constraint}. An xsi:type is admitted where the gate resolves it and
//     xsd.Schema.ValidlySubstitutable answers that it ·overrides· the selected
//     type (clause 4, §3.3.4.2 key-overrides), which is the walk's decision
//     too; the gate then follows that type as the ·governing type definition·
//     (key-governing-type-elem clause 3) through every condition below. It
//     refuses an error from that predicate, the one silent exit of validate's
//     governingType (instanceOverride, #1093) an xsi:type reaches, and the two
//     answers xsd gives without reading the declaration's {disallowed
//     substitutions}, which the walk records nowhere
//     (subtreeGate.governingType, blockingUnread). {identity-constraint
//     definitions} are admitted, at every depth: clause 6
//     (cvc-identity-constraint, §3.11.4) is the walk's, which records each
//     check it declines in Result.Unevaluated, except the ·defaulted
//     attribute· field node "Cases 7 and 8" below names, which the gate
//     refuses. {nillable} is admitted, at the root too: with no xsi:nil
//     anywhere, clause 3.1 holds for a declaration whose {nillable} is false
//     and clause 3.2.1 ("E has no xsi:nil attribute information item") for one
//     whose {nillable} is true, and no element is ·nilled·. A default {value
//     constraint} is admitted, at the root too: clause 5.1 substitutes its
//     {lexical form} for the ·normalized value· of an element with neither
//     element nor character [[children]], and the walk assesses cvc-type over
//     that substituted value and settles 5.1.1 (validate's
//     contentCheck.assessed and contentCheck.defaultValid). Clause 2 is
//     refused outright below the root, where the walk charges it nowhere.
//   - cvc-type clause 2 (§3.3.4.4): a complex {type definition}'s {abstract}
//     is false, at the root too, since the walk decides that clause nowhere.
//   - cvc-type clause 3.1 (§3.3.4.4), for a Simple Type Definition: 3.1.1, the
//     element carries no attribute beyond namespace declarations and the four
//     xsi: names, and 3.1.2, it has no element [[children]] — the gate refuses
//     both, a second check on the walk's charges. 3.1.3 — the ·initial value·,
//     or clause 5.1's substitute, String Valid against the type — is the
//     walk's, the empty string of a content-less element included; its one
//     decline (validate's contentCheck.simpleTypeValue, String Valid withheld)
//     is RECORDED in Result.Unevaluated, as are the assertions-facet sites of
//     the type's closure, which validate records and never evaluates.
//   - cvc-elt clause 7 (cvc-id, §3.3.4.5): the walk's, at every depth. It
//     reads each element's and attribute's item into the [ID/IDREF table]
//     (validate's walk.idAttributes, walk.idDefaultedAttributes and
//     walk.idElement, through walk.idRecord) and charges both clauses once, at
//     the root (idTable.charge). Every item it cannot read is recorded through
//     walk.declineID except at two sites that set ids.declined and record
//     nothing, neither of which arises under this gate:
//     walk.idDefaultedAttributes' unresolved {attribute declaration}
//     ("Unreachable on a *xsd.Schema that exists"), which the gate refuses
//     besides, since each use's declaration must resolve; and walk.child's
//     "child its parent ·attributed to· nothing", which subtreeGate.child
//     keeps out by refusing every child its parent's xsd.Schema.ContentMatcher
//     attributes to nothing, the walk attributing each child through the same
//     matcher. Keep both refusals when editing the gate: the gate's reading of
//     cvc-id clause 1 rests on them. String Valid clause 3's ·declared entity
//     name· check (key-vde) is the walk's too, decided per ENTITY value by
//     walk.entitiesDeclared and recorded as an [Unevaluated] by its callers
//     where undecided (walk.declineAttribute, walk.declineDefaulted,
//     contentCheck.decline). What the gate still refuses is walkUnrecorded, for
//     the reason its doc gives: no element's value type — a Simple Type
//     Definition, or a simple {content type}'s {simple type definition} — and
//     no attribute use's {attribute declaration}.{type definition}, whether the
//     attribute is present or not, and no wildcard-resolved declaration's, has
//     a closure reaching it. Each use's declaration must resolve, so an
//     unresolvable {attribute declaration} is refused too.
//   - cvc-complex-type clause 2: every attribute beyond namespace declarations
//     and the four xsi: names matches an attribute use (2.1) or is
//     ·attributed to· the {attribute wildcard} (2.2, cvc-wildcard §3.10.4.1)
//     (#1860). Under skip it is ·skipped· and not assessed (key-sva clause
//     2.2); under lax or strict a name that ·resolves· to a top-level
//     declaration is assessed against it (key-sva clause 2.1), and one that
//     resolves to none under lax is not assessed. That declaration's {type
//     definition} is held to the walkUnrecorded exclusion a use's is. The gate
//     refuses a strict wildcard's name that resolves to none, which the walk
//     charges nothing for and records nothing of (#1912), and a resolved name
//     whose ·locally declared type· is not ·absent·, for clause 5 below
//     (subtreeGate.wildcardAttribute). Clause 2.1's cvc-attribute and cvc-au,
//     clause 2.2's cvc-attribute clauses 3 and 4 against the resolved declaration,
//     and clauses 3 and 4 — a {required} use the element lacks, and a ·defaulted
//     attribute· for each use it does not carry — are the walk's, which records
//     each check it withholds, on an element with no [[attributes]] as on any
//     other. A wildcard-resolved declaration supplies no ·defaulted attribute·
//     (key-dflt-att ranges over {attribute uses}).
//   - key-sva clause 2 for the xsi: attributes the gate admits: xsi:nil being
//     refused, xsi:type is the walk's (cvc-attribute clauses 3 and 5, charged
//     or recorded wherever it is no QName or ·resolves· to no type), and
//     xsi:schemaLocation and xsi:noNamespaceSchemaLocation are each ·valid·
//     against their built-in declaration's anyURI or list-of-anyURI type
//     (§3.2.7), whose lexical spaces admit every string (Datatypes §3.3.17).
//   - cvc-complex-type clause 5: vacuous for every child and attribute the
//     gate admits, which the walk never checks. An element particle's child
//     has its ·context-determined declaration·'s {type definition} for its
//     ·locally declared type· (key-ldt-elem case 2), which cvc-elt clause 4
//     already holds its ·governing type definition· to; an attribute matching
//     a use has that use's {attribute declaration}'s {type definition} for
//     both (key-ldt-att case 2, clause 2.1). A child ·attributed to· a strict
//     or lax Wildcard or to the {open content}, and an attribute resolved
//     through the {attribute wildcard}, is admitted only where its ·locally
//     declared type· is ·absent· (key-ldt-elem and key-ldt-att, walking the
//     {base type definition} chain, ·implicitly contained· ·substitution
//     group· members included: subtreeGate.locallyDeclared and
//     locallyDeclaredAttribute) or it has no ·governing type definition·.
//     e-validity clause 1.1.3 is the walk's, charged for a strict ·wildcard
//     particle·'s child resolving to none (validate's
//     walk.unresolvedStrictWildcardChild).
//   - cvc-complex-type clause 6: RECORDED, never decided. validate's
//     elementAssertions records an Unevaluated for every member of a governing
//     type's {assertions}, so an empty Unevaluated shows every one empty.
//   - cvc-complex-type clause 1 (§3.4.4.2): the walk's, for an element with no
//     [[children]] as for any other. An empty {content type} meets clause 1.1
//     wherever the walk charged no character or element [[child]]; a simple one
//     is clause 1.2's String Valid, decided and recorded as cvc-type clause
//     3.1.3 is above; an element-only or mixed one is clause 1.4
//     (cvc-complex-content), whose empty sequence is ·valid· only where the
//     particle is emptiable (cvc-particle, "possibly empty"). The gate refuses
//     a {content type} ContentMatcher declines and a child sequence it rejects
//     or does not accept, a second check on the walk's charge.
//
// Every element is then ·strictly assessed· against a declaration and a type
// the walk determined, and its [validity] is valid exactly where the walk
// charged nothing for it, its attributes or its descendants.
//
// The TRUST BOUNDARY is value.ValidateLexical: its verdict on every ·initial
// value· and every attribute value in the subtree is taken as Datatype Valid
// (Datatypes §4.1.4). The datatypes lane is what grounds that verdict; this
// lane does not re-check it. A value of xs:anySimpleType or xs:anyAtomicType
// never reaches it: Datatype Valid holds for every literal against a ·special·
// datatype, and validate decides so itself (#1788).
//
// One more hazard sits outside the clauses altogether: the schema must be the
// one the suite declared. The #1002 GAP(parser) retains elements vc:maxVersion
// excludes, so a schema document carrying version conditionals can assemble
// components a 1.1 processor must not see — VC/vc006.n1, suite-invalid, walks
// clean for exactly that reason. The gate declines any assembly one of whose
// documents carries an attribute in the versioning namespace, a conservative
// superset of that GAP. It declines a DOCTYPE too, whose DTD could default an
// attribute the gate does not see.
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
// Every "not valid" observation this lane emits comes from one of the nine
// charges above, each of which is unconditional. Its "valid" observation
// is an empty Result — no violation, no unevaluated record — on the gated
// shape, whose every applicable clause the section above names the decider of;
// every other empty Result declines. So the lane can record a still-failing
// gap for a suite-invalid case it cannot see the defect in, and for a
// suite-valid case outside that shape, but it cannot score a pass on a
// document it did not really reject, nor on one it did not really decide valid
// — at the root or at any depth, the charges being the same nine either way.
// The bound on the valid side is exactly as wide as the gate is correct and
// ValidateLexical is right: a clause the gate should have excluded and did
// not, or a Datatype Valid verdict the backend gets wrong, is where a false
// pass could come from.
//
// Case 3's ATTRIBUTE clauses are the ones whose unconditionality depends on a
// schema COMPONENT being complete rather than on the instance alone: an
// under-reported {attribute uses} or {attribute wildcard} would make a valid
// attribute look unmatched. Both properties are the spec's for every governing
// type validate can reach, an ANONYMOUS one included: §3.4.2.4 clause 3 and
// §3.4.2.5 clause 2 fold a declaration-owned type through the slot that owns it
// (xsd/ownedtypefold.go, #414), which is why validate reads them with no
// anonymity test at all. The completeness has to hold in the LIBRARY
// rather than here, because assembleCase's decidability gate bounds what THIS
// lane assembles and says nothing about the library's other callers.
//
// Cases 7 and 8 carry their own, and validate declines rather than charging at
// every one: an identity constraint whose {selector} or {fields} fall outside
// the §3.11.6.2/§3.11.6.3 path subset, a field node whose ·governing type
// definition· was not determinable, a ·key-sequence· member pair validated
// against two different simple types, and — for cvc-id clause 1 alone — an
// [ID/IDREF table] any item of the subtree was declined for. Each is a DECLINE
// inside validate, so it cannot arrive here. One case 7 input validate neither
// decides nor declines: a {fields} node that is a ·defaulted attribute·, whose
// [schema actual value] §3.11.4 clause 3's Note has play a part in a
// ·key-sequence·. validate's icCheck.fieldAttributes reads only the attributes
// the instance carries, so the ·key-sequence· goes short with nothing recorded:
// a key is charged clause 4.2.1 for it, and a unique or keyref member drops out
// of the ·qualified node set· uncharged. The "valid" side is closed by the
// gate, which refuses every element with a ·defaulted attribute· under a
// declaration carrying {identity-constraint definitions} (subtreeGate.complex).
// The clause 4.2.1 charge is not closed: it is a false rejection this lane
// reads as a "not valid" verdict, so a suite-invalid case of that shape can
// score a pass on a defect it did not see.
//
// Case 3's clause 1 and case 6 do not share that dependency, and are not
// declined with it: no finalize pass folds a {content type} at all, so a complex
// type's particle is whatever its producer built for it whether the type is
// named or anonymous.
//
// The cvc-assess-elt charge does carry one hazard of its own: a root is equally
// undeclared when an <import>/<include> the assembly did not follow took the
// declaring components with it, which is not the defect the suite meant to test.
// assembleCase's fabricatedRejection bounds that for the SCHEMA lane, where the
// fabricated verdict shows up as a failed parse, and does not transfer here,
// where the parse succeeds and the charge lands anyway.
//
// Three further declines close the ways a NON-verdict could reach that
// comparison. An instance document that will not resolve or read is a recorded
// gap, on ReadDocument's own reasoning in schema.go — a reader limitation is not
// a well-formedness verdict. A validate.Result whose Err is non-nil is a walk
// that STOPPED on a source fault mid-document, so what it did or did not charge
// records how far the walk got and not what the document holds: the abstract-root
// branch keeps walking after charging, so a Result can carry BOTH a decidable
// violation and a truncated walk. And a violation set that holds any rule
// outside the nine enumerated declines rather than being read as a verdict a
// later slice's wider Assess might charge under an approximation; the COUNT is
// not a condition, since one root can honestly carry several charges (see
// decidedNotValid). An EMPTY violation set declines unless the shape's
// conditions above all hold.

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
// instance document against it, and reads the assessment only where the answer
// is unconditional: a set of the nine decidable charges is "not valid", and an
// empty Result on an assessed subtree root (assessedSubtreeRoot) is "valid".
func execInstanceCase(backend value.Backend, c caseSpec) Status {
	if c.schemaDoc == "" {
		// The group declared no single schemaTest to take a schema from
		// (groupSchemaDocs): a case with no stated schema, not a case with an
		// invalid one.
		return Fail()
	}
	schema, report, decidable, perr := assembleCase(backend, c.schemaDoc, c.schemaExtraDocs)
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
	if violations := result.Violations(); len(violations) > 0 {
		if !decidedNotValid(violations) {
			return Fail()
		}
		return decideAgreement(false, c.expect.wantsValid())
	}
	// An empty Result is "valid" for the gated shape alone, and only where the
	// walk recorded no check it reached and did not perform.
	if len(result.Unevaluated()) > 0 {
		return Fail()
	}
	if !assessedSubtreeRoot(schema, report, c.doc) {
		return Fail()
	}
	return decideAgreement(true, c.expect.wantsValid())
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

// These are the nine rules validate.Validator.Assess charges, and the whole of
// what this lane may read as a verdict. All nine are catalog IDs in their BARE
// form: the charged clause lives in the message text, not in a dotted rule ID,
// so matching the rule alone is the only stable match — and it is the right
// one, since a root failing ANY clause of any of the nine is not locally valid
// and so not valid (§3.3.5.1 e-validity clause 1.1.1.1).
const (
	ruleCvcAssessElt      xsderr.Rule = "cvc-assess-elt"
	ruleCvcElt            xsderr.Rule = "cvc-elt"
	ruleCvcType           xsderr.Rule = "cvc-type"
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
	ruleCvcAssessElt, ruleCvcElt, ruleCvcType, ruleCvcComplexType, ruleCvcComplexContent,
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
