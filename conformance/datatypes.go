package conformance

import (
	"bufio"
	"encoding/xml"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file activates the datatypes lane (issue #15, extended by issues #57 and
// #80) by giving the datatypes entry of defaultLanes a real selector and
// executor. It touches nothing else in the runner (the #6 seam). It is
// package-internal conformance support: it exports nothing and no library code
// imports it.
//
// # The lexical cohort (issue #15, widened by issue #80)
//
// The lane claims the Microsoft datatype LEXICAL cases under
// msData/datatypes/{boolean,decimal,string,float,double,anyURI,hexBinary,
// base64Binary,duration,dateTime,dateTimeStamp,time,date,gYearMonth,gYear,
// gMonthDay,gDay,gMonth}NNN.xml. Each
// such schema declares an element of an UNRESTRICTED builtin primitive
// (xsd:boolean / xsd:decimal / xsd:string / xsd:float / xsd:double / xsd:anyURI /
// xsd:hexBinary / xsd:base64Binary / xsd:duration / xsd:dateTime and the six
// remaining seven-property date/time siblings xsd:time / xsd:date /
// xsd:gYearMonth / xsd:gYear / xsd:gMonthDay / xsd:gDay / xsd:gMonth —
// comp_foo directly, simpleTest via a facet-free restriction), so an instance is
// valid iff its content lies in that primitive's lexical space. That is exactly
// what value.Mapping.Parse decides, so the executor is a genuine, complete check:
// both polarities are decided for the right reason, and Parse really
// discriminates (boolean rejects "True"/"+1"/""; decimal rejects
// "1E2"/"INF"/"NaN"/"13.1513.561"/"ABCDEF"; float/double admit scientific
// notation and bare exponents like "1E2" and the special values "INF"/"+INF"/
// "-INF"/"NaN" case-sensitively, while rejecting "Infinity"/"nan"
// (xmlschema11-2.md §3.3.4.2/§3.3.5.2)). anyURI's lexical space is every Char*
// sequence — its Parse is the identity and rejects nothing, matching xs:string's
// permissiveness (§3.3.17.1/§3.3.17.2). hexBinary rejects odd-length and non-hex
// input (nt-hexBinary §3.3.15.2) and base64Binary rejects a non-multiple-of-four
// character count, misplaced '=' padding and a restricted-final-character
// violation (nt-Base64Binary §3.3.16.2); both count length in octets, not lexical
// characters (§4.3.1.3 clause 1.2). duration rejects a missing 'P', a bare "P"
// or "PT", a sign inside a field, out-of-order or 'T'-final fields
// (nt-durationRep §3.3.6.2). dateTime rejects a missing 'T' separator, an
// out-of-range month/hour/minute/second, a malformed timezone, and — beyond the
// grammar regex — a day-of-month that its month and (leap) year forbid, e.g.
// 2023-02-29 (con-dateTime-day/con-dateTime-dayValue §3.3.7.1, nt-dateTimeRep
// §3.3.7.2). The six seven-property siblings are thin lexical projections of the
// same model (§3.3.8–§3.3.14): each has its own nt-*Rep grammar (e.g. time drops
// the date fields, gYear keeps only the year) and, for date and gMonthDay, the
// day-of-month value constraint (con-date-dayValue §3.3.9.1, year-dependent;
// con-gMonthDay-dayValue §3.3.12.1, year-free so --02-29 is always valid) beyond
// the grammar regex; gDay/gMonth/gYear/gYearMonth carry no day-value rule.
//
// ## The context-dependent QName/NOTATION sub-cohort (issue #131)
//
// The lane also claims the QName lexical cases under msData/datatypes/QNameNNN.xml
// (and, for spec/mechanism parity, NOTATIONNNN.xml — of which the current checkout
// has none; NOTATION's cases are all facet-cohort under Facets/NOTATION). Unlike
// every other lexical member, QName's and NOTATION's lexical→value mapping is
// CONTEXT-DEPENDENT: a literal "prefix:local" resolves the prefix against the
// XML namespace bindings in scope where the literal occurs (§3.3.18; NOTATION's
// mapping is "as given for QName", §3.3.19). So execLexicalCase routes these
// (isContextDependent) to execContextualCase, which reads each comp_foo/simpleTest
// literal WITH the in-scope bindings the harness decodes from the instance
// (readQNameContexts builds an nsContext by tracking xmlns declarations down the
// ancestor chain — a raw literal's prefix is character content, not an XML name
// the decoder resolves) and threads that real value.Context to strict's Parse
// instead of the context-free path's nil. An unprefixed name binds to the default
// namespace (element-name semantics, no namespace when undeclared); a declared or
// reserved (only "xml", bound by definition — Namespaces in XML §3; "xmlns" is a
// declaration-attribute name, not a bindable prefix, WG ruling bugzilla 4053) prefix
// resolves; an unbound non-empty prefix or malformed grammar is a genuine
// rejection (cvc-datatype-valid §4.1.4), never a value fabricated with a guessed
// namespace (PRINCIPLES 19). This is a complete lexical check: QName/NOTATION have
// no spec-defined canonical form, and the declared-notation SCC of NOTATION
// (§3.3.19) is a Structures concern above this leaf mapping, out of scope here.
// The value's whiteSpace is fixed to collapse for both, applied by the shared
// normalizeWhiteSpace before Parse.
//
// ## The <item>-attribute sub-shape (issue #146)
//
// A few lexical-cohort instances use a different document shape than
// comp_foo/simpleTest: <data><item SOMITEM_DATATYPE_X="value"/></data>, with the
// tested value in an attribute (some documents carry two <item> children testing
// two attributes/types at once). These declare their schema OUT-OF-BAND in the
// suite's testGroup metadata, so the instance carries no
// noNamespaceSchemaLocation for readLexicalCase to follow; the schema is always the
// sibling datatypes.xsd, which types each SOMITEM_DATATYPE_* attribute directly as
// an UNRESTRICTED builtin primitive (SOMITEM_DATATYPE_DURATION as xsd:duration,
// _DATETIME as xsd:dateTime, _MONTHDAY as xsd:gMonthDay, _DATE as xsd:date, …). So
// this sub-shape has exactly the lexical cohort's semantics — validity is
// lexical-space membership (value.Parse) of each tested value — merely a different
// carrier. execLexicalCase falls back to execItemCase when the comp_foo decode
// declines, which reads the sibling datatypes.xsd (parsed, never hand-typed — same
// discipline as decodeTestedPrimitive) to resolve each attribute's primitive and
// ANDs parseOK across every recognized tested value. Only attributes typed as a
// seeded, directly-mapped primitive are decided (the same guard the comp_foo path
// applies); an attribute typed as a non-directly-mapped builtin (integer/derived-
// string family) is skipped, since Parse alone is not a complete check for those.
//
// dateTimeStamp (§3.4.28) is listed for forward parity but is the one member of
// this cohort whose Parse-only path is NOT a complete check, and it has ZERO cases
// in the current checkout so the gap is unexercised. Being a restriction of
// dateTime that fixes explicitTimezone=required, its validity also depends on the
// timezone being present — but execLexicalCase decides via parseDateTime alone,
// which does not enforce that facet (only the facet cohort's value.ValidateLexical
// does), so a tz-ABSENT dateTimeStamp literal would be false-ACCEPTED here. That is
// a fail-open risk, flagged at the datatypesCase regex, not a decided case today.
//
// # The facet cohort (issue #57, widened by issues #80, #81, #85, #106, #116, #123 and #124)
//
// The lane additionally claims the Microsoft *Facets* instance cases under
// msData/datatypes/Facets/<base>/<base>_<facet>NNN.xml where <base> is a
// strict-mapped primitive (string, decimal, float, double), an integer-family
// builtin (issue #81): integer, int, long, short, byte, unsignedInt/Long/Short/
// Byte, nonNegativeInteger, nonPositiveInteger, positiveInteger, negativeInteger,
// a derived string-family builtin: normalizedString, token (issue #85), the
// pattern-restricted string family language, Name, NCName, NMTOKEN (issue #106)
// and the NCName-derived ID, IDREF, ENTITY (issue #116), one of the length-facet-
// carrying primitives anyURI, hexBinary, base64Binary (issue #124), or a temporal
// primitive (issue #123): dateTime, time, date, gYearMonth, gYear, gMonthDay, gDay,
// gMonth, duration.
// Each such schema restricts <base> by one or more constraining facets
// (length/minLength/maxLength/pattern/enumeration on string; minInclusive/
// maxInclusive/minExclusive/maxExclusive/totalDigits/pattern/enumeration on
// decimal; the bound facets plus pattern/enumeration on float/double; pattern/
// enumeration/bounds on the temporal types). The
// float/double bound facets are checked over the PARTIAL order (NaN is
// incomparable to every value, so a NaN bound yields an empty value space and any
// bound comparison against NaN excludes — §2.2.3; §3.3.4.1 Note), which the
// existing boundFacet path already decides (incomparable ⇒ reject, spec-correct
// per §4.3.7.3–§4.3.10.3). The temporal bound facets ride that SAME incomparable ⇒
// reject path over the timeline's partial order (§3.3.6.3 for duration, e.g.
// P1M vs P31D; §3.2.7.3-style timezone-straddling incomparability for the
// date/time siblings), so an incomparable candidate is a genuine rejection, never
// a vacuous pass.
//
// The integer family is NOT a set of new primitives: xs:integer fixes decimal's
// fractionDigits to 0 and its lexical space to [\-+]?[0-9]+, and each narrowing
// adds only min/maxInclusive bounds (§3.4.13–§3.4.25); all thirteen share
// decimal's value space, order and identity (Datatypes §2.2.1 Identity note). So
// the generated builtin table already carries their fixed fractionDigits=0,
// fixed pattern and per-type bounds as EffectiveFacets, and strict's decimal
// mapping (walked to via the widest-space rule) parses their arbitrary-precision
// values unchanged — the same generic <restriction base="xsd:decimal"> pipeline.
// A fraction-point literal like "5.0" is rejected by the fixed pattern
// (cvc-pattern-valid §4.3.4.4), NOT cvc-fractionDigits-valid, since the pattern
// gate runs before the value facets; an out-of-range value (e.g. int 2147483648)
// is rejected by the type's own maxInclusive/minInclusive bound
// (cvc-max/minInclusive-valid §4.3.7.3/§4.3.10.3).
//
// The derived string family (normalizedString, token) is likewise NOT a set of
// new primitives: both share xs:string's value space and differ only by their
// fixed whiteSpace facet — normalizedString fixes it to replace, token to
// collapse (§3.4.1.1/§3.4.2.1) — with the chain token → normalizedString →
// string (§3.4.2/§3.4.1). So strict's string mapping (walked to via the
// widest-space rule) parses their values unchanged, and the leaf's overlaid
// whiteSpace (token's collapse replaces normalizedString's replace, the standard
// same-kind overlay of st-restrict-facets §3.16.6.4) normalizes the value once,
// as a pre-lexical step with no cvc-* rule (§4.1.4/§4.3.6.3), BEFORE the string
// lexical/length/pattern checks. A token instance carrying interior whitespace
// runs is collapsed, then length/pattern-checked on the normalized form; a value
// violating an own length/pattern/enumeration facet is rejected through the
// ordinary cvc-length/pattern/enumeration path.
//
// The wider string family (language, Name, NCName, NMTOKEN, issue #106; ID,
// IDREF, ENTITY, issue #116) extends
// this the same way: all derive from token (NCName via Name; ID/IDREF/ENTITY via
// NCName, §3.4.8/§3.4.9/§3.4.11 dt-ID/dt-IDREF/dt-ENTITY) and resolve to
// the xs:string primitive, so strict's string mapping governs them unchanged.
// They differ from normalizedString/token only by carrying an intrinsic pattern
// facet in the generated builtin table (language [a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*,
// NMTOKEN \c+, Name \i\c*, NCName's own [\i-[:]][\c-[:]]* ANDed across the
// Name→NCName step with Name's \i\c* — §4.3.4.2 xr-pattern, the cross-step pattern
// AND EffectiveFacets already realizes; ID/IDREF/ENTITY inherit NCName's pattern
// verbatim, adding none of their own) plus inherited whiteSpace=collapse. A
// value violating an intrinsic pattern (e.g. an NCName or ID with a colon) is
// rejected via cvc-pattern-valid before the own length/pattern/enumeration facets.
// ID-uniqueness and IDREF-target-existence are Structures-level checks (cvc-id,
// xmlschema11-1.md §3.3.4.5), NOT part of cvc-datatype-valid/cvc-facet-valid, so
// this cohort decides only per-value lexical+facet validity, exactly as it does
// for NCName. Unlike
// the string-content cohorts, the NMTOKEN and ID/IDREF cases carry the tested
// value in a named
// attribute of <foo> rather than its content, so readFacetsCase reads the value
// named by the enclosing xsd:attribute.
//
// Validity in this cohort
// depends on FACET checking, not
// just primitive lexical-space membership: an instance can be lexically valid
// yet facet-invalid (e.g. a 5-character string under length=4). The executor
// synthesizes the corresponding xsd.SimpleType (the seeded builtin as base, its
// primitive ancestor as {primitive type definition}, the schema's facet children
// as ownFacets) and decides validity through the
// now-complete facet pipeline (value.ValidateLexical, issue #45) — pattern
// (cvc-pattern-valid §4.3.4.4), lexical mapping (cvc-datatype-valid §4.1.4),
// then the value facets cvc-enumeration-valid (§4.3.5.4),
// cvc-min/maxInclusive/Exclusive-valid (§4.3.7–4.3.10), cvc-totalDigits-valid
// (§4.3.11.3) and cvc-length/minLength/maxLength-valid (§4.3.1.3–4.3.3.3). This
// is the facet-invalid-but-lexically-valid class the original #15 landing could
// not discriminate with Parse alone.
//
// The executor OWNS facet applicability (cos-applicable-facets §4.1.5): it
// attaches a facet to the synthesized leaf only when builtin's applicable-facet
// metadata says it applies to the base primitive, so an instance-level facet
// violation always returns an *xsderr.Error through the normal path and the
// panic precondition ValidateLexical documents is never reached. A case pairing
// an inapplicable facet with a primitive (a schema-construction error, not an
// instance validity case) is declined rather than fed through and crashed.
//
// # The precisionDecimal cohort (issue #135)
//
// The lane additionally claims the Saxon precisionDecimal instance cases under
// saxonData/PDecimal/pdecimalNNN.{vK,nK}.xml (discovered via the auxiliary
// extra-suite.xml index, runner.go — the W3C suite moved the precisionDecimal sets
// out of suite.xml when the type was withdrawn from XSD 1.1 but retained as a
// Working Group Note; goxsd8 implements it as an implementation-defined primitive,
// strict #115, maxScale/minScale #133). Unlike every prior cohort, the instance
// shape is a <doc> root with REPEATED <e value="…"/> children, all validated
// against ONE tested type — the attribute value's type in the sibling
// pdecimalNNN.xsd (schema-out-of-band, no noNamespaceSchemaLocation: derived from
// the case-prefix filename, like the #146 item shape). execPDecimalCase synthesizes
// that leaf (precisionDecimal as {primitive type definition}, its schema facets as
// ownFacets) once and ANDs value.ValidateLexical over every literal, so the
// instance is valid iff EVERY literal is — the suite's whole-document polarity.
//
// precisionDecimal's spec-exact facet semantics fall out of the existing pipeline
// with no new value code: totalDigits vacuously passes zero AND the specials
// (value.TotalDigits reports 1, xsd-precisionDecimal.md §4.1, a rule DISTINCT from
// decimal's — the pD value model owns the zero special-case, not this lane); the
// four bound facets ride the boundFacet incomparable⇒reject path over the PARTIAL
// order, so NaN — incomparable with every value including itself (§3.1) — fails
// EVERY bound symmetrically (cvc-min/maxInclusive/Exclusive-valid §4.3.7–4.3.10);
// maxScale/minScale skip the specials' absent ·scale· (#133, cvc-maxScale/minScale-
// valid §4.2.3/§4.3.3); enumeration matches value-space "equal or identical" on
// ·numericalValue· (10 == 1.0E1; NaN matches a NaN member via identity, §4.3.5.4),
// via the shared enumMatch Identical-then-Eq path; pattern checks the literal
// unchanged. whiteSpace=collapse (fixed, §3.3) is inherited from the seeded
// precisionDecimal builtin, so the .v2 leading/trailing-whitespace instances
// normalize before the lexical check.
//
// Only the directly-mapped and SINGLE-STEP restriction shapes are decided:
// pdecimal001–008,010 (attribute typed xs:precisionDecimal, or a named simpleType
// restricting it with one facet kind). The two-step chain (pdecimal016, a
// restriction of a restriction), the list variety (pdecimal019, <list itemType>)
// and the union variety (pdecimal020, <union memberTypes>) are DECLINED by
// decodePDecimalSchema — a synthesized single leaf cannot carry a multi-step
// effective-facet set nor a list/union variety — and honestly recorded as gaps
// (Fail) rather than mis-decided. One further gap is a suite quirk, not a shape
// limit: pdecimal006.n2 ("NaN" against a NaN-bearing enumeration) is spec-VALID
// (identity match) but suite-declared invalid, so the spec-correct verdict records
// a Fail against it (see execPDecimalCase). The IBM ibmData/D3_3_4 precisionDecimal
// shape (several named types per schema, each tested by a dedicated element) is
// NOT claimed here — its multi-type document shape is a distinct, larger executor,
// left to a follow-up issue; those instance cases route to the inert instance lane
// as recorded gaps meanwhile.
//
// # Still deferred
//
// Facets over QName (the Facets/QName dir) are now PARTIALLY claimed (issue #125):
// the length/minLength/maxLength cases (vacuous per clause 1.3) and the pattern case
// decide through the ordinary pipeline, but the enumeration cases are declined
// pending schema-declaring-context threading — a QName enumeration member's prefix
// must resolve against the schema's in-scope bindings, which xsd.Facet/
// value.newEnumFacet cannot yet carry (value/facets.go flags this) — an honest gap
// buildOwnFacets records, promised as a warden-gated follow-up issue. Facets over
// NOTATION (the Facets/NOTATION dir) are NOT claimed at all: unlike every other
// cohort member their fixtures use a two-step restriction through a locally-named
// simpleType, paired with <xsd:notation> component declarations, with the tested
// value in an attribute of a complexType (not the single-step xsd:NOTATION
// restriction of a <foo> element's content this cohort decodes) — a shape
// readFacetsCase/decodeRestriction does not model — deferred to its own follow-up
// issue. xsd:boolean facets (no Facets dir exists for it), the plural list-typed
// dirs (IDREFS, NMTOKENS), the NIST corpus, and list/union varieties remain out of
// scope until their backends land.
// string_pattern002_1031.i (issue #146) falls under that list-variety exclusion:
// its Facets/string/string_pattern002.xml restricts via <xsd:list itemType="Hex"/>
// (a per-token pattern facet decided by cvc-datatype-valid §4.1.4 clause dv_list,
// unimplemented here), and its instance shape (a <Xml xmlns="TestNamespace"> root
// with three <Hex> list-valued children) does not match readFacetsCase's single-
// <foo> shape either, so it is honestly declined (Fail), never false-accepted.
// Within the integer family, the odd
// multi-element cases (e.g. Facets/int/test111092.xml, two named restriction
// steps under distinct elements) do not fit the single-<foo> instance shape and
// fall through to the instance lane as recorded gaps. boolean018 (a list-of-
// boolean + enumeration on a user-defined "myList") and anyURI011 (a list-of-
// anyURI, whose simplefooType restricts the "myList" list type) resolve to a
// non-seeded type and are honestly recorded as gaps (Fail); they flip only when
// list variety is reachable here. time_minInclusive006_1163.i (issue #123) is a
// recorded gap for a different reason: its instance file carries no
// xsi:noNamespaceSchemaLocation (a defect in that one suite file), so
// readFacetsCase cannot resolve its schema and declines it (Fail) rather than
// guessing the base — an honest decline, not a false accept. The anyURI
// Facets/anyURI/anyURI_a*.xml and anyURI_b*.xml cases (issue #124) are honest
// gaps for the same class of reason: the a* instances carry the value in a
// namespace-qualified <bar> reached via xsi:schemaLocation (not
// noNamespaceSchemaLocation), and the b* instances hold zero or several <foo>
// leaves (b001 puts the values in repeated <bar> children; b006 repeats many
// <foo> values against one enumeration, a list-style shape), so neither fits the
// single-<foo> instance shape readFacetsCase decodes — all are declined (Fail,
// readFacetsCase requires exactly one <foo>) rather than mis-read as an empty or
// last-wins tested value. Only the anyURI/hexBinary/base64Binary length/
// minLength/maxLength/enumeration cases in the canonical <test><foo> shape are
// decided here.

// synthNS namespaces the anonymous leaf types the facet cohort synthesizes. It
// is deliberately outside xsd.XMLSchemaNS so a synthesized leaf is never mistaken
// for a backend-mapped builtin (the widest-space facet checks resolve to its
// primitive base's mapping, never the leaf's own).
const synthNS = "urn:goxsd8:conformance:facets"

// datatypesCase matches an instance case in the lexical cohort. QName and
// NOTATION are the context-dependent members (their Parse resolves a prefix
// against the in-scope namespace bindings); execLexicalCase routes them to
// execContextualCase. NOTATION has ZERO plain lexical cases in the current
// checkout (all its cases are facet-cohort under Facets/NOTATION), so it is
// listed for parity and exercised only by QName today.
//
// GAP(datatypes): dateTimeStamp (§3.4.28) is listed but has ZERO cases in the
// current W3C checkout (no msData/datatypes/dateTimeStampNNN.xml), so the
// alternative is inert today. Unlike every other cohort type, its Parse-only path
// is NOT a complete check: dateTimeStamp fixes explicitTimezone=required, but
// execLexicalCase decides validity purely via parseDateTime (parseOK), which does
// not enforce the timezone (that check lives only in the facet cohort's
// value.ValidateLexical). So a
// tz-ABSENT dateTimeStamp literal would be FALSE-ACCEPTED here — a fail-open gap,
// currently unexercised because no such case exists. Should the suite ever add a
// tz-absent dateTimeStamp case, it must move to the facet cohort (or execLexicalCase
// must run the explicitTimezone facet) rather than be decided by this Parse-only path.
var datatypesCase = regexp.MustCompile(`msData/datatypes/(boolean|decimal|string|float|double|anyURI|hexBinary|base64Binary|duration|dateTime|dateTimeStamp|time|date|gYearMonth|gYear|gMonthDay|gDay|gMonth|QName|NOTATION)[0-9]+\.xml$`)

// facetsBaseTypes lists the builtin datatypes whose Facets-cohort restrictions
// the lane decides: the strict-mapped primitives (string/decimal/float/double),
// the integer family (xs:integer and its twelve narrowings, issue #81), the
// derived string family — normalizedString/token (issue #85), the
// pattern-restricted language/Name/NCName/NMTOKEN (issue #106) and the
// NCName-derived ID/IDREF/ENTITY (issue #116) — the length-facet-carrying
// primitives anyURI/hexBinary/base64Binary (issue #124), and the temporal
// primitives dateTime/time/date/gYearMonth/gYear/gMonthDay/gDay/gMonth/duration
// (issue #123).
// Every
// integer-family type is a facet restriction of xs:decimal (§3.4.13–§3.4.25) that
// shares decimal's value space, order and identity (Datatypes §2.2.1 Identity
// note), so strict's decimal mapping governs it unchanged; the derived string
// types are facet restrictions of xs:string (chain token → normalizedString →
// string, §3.4.1/§3.4.2; language/Name/NMTOKEN off token, NCName off Name,
// ID/IDREF/ENTITY off NCName — §3.4.8/§3.4.9/§3.4.11) that
// share string's value space and differ only by inherited whiteSpace and their
// intrinsic pattern facets, so strict's string mapping governs them unchanged. The
// nine temporal types are themselves primitives (§3.3.6–§3.3.14), each mapped
// directly by strict, so their Facets restrictions resolve to their own primitive
// mapping (the string/numeric cohorts' widest-space pattern) with no derivation
// walk. Their applicable facets (cos-applicable-facets §4.1.5) admit pattern,
// enumeration and the four bound facets — exactly the kinds the present suite's
// temporal Facets schemas carry (no length, no explicitTimezone cases exist), all
// already in facetKinds — and the bound facets are decided over the temporal
// primitives' PARTIAL timeline order, where an incomparable candidate-vs-bound
// comparison (common for duration, §3.3.6.3) is a real rejection, exactly as the
// existing boundFacet path already decides it (cvc-*Inclusive/Exclusive-valid
// §4.3.7.3–§4.3.10.3; duration lacks explicitTimezone per §4.1.5, immaterial here
// since no such case exists). anyURI, hexBinary and base64Binary (issue #124) are
// likewise primitives strict maps directly (#82, #83), so their Facets restrictions
// resolve to their own mapping. All three are unordered (ordered=false,
// §3.3.15.3/§3.3.16.3/§3.3.17.3) and share xs:string's applicable-facet set —
// length/minLength/maxLength/pattern/enumeration (cos-applicable-facets §4.1.5), all
// in facetKinds — with NO bound facets. The length facets measure the value's
// intrinsic size, which is unit-aware per type (§4.3.1.3 clauses 1.1/1.2): rune count
// for anyURI (like string) but decoded-OCTET count for the two binary types, a split
// value.Lengthed already realizes through each mapping's Len() (anyURIVal.Len over
// runes; hexBinaryVal/base64BinaryVal.Len over decoded []byte), so no length-unit
// special-casing is needed here.
//
// xsd:QName (issue #125) is likewise a primitive strict maps directly (#131), so its
// Facets restrictions resolve to its own CONTEXT-DEPENDENT mapping. Its applicable
// facets (cos-applicable-facets §4.1.5) are length/minLength/maxLength/pattern/
// enumeration/whiteSpace/assertions — the same shape as string — but the cohort
// admits QName with two carve-outs. First, length/minLength/maxLength ARE applicable
// (schema-valid to declare), yet §4.3.1.3/§4.3.2.3/§4.3.3.3 clause 1.3 makes EVERY
// value facet-valid when {primitive type definition} is QName — a
// deprecated-but-still-legal no-op — which value.lengthFacet's lengthExemptPrimitive
// exemption (#130) already realizes, so those cases decide through the ordinary
// pipeline as vacuous passes with no QName-specific code here. Second, enumeration
// over QName compares §3.2.18 {namespace name, local name} tuples, so a prefixed enum
// member (e.g. "foo:fo") must resolve against the DECLARING SCHEMA's in-scope
// bindings — a context xsd.Facet/value.newEnumFacet cannot yet carry (value/facets.go
// flags exactly this hazard), so a QName case carrying an enumeration facet child is
// explicitly declined by buildOwnFacets as an honest recorded gap (see the issue #125
// GROUNDING comment; the schema-declaring-context threading is a warden-gated
// follow-up). QName's TESTED literals, by contrast, resolve their prefixes against the
// INSTANCE's bindings, which execFacetsCase threads to value.ValidateLexical as a real
// value.Context: strict's parseQName rejects a nil context UNCONDITIONALLY, even for an
// unprefixed name (qname.go resolveQNameLexical), so this threading is required for
// every claimed QName case (all 11 carry unprefixed literals like "foofo"/"abc"), not
// only hypothetical prefixed ones. NOTATION is deliberately NOT admitted (its fixtures
// use an incompatible two-step/locally-named-type shape — see "Still deferred").
// No new backend mapping is introduced in any case. ENTITY has no Facets cases in the
// current W3C checkout (no msData/datatypes/Facets/ENTITY dir); it is listed for
// spec parity and mechanism reuse (a zero-case regex alternative is harmless), so
// a future suite update carrying such cases is decided with no further code change.
// The list feeds both the
// directory and the filename-prefix alternation of facetsCase.
const facetsBaseTypes = `string|normalizedString|token|language|Name|NCName|NMTOKEN|` +
	`ID|IDREF|ENTITY|` +
	`anyURI|hexBinary|base64Binary|` +
	`QName|` +
	`decimal|float|double|` +
	`integer|int|long|short|byte|` +
	`unsignedInt|unsignedLong|unsignedShort|unsignedByte|` +
	`nonNegativeInteger|nonPositiveInteger|positiveInteger|negativeInteger|` +
	`dateTime|time|date|gYearMonth|gYear|gMonthDay|gDay|gMonth|duration`

// facetsCase matches an instance case in the facet cohort: an MS Facets instance
// under a facetsBaseTypes directory whose filename prefixes the same type name
// (e.g. Facets/int/int_maxInclusive001.xml).
var facetsCase = regexp.MustCompile(
	`msData/datatypes/Facets/(` + facetsBaseTypes + `)/(` + facetsBaseTypes + `)_[A-Za-z]+[0-9]+\.xml$`)

// pdecimalCase matches a precisionDecimal instance case in the Saxon PDecimal
// cohort (issue #135): saxonData/PDecimal/pdecimalNNN.{vK,nK}.xml. Each such
// document is a <doc> root with repeated <e value="…"/> children, all validated
// against ONE type — the attribute value's type declared in the sibling
// pdecimalNNN.xsd (either xs:precisionDecimal directly or a single-step
// restriction of it). The executor (execPDecimalCase) declines a case whose type
// is a multi-step chain, list or union variety (pdecimal016/019/020), which this
// synthesized-single-leaf model cannot decide — an honest recorded gap, never a
// mis-decided one. The IBM ibmData/D3_3_4 precisionDecimal shape (several named
// types per schema) is NOT claimed here and remains a deferred follow-up.
var pdecimalCase = regexp.MustCompile(`saxonData/PDecimal/pdecimal[0-9]+\.[vn][0-9]+\.xml$`)

// selectsDatatypes claims the instance cases of the lexical, facet and
// precisionDecimal cohorts. It is a cheap path predicate; the executor does the
// real document reading.
func selectsDatatypes(c caseSpec) bool {
	if c.kind != kindInstance {
		return false
	}
	doc := filepath.ToSlash(c.doc)
	return datatypesCase.MatchString(doc) || facetsCase.MatchString(doc) || pdecimalCase.MatchString(doc)
}

// newDatatypesExec builds the lane's executor: it Seeds the builtins once (the
// M3 composition step) from builtin/strict — which maps all 20 primitives, so
// builtin.Seed's all-primitives precondition holds — and captures the strict
// backend plus the seeded symbol table in the returned closure.
func newDatatypesExec() executor {
	// strict.New() maps all 20 builtin primitives (decimal/precisionDecimal/
	// boolean/string/anyURI/float/double/hexBinary/base64Binary/duration/dateTime
	// plus the six seven-property siblings time/date/gYearMonth/gYear/gMonthDay/
	// gDay/gMonth and QName/NOTATION), which is exactly builtin.Seed's precondition,
	// so it feeds Seed directly. The lane claims boolean/decimal/string/float/
	// double/anyURI/hexBinary/base64Binary/duration/dateTime/time/date/gYearMonth/
	// gYear/gMonthDay/gDay/gMonth and the context-dependent QName/NOTATION (#131)
	// (lexical cohort) and string/decimal/float/double plus the integer and
	// derived-string (normalizedString/token, #85; language/Name/NCName/NMTOKEN,
	// #106; ID/IDREF/ENTITY, #116) families, anyURI/hexBinary/base64Binary (#124)
	// and the temporal primitives (#123) (facet cohort) cases
	// (float/double added in #80, anyURI in #82, hexBinary/base64Binary in #83,
	// duration in #84, dateTime in #103, the seven-property siblings in #109),
	// every one of which resolves (directly or via a base ancestor) to a strict
	// mapping.
	strictBackend := strict.New()

	// Seed proves the strict backend satisfies the all-primitives precondition —
	// else it returns a typed *builtin.MissingPrimitivesError naming the gaps — and
	// yields the builtin components; the executor confirms a claimed case's type is
	// a seeded builtin before validating it. strict maps all 20 primitives by
	// construction (guarded by TestDatatypesBackendSeeds), so a Seed error here is a
	// programming error, not a runtime condition — panic rather than drop it.
	types, err := builtin.Seed(strictBackend)
	if err != nil {
		panic("conformance: datatypes lane backend must Seed by construction: " + err.Error())
	}
	sym := make(map[xsd.QName]*xsd.SimpleType, len(types))
	for _, t := range types {
		sym[t.Name()] = t
	}

	return func(c caseSpec) Status {
		doc := filepath.ToSlash(c.doc)
		if pdecimalCase.MatchString(doc) {
			return execPDecimalCase(strictBackend, sym, c)
		}
		if facetsCase.MatchString(doc) {
			return execFacetsCase(strictBackend, sym, c)
		}
		return execLexicalCase(strictBackend, sym, c)
	}
}

// execLexicalCase decides a lexical-cohort case: an instance is valid iff every
// tested leaf value lies in the tested primitive's lexical space (value.Parse).
// The comp_foo/simpleTest shape is decided directly; the alternate
// <data><item ATTR="value"/></data> shape (issue #146), which declares its schema
// out-of-band and so has no noNamespaceSchemaLocation for readLexicalCase to
// resolve, falls through to execItemCase rather than being mis-declined here.
func execLexicalCase(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	prim, values, ok := readLexicalCase(c.doc)
	if !ok {
		return execItemCase(backend, sym, c)
	}
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: prim}
	if _, seeded := sym[qn]; !seeded {
		return Fail()
	}
	m, mapped := backend.Mapping(qn)
	if !mapped {
		return Fail()
	}
	// Context-dependent primitives (QName/NOTATION, §3.3.18/§3.3.19) resolve a
	// prefix against the in-scope namespace bindings at the literal, so they take
	// the contextual path with a real value.Context rather than the nil-context
	// value scan below (whose whiteSpace-only reading suffices for the
	// context-free primitives).
	if isContextDependent(prim) {
		return execContextualCase(m, prim, c)
	}
	observedValid := true
	for _, v := range values {
		if !parseOK(m, prim, v, nil) {
			observedValid = false
			break
		}
	}
	if observedValid == c.expectValid {
		return Pass()
	}
	return Fail()
}

// isContextDependent reports whether prim's lexical→value mapping depends on the
// in-scope XML namespace bindings at the literal (§3.3.18 for QName, §3.3.19 for
// NOTATION, whose lexical mapping is "as given for QName"). These are the only
// primitives whose Parse consumes a value.Context; every other cohort member
// maps context-free, so the harness passes them a nil context (strict's Parse
// tolerates nil for those). NOTATION carries no plain lexical case in the current
// W3C checkout (its cases are all facet-cohort under Facets/NOTATION), so it is
// listed for spec/mechanism parity and exercised only by QName today.
func isContextDependent(prim string) bool {
	return prim == "QName" || prim == "NOTATION"
}

// execContextualCase decides a lexical-cohort case for a context-dependent
// primitive (QName/NOTATION): each tested leaf literal resolves its prefix
// against the in-scope XML namespace bindings at the element carrying it
// (readQNameContexts), and the instance is valid iff every leaf lies in the
// primitive's lexical space under that REAL context. An unbound prefix, an
// unprefixed name that is not an NCName, or malformed grammar is a genuine
// rejection through strict's Parse (cvc-datatype-valid §4.1.4), never a value
// fabricated with a guessed namespace (PRINCIPLES 19). A case whose instance
// shape does not decode is declined (Fail), an honest recorded gap.
func execContextualCase(m value.Mapping, prim string, c caseSpec) Status {
	lits, ok := readQNameContexts(c.doc)
	if !ok {
		return Fail()
	}
	observedValid := true
	for _, lit := range lits {
		if !parseOK(m, prim, lit.value, lit.ctx) {
			observedValid = false
			break
		}
	}
	if observedValid == c.expectValid {
		return Pass()
	}
	return Fail()
}

// execItemCase decides a lexical-cohort case in the <data><item ATTR="value"/>
// shape (issue #146): each <item> carries a tested value in a SOMITEM_DATATYPE_*
// attribute whose builtin primitive the sibling datatypes.xsd declares. The
// instance is valid iff every tested value lies in its primitive's lexical space
// (value.Parse), AND across every recognized attribute of every <item> — mirroring
// execLexicalCase's polarity, since any invalid tested value makes the whole
// instance invalid. Only attributes whose declared primitive is a seeded,
// backend-mapped builtin are decided (the same sym/backend.Mapping guards the
// comp_foo path uses); an attribute typed as a non-directly-mapped builtin (e.g.
// the integer/derived-string families, whose validity needs the facet pipeline,
// not Parse alone) is skipped, not guessed. A case whose shape does not decode,
// whose sibling schema is unreadable, or that references no recognized attribute
// at all is declined (Fail, an honest recorded gap) rather than mis-decided.
func execItemCase(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	lits, ok := readItemCase(c.doc)
	if !ok {
		return Fail()
	}
	observedValid := true
	decided := false
	for _, lit := range lits {
		qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: lit.prim}
		if _, seeded := sym[qn]; !seeded {
			continue
		}
		m, mapped := backend.Mapping(qn)
		if !mapped {
			continue
		}
		decided = true
		if !parseOK(m, lit.prim, lit.value, nil) {
			observedValid = false
			break
		}
	}
	if !decided {
		return Fail()
	}
	if observedValid == c.expectValid {
		return Pass()
	}
	return Fail()
}

// execFacetsCase decides a facet-cohort case: it synthesizes the schema's
// faceted leaf type and runs the tested value through the real facet pipeline
// (value.ValidateLexical). A case whose base is not strict-mapped, whose schema
// cannot be read, or that pairs an inapplicable facet with its primitive is
// declined (Fail, a recorded gap) rather than mis-decided or crashed.
func execFacetsCase(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	raw, base, children, ctx, ok := readFacetsCase(c.doc)
	if !ok {
		return Fail()
	}
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: base}
	builtinType, seeded := sym[qn]
	if !seeded {
		return Fail()
	}
	// Authoritative cohort guard: the leaf's governing mapping (its own or a base
	// ancestor's, widest-space rule st-restrict-facets §3.16.6.4) must be supplied
	// by the strict backend, so ValidateLexical parses through a spec-exact mapping.
	// Directly-mapped primitives (string/decimal/float/double) satisfy this at the
	// first step; the integer family resolves to its xs:decimal ancestor (#81).
	if !strictGoverns(backend, builtinType) {
		return Fail()
	}
	ownFacets, ok := buildOwnFacets(base, children)
	if !ok {
		return Fail()
	}
	leaf, err := xsd.NewSimpleType(xsderr.Loc{},
		xsd.QName{Space: synthNS, Local: base + "-facets"},
		xsd.Atomic{Primitive: primitiveOfType(builtinType)}, builtinType, ownFacets, nil)
	if err != nil {
		return Fail()
	}
	_, verr := value.ValidateLexical(backend, leaf, raw, ctx)
	observedValid := verr == nil
	if observedValid == c.expectValid {
		return Pass()
	}
	return Fail()
}

// execPDecimalCase decides a Saxon PDecimal cohort case (issue #135): every
// tested <e value="…"/> literal is validated against ONE synthesized leaf — the
// precisionDecimal primitive restricted by the attribute value's declared facets
// — through the real facet pipeline (value.ValidateLexical). The instance is
// valid iff EVERY literal is, mirroring the suite's whole-document polarity (a
// .nK document carries at least one out-of-space or facet-invalid literal). The
// pipeline already realizes precisionDecimal's spec-exact semantics: NaN fails
// every bound facet (partial order, incomparable ⇒ excluded, §3.1), totalDigits
// vacuously passes zero and the specials (value.TotalDigits reports 1, §4.1),
// maxScale/minScale skip the specials' absent ·scale· (#133), and enumeration is
// value-space "equal or identical" on ·numericalValue· (10 == 1.0E1; NaN matches
// NaN via identity, §4.3.5.4). A case whose type is not a directly-mapped or
// single-step precisionDecimal restriction (a multi-step chain, list or union —
// pdecimal016/019/020), whose schema cannot be read, or that pairs an
// inapplicable facet with the primitive is declined (Fail, a recorded gap).
//
// One claimed case, pdecimal006.n2 (a lone "NaN" against an enumeration whose
// members include "NaN"), is a KNOWN suite quirk: cvc-enumeration-valid matches
// on "equal OR identical" (§4.3.5.4) and NaN is identical to itself (§3.1, the
// single notANumber value), so the strict pipeline decides it VALID — yet the
// Saxon suite declares it invalid. Per the issue #135 GROUNDING (don't bend the
// spec to a fixture bug), the executor keeps the spec-correct verdict, so the
// harness honestly records this one case as a Fail (a New gap reflecting the suite
// bug, never a false Pass) rather than mis-implementing enumeration identity.
func execPDecimalCase(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	children, values, ok := readPDecimalCase(c.doc)
	if !ok {
		return Fail()
	}
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: "precisionDecimal"}
	builtinType, seeded := sym[qn]
	if !seeded {
		return Fail()
	}
	if !strictGoverns(backend, builtinType) {
		return Fail()
	}
	ownFacets, ok := buildOwnFacets("precisionDecimal", children)
	if !ok {
		return Fail()
	}
	leaf, err := xsd.NewSimpleType(xsderr.Loc{},
		xsd.QName{Space: synthNS, Local: "precisionDecimal-facets"},
		xsd.Atomic{Primitive: primitiveOfType(builtinType)}, builtinType, ownFacets, nil)
	if err != nil {
		return Fail()
	}
	// precisionDecimal maps context-free (§3.2), so a nil value.Context suffices —
	// unlike the QName cohort, no prefix resolution is involved.
	observedValid := true
	for _, v := range values {
		if _, verr := value.ValidateLexical(backend, leaf, v, nil); verr != nil {
			observedValid = false
			break
		}
	}
	if observedValid == c.expectValid {
		return Pass()
	}
	return Fail()
}

// strictGoverns reports whether st's governing mapping — its own or that of a
// base ancestor (widest-space rule, st-restrict-facets §3.16.6.4) — is supplied
// by the strict backend, so ValidateLexical parses through a spec-exact mapping.
// The integer family (xs:integer and its narrowings) has no strict mapping of its
// own; its nearest mapped ancestor is xs:decimal, which strict supplies (#81).
func strictGoverns(strictBackend value.Backend, st *xsd.SimpleType) bool {
	for s := st; s != nil; s = s.Base() {
		if _, ok := strictBackend.Mapping(s.Name()); ok {
			return true
		}
	}
	return false
}

// primitiveOfType returns st's primitive ancestor (§2.4.2) by walking Base(), so
// a synthesized leaf's {primitive type definition} points at the real primitive
// (xs:decimal for the integer family) rather than st's immediate builtin base. A
// directly-mapped primitive returns itself; the anySimpleType/anyAtomicType
// anchors (never in this cohort) yield nil.
func primitiveOfType(st *xsd.SimpleType) *xsd.SimpleType {
	for s := st; s != nil; s = s.Base() {
		if s.IsPrimitive() {
			return s
		}
	}
	return nil
}

// parseOK reports whether raw is in prim's lexical space, after applying prim's
// whiteSpace normalization (Datatypes §4.3.6) — collapse for boolean/decimal/
// QName/NOTATION (their fixed whiteSpace facet), preserve for string. ctx is the
// namespace context threaded to Parse: nil for the context-free primitives
// (whose Parse ignores it), a real value.Context for QName/NOTATION so a
// prefixed literal resolves against the bindings in scope (§3.3.18). This is the
// lexical cohort's path only; the facet cohort normalizes inside
// value.ValidateLexical.
func parseOK(m value.Mapping, prim, raw string, ctx value.Context) bool {
	_, err := m.Parse(normalizeWhiteSpace(prim, raw), ctx)
	return err == nil
}

// normalizeWhiteSpace applies prim's whiteSpace facet (read from the generated
// builtin table) to raw. Used only by the lexical cohort (parseOK); the facet
// cohort's normalization lives in value.ValidateLexical's whiteSpace stage, so
// there is exactly one normalization per path and no double-normalizing.
func normalizeWhiteSpace(prim, raw string) string {
	switch whiteSpaceOf(prim) {
	case "collapse":
		return strings.Join(strings.Fields(raw), " ")
	case "replace":
		return strings.Map(func(r rune) rune {
			if r == '\t' || r == '\n' || r == '\r' {
				return ' '
			}
			return r
		}, raw)
	default: // preserve
		return raw
	}
}

// whiteSpaceOf returns the spec whiteSpace value for a primitive, from the
// generated builtin table (never hand-typed); "" if the primitive is unknown.
func whiteSpaceOf(prim string) string {
	for _, t := range builtin.Types {
		if t.Name != prim {
			continue
		}
		for _, f := range t.Facets {
			if f.Name == "whiteSpace" {
				return f.Default
			}
		}
	}
	return ""
}

// readLexicalCase reads one lexical-cohort instance: it decodes the instance's
// leaf values (comp_foo and simpleTest) and the schema-under-test's tested
// primitive (from the instance's noNamespaceSchemaLocation). ok is false when
// either document cannot be read for this shape.
func readLexicalCase(instancePath string) (prim string, values []string, ok bool) {
	inst, err := decodeLexicalInstance(instancePath)
	if err != nil {
		return "", nil, false
	}
	if inst.SchemaLoc == "" {
		return "", nil, false
	}
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(inst.SchemaLoc))
	prim, err = decodeTestedPrimitive(schemaPath)
	if err != nil || prim == "" {
		return "", nil, false
	}
	return prim, []string{inst.ComplexTest.CompFoo, inst.SimpleTest}, true
}

// lexicalInstance mirrors the lexical cohort's instance shape: a root carrying
// the same value in complexTest/comp_foo (the primitive directly) and simpleTest
// (a facet-free restriction of it).
type lexicalInstance struct {
	SchemaLoc   string `xml:"http://www.w3.org/2001/XMLSchema-instance noNamespaceSchemaLocation,attr"`
	ComplexTest struct {
		CompFoo string `xml:"comp_foo"`
	} `xml:"complexTest"`
	SimpleTest string `xml:"simpleTest"`
}

func decodeLexicalInstance(path string) (lexicalInstance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return lexicalInstance{}, err
	}
	var inst lexicalInstance
	if err := xml.Unmarshal(data, &inst); err != nil {
		return lexicalInstance{}, err
	}
	return inst, nil
}

// lexicalSchema mirrors the lexical cohort's schema shape: its simplefooType
// restricts the tested builtin primitive with no facets.
type lexicalSchema struct {
	SimpleTypes []struct {
		Restriction struct {
			Base string `xml:"base,attr"`
		} `xml:"restriction"`
	} `xml:"simpleType"`
}

// decodeTestedPrimitive returns the local name of the primitive the schema
// tests (the restriction base of its first simpleType, prefix stripped).
func decodeTestedPrimitive(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var s lexicalSchema
	if err := xml.Unmarshal(data, &s); err != nil {
		return "", err
	}
	for _, st := range s.SimpleTypes {
		if base := st.Restriction.Base; base != "" {
			return localName(base), nil
		}
	}
	return "", nil
}

// typedLiteral pairs a tested value with the local name of the builtin primitive
// its attribute is declared as (from the sibling datatypes.xsd), so execItemCase
// can decide each <item>-attribute value against the right primitive.
type typedLiteral struct {
	prim  string
	value string
}

// readItemCase reads one lexical-cohort instance in the alternate
// <data><item ATTR="value"/></data> shape (issue #146): a handful of
// msData/datatypes/{dateTime013,duration028,duration029,duration030,gMonthDay006}
// .xml cases carry their tested values in SOMITEM_DATATYPE_* attributes of <item>
// children (some in two items, testing two attributes/types in one document)
// rather than the comp_foo/simpleTest shape. Their schema is declared out-of-band
// in the suite's testGroup metadata — the instance carries no
// noNamespaceSchemaLocation — and is always the sibling datatypes.xsd, which types
// each SOMITEM_DATATYPE_* attribute directly as an UNRESTRICTED builtin primitive
// (e.g. SOMITEM_DATATYPE_DURATION as xsd:duration). readItemCase resolves each
// present attribute name through that schema, returning one typedLiteral per
// recognized attribute in document order (the map is only a lookup; output order
// comes from the item/attribute document order, D3). ok is false when the shape
// does not decode, the sibling schema is unreadable, or no attribute matches a
// declared name — an honest decline, never a guess.
func readItemCase(instancePath string) (lits []typedLiteral, ok bool) {
	inst, err := decodeItemInstance(instancePath)
	if err != nil || len(inst.Items) == 0 {
		return nil, false
	}
	schemaPath := filepath.Join(filepath.Dir(instancePath), "datatypes.xsd")
	attrTypes, err := decodeItemAttrTypes(schemaPath)
	if err != nil || len(attrTypes) == 0 {
		return nil, false
	}
	for _, item := range inst.Items {
		for _, a := range item.Attrs {
			prim, known := attrTypes[a.Name.Local]
			if !known {
				continue
			}
			lits = append(lits, typedLiteral{prim: prim, value: a.Value})
		}
	}
	if len(lits) == 0 {
		return nil, false
	}
	return lits, true
}

// itemInstance mirrors the alternate lexical shape: a <data> root whose <item>
// children each carry the tested value(s) in arbitrary attributes (,any,attr,
// mirroring fooElem), read positionally so a document testing several attributes
// or several items is decoded whole.
type itemInstance struct {
	Items []itemElem `xml:"item"`
}

// itemElem is one <item>: its attributes, each a candidate tested value keyed by
// the attribute's local name.
type itemElem struct {
	Attrs []xml.Attr `xml:",any,attr"`
}

func decodeItemInstance(path string) (itemInstance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return itemInstance{}, err
	}
	var inst itemInstance
	if err := xml.Unmarshal(data, &inst); err != nil {
		return itemInstance{}, err
	}
	return inst, nil
}

// itemSchema mirrors datatypes.xsd's shape: top-level <xsd:attribute name type>
// declarations, each binding a SOMITEM_DATATYPE_* name to a builtin type.
type itemSchema struct {
	Attributes []struct {
		Name string `xml:"name,attr"`
		Type string `xml:"type,attr"`
	} `xml:"attribute"`
}

// decodeItemAttrTypes parses the sibling datatypes.xsd into a name -> primitive
// local-name lookup, reading the fixture itself rather than hand-typing the
// name->type table (STYLE 10; the same fixture-parsing discipline as
// decodeTestedPrimitive). Attributes with no type (e.g. SOMITEM_DATATYPE_ANYTYPE)
// are omitted, so an untyped attribute is never treated as a tested value.
func decodeItemAttrTypes(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s itemSchema
	if err := xml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(s.Attributes))
	for _, a := range s.Attributes {
		if a.Name == "" || a.Type == "" {
			continue
		}
		out[a.Name] = localName(a.Type)
	}
	return out, nil
}

// The single reserved, implicitly-bound XML namespace prefix (Namespaces in XML,
// §3): "xml" is bound by definition without any declaration. "xmlns" is NOT a
// resolvable QName prefix — it is the name of namespace-declaration attributes,
// not a binding for the prefix "xmlns" itself. The WG confirmed this (2010-02-05
// telcon, bugzilla 4053), reflected in the W3C suite's QName009_2092 expecting
// invalid for the literal "xmlns:xsi": its "xmlns" prefix has no in-scope binding.
const xmlPrefixNS = "http://www.w3.org/XML/1998/namespace"

// nsContext is the QName/NOTATION lexical cohort's value.Context: it resolves a
// prefix to the namespace name bound in scope at the point of a tested literal
// (§3.3.18: "the bindings to be used in the lexical mapping are those in the
// [in-scope namespaces] property of the relevant element"). bindings is an
// innermost-wins snapshot of the xmlns declarations on the literal's element and
// its ancestors, captured immutably during the streaming decode so each leaf
// keeps the bindings live where it occurred. It is an internal lookup, never
// ranged into output (STYLE D2).
type nsContext struct {
	bindings map[string]string
}

// LookupNamespace resolves prefix per §3.3.18's rules. The reserved prefix "xml"
// is always bound (Namespaces in XML §3); "xmlns" is deliberately NOT bound — it
// names namespace-declaration attributes, not a resolvable prefix (WG ruling,
// bugzilla 4053; the suite's QName009_2092 expects "xmlns:xsi" invalid on exactly
// this ground). A declared prefix resolves to its snapshot binding. The empty
// prefix (an unprefixed name) binds
// to the default namespace if declared, else to no namespace (ok=true, "") —
// element-name semantics, so an unprefixed QName is never rejected as unbound. A
// non-empty prefix with no declaration is genuinely unbound (ok=false), which
// strict's Parse turns into a cvc-datatype-valid rejection (§4.1.4).
func (c nsContext) LookupNamespace(prefix string) (namespace string, ok bool) {
	if prefix == "xml" {
		return xmlPrefixNS, true
	}
	if uri, bound := c.bindings[prefix]; bound {
		return uri, true
	}
	if prefix == "" {
		return "", true
	}
	return "", false
}

// qnameLiteral pairs a tested QName/NOTATION leaf value with the namespace
// context in scope at its element, so execContextualCase resolves each literal
// against the bindings live where it occurs.
type qnameLiteral struct {
	value string
	ctx   value.Context
}

// readQNameContexts streams a QName/NOTATION lexical-cohort instance and returns
// each tested leaf value (the comp_foo and simpleTest content, the same shape
// readLexicalCase decodes) paired with the in-scope namespace context at its
// element. It tracks the xmlns declarations down the ancestor chain itself (a raw
// literal's prefix is character content, not an XML name the decoder resolves),
// snapshotting the accumulated bindings when a leaf opens. ok is false when the
// instance cannot be read or carries no tested leaf — an honest decline.
func readQNameContexts(instancePath string) (lits []qnameLiteral, ok bool) {
	f, err := os.Open(instancePath)
	if err != nil {
		return nil, false
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the parsed result
	dec := xml.NewDecoder(bufio.NewReader(f))
	var frames []map[string]string // one innermost-wins snapshot per open element
	capturing := false
	var capText strings.Builder
	var capCtx nsContext
	for {
		tok, terr := dec.Token()
		if terr != nil {
			break // io.EOF or malformed: stop; a partial decode yields no leaves and declines
		}
		switch t := tok.(type) {
		case xml.StartElement:
			frames = append(frames, childBindings(frames, t.Attr))
			if !capturing && isQNameLeaf(t.Name.Local) {
				capturing = true
				capText.Reset()
				capCtx = nsContext{bindings: frames[len(frames)-1]}
			}
		case xml.EndElement:
			if capturing && isQNameLeaf(t.Name.Local) {
				lits = append(lits, qnameLiteral{value: capText.String(), ctx: capCtx})
				capturing = false
			}
			if len(frames) > 0 {
				frames = frames[:len(frames)-1]
			}
		case xml.CharData:
			if capturing {
				capText.Write(t)
			}
		}
	}
	if len(lits) == 0 {
		return nil, false
	}
	return lits, true
}

// isQNameLeaf reports whether an element local name carries a tested QName/
// NOTATION literal in the lexical cohort's instance shape (comp_foo under
// complexTest, and simpleTest), mirroring readLexicalCase's decoded value set.
func isQNameLeaf(local string) bool {
	return local == "comp_foo" || local == "simpleTest"
}

// childBindings returns the namespace snapshot for an element: its parent's
// snapshot (the innermost frame, or empty at the root) overlaid with this
// element's own xmlns declarations. The clone keeps each snapshot immutable so a
// captured leaf's context is unaffected by later siblings (maps.Clone/overlay is
// an internal state copy, order-independent, never output — STYLE D2).
func childBindings(frames []map[string]string, attrs []xml.Attr) map[string]string {
	var snap map[string]string
	if n := len(frames); n > 0 {
		snap = maps.Clone(frames[n-1])
	}
	if snap == nil {
		snap = map[string]string{}
	}
	for _, a := range attrs {
		prefix, isNS := nsDeclaration(a)
		if !isNS {
			continue
		}
		snap[prefix] = a.Value
	}
	return snap
}

// nsDeclaration reports whether attribute a is an XML namespace declaration and,
// if so, the prefix it binds: xmlns:p="…" binds p (Go models it as Name.Space
// "xmlns", Name.Local the prefix); xmlns="…" binds the empty (default) prefix
// (Name.Space "", Name.Local "xmlns"). Any other attribute is not a declaration.
func nsDeclaration(a xml.Attr) (prefix string, ok bool) {
	if a.Name.Space == "xmlns" {
		return a.Name.Local, true
	}
	if a.Name.Space == "" && a.Name.Local == "xmlns" {
		return "", true
	}
	return "", false
}

// facetChild is one constraining-facet element read from a Facets-cohort schema:
// its element local name (e.g. "length") and its value attribute.
type facetChild struct {
	name  string
	value string
}

// facetKinds is the set of facet kinds the facet cohort recognizes: the value-
// and pattern-facet kinds value.ValidateLexical decides for
// string/decimal/float/double (the bound facets also serve the
// partially-ordered float/double) plus precisionDecimal's two extension facets
// maxScale/minScale (issue #135, applicable ONLY to precisionDecimal per
// xsd-precisionDecimal.md §3.3; harmless for the msData cohort, whose bases never
// carry them and whose Applies metadata rejects them regardless).
// whiteSpace (normalization, no cvc-* rule), assertions and explicitTimezone are
// deliberately excluded, so a schema carrying one is declined rather than
// silently ignored.
var facetKinds = []xsd.FacetKind{
	xsd.FacetLength, xsd.FacetMinLength, xsd.FacetMaxLength,
	xsd.FacetPattern, xsd.FacetEnumeration,
	xsd.FacetMaxInclusive, xsd.FacetMaxExclusive,
	xsd.FacetMinExclusive, xsd.FacetMinInclusive,
	xsd.FacetTotalDigits, xsd.FacetFractionDigits,
	xsd.FacetMaxScale, xsd.FacetMinScale,
}

// facetKindOf maps a facet element's local name to its xsd.FacetKind by matching
// the kind's spec token (never a hand-typed name table; the token is
// FacetKind.String's own output). ok is false for an unrecognized name.
func facetKindOf(name string) (xsd.FacetKind, bool) {
	for _, k := range facetKinds {
		if k.String() == name {
			return k, true
		}
	}
	return 0, false
}

// typeSpecOf returns the builtin TypeSpec for the primitive named name, carrying
// its applicable-facet metadata (cos-applicable-facets). ok is false if unknown.
func typeSpecOf(name string) (builtin.TypeSpec, bool) {
	for _, t := range builtin.Types {
		if t.Name == name {
			return t, true
		}
	}
	return builtin.TypeSpec{}, false
}

// buildOwnFacets translates the schema's facet children into the leaf's
// ownFacets, grouping same-kind children (pattern/enumeration carry a set of
// {value}s) into one facet in first-seen order (D2: the map is a lookup, output
// order comes from the order slice). It returns ok=false — declining the case —
// when a child names an unrecognized facet or a facet inapplicable to base
// (cos-applicable-facets §4.1.5), so the synthesized leaf never carries a facet
// that would trip ValidateLexical's panic precondition.
func buildOwnFacets(base string, children []facetChild) ([]xsd.Facet, bool) {
	spec, ok := typeSpecOf(base)
	if !ok {
		return nil, false
	}
	// EXPLICIT decline (issue #125): enumeration over QName compares §3.2.18
	// {namespace name, local name} tuples, so a prefixed enum member must resolve
	// against the DECLARING SCHEMA's in-scope bindings — a context xsd.Facet/
	// value.newEnumFacet cannot yet carry (value/facets.go parses each enum lexical
	// with a hardcoded nil context). Building that threading is a warden-gated
	// follow-up (see the issue #125 GROUNDING comment); until then a QName
	// enumeration case is an honest recorded gap, declined here rather than fed
	// through and mis-decided against the wrong (instance) context.
	if base == "QName" {
		for _, ch := range children {
			if ch.name == xsd.FacetEnumeration.String() {
				return nil, false
			}
		}
	}
	var order []xsd.FacetKind
	values := map[xsd.FacetKind][]string{}
	for _, ch := range children {
		kind, ok := facetKindOf(ch.name)
		if !ok {
			return nil, false
		}
		if !spec.Applies(builtin.FacetName(kind.String())) {
			return nil, false
		}
		if _, seen := values[kind]; !seen {
			order = append(order, kind)
		}
		values[kind] = append(values[kind], ch.value)
	}
	facets := make([]xsd.Facet, 0, len(order))
	for _, kind := range order {
		facets = append(facets, xsd.NewFacet(kind, values[kind], false))
	}
	return facets, true
}

// readFacetsCase reads one facet-cohort instance: the tested value (the <foo>
// leaf text, un-normalized — ValidateLexical's whiteSpace stage normalizes it)
// and, from the schema at the instance's noNamespaceSchemaLocation, the
// restriction's base primitive and facet children. ok is false when either
// document cannot be read for this shape.
func readFacetsCase(instancePath string) (raw, base string, children []facetChild, ctx value.Context, ok bool) {
	inst, err := decodeFacetsInstance(instancePath)
	if err != nil || inst.SchemaLoc == "" || len(inst.Foos) != 1 {
		return "", "", nil, nil, false
	}
	foo := inst.Foos[0]
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(inst.SchemaLoc))
	base, attrName, children, ok := decodeRestriction(schemaPath)
	if !ok || base == "" || len(children) == 0 {
		return "", "", nil, nil, false
	}
	// The instance's root-level namespace bindings, threaded to ValidateLexical so a
	// context-dependent primitive's Parse (QName, §3.3.18) can resolve prefixes.
	// childBindings(nil, ...) reuses the lexical cohort's declaration reader; the
	// result is a lookup-only map, never ranged into output (STYLE D2).
	ctx = nsContext{bindings: childBindings(nil, inst.Attrs)}
	// The NMTOKEN cohort (unlike language/Name/NCName) carries the tested value
	// in a named attribute of <foo> rather than its content: the restriction is
	// declared on an <xsd:attribute>. When decodeRestriction reports that
	// attribute's name, read the matching instance attribute; otherwise the value
	// is <foo>'s element content.
	if attrName != "" {
		v, found := foo.attr(attrName)
		if !found {
			return "", "", nil, nil, false
		}
		return v, base, children, ctx, true
	}
	return foo.Text, base, children, ctx, true
}

// facetsInstance mirrors the Facets cohort's instance shape: a <test> root whose
// single <foo> child holds the tested value in its content or a named attribute.
// Foos collects every <foo> child so readFacetsCase can require EXACTLY ONE: an
// out-of-cohort shape carrying zero <foo> leaves (e.g. the anyURI
// Facets/anyURI/anyURI_b001.xml case whose values live in repeated <bar>
// children) or several (e.g. anyURI_b006.xml, a list-style instance repeating
// many <foo> values against one enumeration) is honestly declined rather than
// mis-read as a single empty or last-wins tested value.
// Attrs captures the <test> root's raw attributes (mirroring fooElem.Attrs) so the
// QName cohort (issue #125) can build the instance's root-level namespace context.
// Every Facets/QName fixture in the current checkout declares its xmlns bindings
// only on this root (verified), so a root-only snapshot is a complete context for
// the tested <foo> literal — no ancestor-chain streaming (readQNameContexts) is
// needed here. The field is inert for every other base type (their Parse ignores
// the threaded context).
type facetsInstance struct {
	SchemaLoc string     `xml:"http://www.w3.org/2001/XMLSchema-instance noNamespaceSchemaLocation,attr"`
	Attrs     []xml.Attr `xml:",any,attr"`
	Foos      []fooElem  `xml:"foo"`
}

// fooElem is the <foo> element: its text content plus any attributes, so a case
// carrying the tested value in an attribute can be read as well as one carrying
// it in element content.
type fooElem struct {
	Text  string     `xml:",chardata"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// attr returns the value of the unqualified attribute named local, and whether
// it was present.
func (f fooElem) attr(local string) (string, bool) {
	for _, a := range f.Attrs {
		if a.Name.Local == local {
			return a.Value, true
		}
	}
	return "", false
}

func decodeFacetsInstance(path string) (facetsInstance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return facetsInstance{}, err
	}
	var inst facetsInstance
	if err := xml.Unmarshal(data, &inst); err != nil {
		return facetsInstance{}, err
	}
	return inst, nil
}

// decodeRestriction streams the schema and returns the base primitive (prefix
// stripped), the name of the enclosing xsd:attribute if the restriction is
// declared on one (empty when it constrains element content), and the
// constraining-facet children of its first xsd:restriction. Facet children are
// the restriction's direct element children in the XML Schema namespace, in
// document order (P4: token stream, no whole-document buffer). ok is false when
// no restriction is found.
func decodeRestriction(path string) (base, attrName string, children []facetChild, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", nil, false
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the parsed result
	dec := xml.NewDecoder(bufio.NewReader(f))
	inRestriction := false
	lastAttr := ""
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if end, isEnd := tok.(xml.EndElement); isEnd {
			if inRestriction && end.Name.Local == "restriction" && end.Name.Space == xsd.XMLSchemaNS {
				return base, attrName, children, true
			}
			continue
		}
		se, isStart := tok.(xml.StartElement)
		if !isStart {
			continue
		}
		if !inRestriction {
			if se.Name.Local == "attribute" && se.Name.Space == xsd.XMLSchemaNS {
				lastAttr = attrValue(se, "name")
			}
			if se.Name.Local == "restriction" && se.Name.Space == xsd.XMLSchemaNS {
				inRestriction = true
				base = localName(attrValue(se, "base"))
				attrName = lastAttr
			}
			continue
		}
		if se.Name.Space == xsd.XMLSchemaNS {
			children = append(children, facetChild{name: se.Name.Local, value: attrValue(se, "value")})
		}
	}
	if inRestriction {
		return base, attrName, children, true
	}
	return "", "", nil, false
}

// readPDecimalCase reads one Saxon PDecimal cohort instance (issue #135): the
// tested <e value="…"/> literals and the sole tested precisionDecimal type's
// facet children. The schema is out-of-band (the instance carries no
// noNamespaceSchemaLocation — the suite's testGroup pairs pdecimalNNN.{vK,nK}.xml
// with the sibling pdecimalNNN.xsd), so pdecimalSchemaPath derives it from the
// instance filename's case prefix. ok is false when the instance decodes to no
// <e> value, the schema cannot be read, or the attribute value's type is not a
// directly-mapped or single-step precisionDecimal restriction (a multi-step
// chain, list or union — pdecimal016/019/020 — which this single-leaf model
// cannot decide) — an honest decline, never a guess.
func readPDecimalCase(instancePath string) (children []facetChild, values []string, ok bool) {
	values, ok = decodePDecimalValues(instancePath)
	if !ok {
		return nil, nil, false
	}
	base, children, ok := decodePDecimalSchema(pdecimalSchemaPath(instancePath))
	if !ok || base != "precisionDecimal" {
		return nil, nil, false
	}
	return children, values, true
}

// pdecimalSchemaPath derives the sibling schema path for a PDecimal instance from
// its filename's case prefix: pdecimal001.v1.xml → pdecimal001.xsd (the schema
// the suite's testGroup pairs with every instance of that case). The prefix is
// the basename up to its first '.', so the .vK/.nK/.xml suffixes are stripped.
func pdecimalSchemaPath(instancePath string) string {
	base := filepath.Base(instancePath)
	prefix := base
	if i := strings.IndexByte(base, '.'); i >= 0 {
		prefix = base[:i]
	}
	return filepath.Join(filepath.Dir(instancePath), prefix+".xsd")
}

// pdecimalInstance mirrors the PDecimal cohort's instance shape: a root (<doc>)
// whose repeated <e value="…"/> children each carry one tested literal in an
// unqualified value attribute.
type pdecimalInstance struct {
	Es []struct {
		Value string `xml:"value,attr"`
	} `xml:"e"`
}

// decodePDecimalValues reads every <e value="…"/> literal in document order. ok
// is false when the document cannot be read or carries no <e> child (an empty or
// out-of-shape document is declined rather than treated as a vacuous pass).
func decodePDecimalValues(path string) (values []string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var inst pdecimalInstance
	if err := xml.Unmarshal(data, &inst); err != nil {
		return nil, false
	}
	if len(inst.Es) == 0 {
		return nil, false
	}
	for _, e := range inst.Es {
		values = append(values, e.Value)
	}
	return values, true
}

// pdecimalSchema mirrors the PDecimal cohort's schema shape: an <element name="e">
// whose complexType declares the tested attribute "value", and the named
// simpleTypes it may reference. Only the value attribute's own type matters.
type pdecimalSchema struct {
	Elements []struct {
		Name        string `xml:"name,attr"`
		ComplexType struct {
			Attributes []struct {
				Name string `xml:"name,attr"`
				Type string `xml:"type,attr"`
			} `xml:"attribute"`
		} `xml:"complexType"`
	} `xml:"element"`
	SimpleTypes []struct {
		Name        string `xml:"name,attr"`
		Restriction struct {
			Base   string `xml:"base,attr"`
			Facets []struct {
				XMLName xml.Name
				Value   string `xml:"value,attr"`
			} `xml:",any"`
		} `xml:"restriction"`
	} `xml:"simpleType"`
}

// decodePDecimalSchema resolves the type of the tested attribute "value" on
// element "e" and returns its precisionDecimal base plus facet children. Two
// shapes are decided: an attribute typed xs:precisionDecimal directly (base
// "precisionDecimal", no facets), or one typed as a named simpleType that is a
// SINGLE-STEP restriction of precisionDecimal (base "precisionDecimal", its facet
// children). Any other shape — a restriction of another named type (a multi-step
// chain), or a list/union variety (whose simpleType carries no precisionDecimal
// restriction, so its Restriction.Base is empty) — yields ok=false, declining the
// case. ok is false too when the schema cannot be read or has no such attribute.
func decodePDecimalSchema(path string) (base string, children []facetChild, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, false
	}
	var s pdecimalSchema
	if err := xml.Unmarshal(data, &s); err != nil {
		return "", nil, false
	}
	attrType, found := pdecimalValueType(s)
	if !found {
		return "", nil, false
	}
	if localName(attrType) == "precisionDecimal" {
		return "precisionDecimal", nil, true
	}
	for _, st := range s.SimpleTypes {
		if st.Name != attrType {
			continue
		}
		if localName(st.Restriction.Base) != "precisionDecimal" {
			return "", nil, false
		}
		for _, f := range st.Restriction.Facets {
			children = append(children, facetChild{name: f.XMLName.Local, value: f.Value})
		}
		return "precisionDecimal", children, true
	}
	return "", nil, false
}

// pdecimalValueType returns the type QName (prefix intact) of the tested
// attribute "value" declared on element "e". found is false when the schema
// declares no such element/attribute.
func pdecimalValueType(s pdecimalSchema) (attrType string, found bool) {
	for _, el := range s.Elements {
		if el.Name != "e" {
			continue
		}
		for _, a := range el.ComplexType.Attributes {
			if a.Name == "value" {
				return a.Type, true
			}
		}
	}
	return "", false
}

// attrValue returns the value of se's unqualified attribute local, or "".
func attrValue(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// localName strips a QName's prefix, returning its local part.
func localName(qn string) string {
	if i := strings.LastIndexByte(qn, ':'); i >= 0 {
		return qn[i+1:]
	}
	return qn
}
