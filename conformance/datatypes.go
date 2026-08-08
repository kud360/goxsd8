package conformance

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
// # The lexical cohort (issue #15, widened by issues #80 and #331)
//
// The lane claims the Microsoft datatype LEXICAL cases under
// msData/datatypes/{boolean,decimal,string,float,double,anyURI,hexBinary,
// base64Binary,duration,dateTime,dateTimeStamp,time,date,gYearMonth,gYear,
// gMonthDay,gDay,gMonth}NNN.xml and, since issue #331, the DERIVED integer
// family {byte,long,short,unsignedByte,unsignedInt,unsignedLong,
// unsignedShort}NNN.xml ("The derived sub-cohort" below). Each schema of the
// PRIMITIVE group declares an element of an UNRESTRICTED builtin primitive
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
// ## The derived (integer-family) sub-cohort (issue #331)
//
// The forty-eight msData/datatypes/{byte,long,short}00[1-8].xml and
// {unsignedByte,unsignedInt,unsignedLong,unsignedShort}00[1-6].xml fixtures have
// the same comp_foo/simpleTest shape but a tested type that is NOT a primitive:
// xs:byte and its six siblings are facet restrictions of xs:decimal
// (§3.4.13–§3.4.25), and the strict backend maps the 20 primitives only. Parse
// alone is therefore an INCOMPLETE check for them — it would satisfy only
// cvc-datatype-valid clause 2.1 (§4.1.4) and false-accept "128" or "1.5" as an
// xs:byte — which is why execLexicalCase used to decline them at a
// backend.Mapping miss. It no longer does: a seeded type with no direct mapping
// now routes to decideLexicalByFacets, exactly as a type fixing explicitTimezone
// does, so value.ValidateLexical applies the type's OWN effective facets (the
// fixed pattern [\-+]?[0-9]+ at clause 1, then fractionDigits=0 and the per-type
// min/maxInclusive bounds at clause 3) over the lexical space of the
// {primitive type definition} its base chain reaches (value.governingMapping, the
// widest-space rule of st-restrict-facets §3.16.6.4). Both polarities are then
// decided for the right reason: the out-of-range fixtures (byte008's "-129",
// unsignedShort006's "65536", …) are rejected by cvc-min/maxInclusive-valid
// (§4.3.10.3/§4.3.7.3), a fraction-point or empty literal by cvc-pattern-valid
// (§4.3.4.4) — the pattern gate running BEFORE the value facets, so a "5.0" is a
// pattern rejection, not a fractionDigits one. The same generated builtin facets
// the Facets integer cohort (issue #81) and the list cohort (issue #224) already
// ride; no new backend mapping and no change to package value.
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
// dateTimeStamp (§3.4.28) is listed and decided completely, though it has ZERO
// cases in the current checkout. Being a restriction of dateTime that fixes
// explicitTimezone=required, its validity ALSO depends on the timezone being
// present — a VALUE-based facet (cvc-explicitTimezone-valid §4.3.14.3) checked at
// cvc-datatype-valid §4.1.4 clause 3, AFTER the lexical mapping, so parseDateTime
// alone would false-ACCEPT a tz-absent literal. execLexicalCase therefore routes
// any type whose EffectiveFacets fix a non-optional explicitTimezone (fixesTimezone
// — read from the fact, never a dateTimeStamp type-name special case, so a
// user-defined date/time restriction that fixes explicitTimezone per §4.3.14.4 is
// covered too) through the SAME facet pipeline the facet cohort uses
// (value.ValidateLexical, decideLexicalByFacets), which enforces the timezone. A
// tz-absent dateTimeStamp literal is thus correctly REJECTED (issue #140), so a
// future msData/datatypes/dateTimeStampNNN.xml case cannot regress the ratchet.
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
// violation always returns an *xsderr.Error through the normal path and
// ValidateLexical's facet PRECONDITION is never violated. A case pairing an
// inapplicable facet with a primitive (a schema-construction error, not an instance
// validity case) is declined rather than fed through. That ownership is EXECUTED, not
// merely documented: every ValidateLexical call site in this file routes its error
// through mustNotBeFacetPrecondition, which fails the run rather than let a
// precondition fault be scored as an instance rejection.
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
// shape (several named types per schema, each tested by a dedicated element) is a
// distinct, larger executor, now claimed by execD34Case (issue #162 — see its doc
// comment for the multi-type dispatch, the decidable/declined split and the shared
// spec rules); its list (v16) and union (v17) shapes joined the decidable set with
// issue #223 and its multi-step restriction chains (v18, v19–v22) with issue #574,
// while the one D3_3_4 shape still structurally out of reach — v15's
// complexType-typed children — is declined honestly and recorded as a gap in the
// datatypes lane, which is this repo's most complete lane rather than an inert one,
// so the decline costs a real point.
//
// # The list-variety cohort (issues #75, #224)
//
// The lane also decides the LIST-variety Facets cases under
// msData/datatypes/{boolean018,float038,float039,anyURI011,hexBinary002,
// base64Binary002,duration027}.xml and, since issue #224, their seven
// integer-family siblings {byte009,long009,short009,unsignedByte007,
// unsignedInt007,unsignedLong007,unsignedShort007}.xml (claimed by
// datatypesCase's integer-family alternation since issue #331, by the dedicated
// integerListCase selector before it). Each such schema declares a user-defined
// "myList" (<xsd:list itemType="xsd:BUILTIN"/>) reached through comp_foo (either
// type="myList" directly or an inline anonymous restriction of it) and one-or-more
// simpleTest elements (a named restriction of "myList"); the restriction may carry
// <xsd:enumeration> children (boolean018 enumerates 0/1 on comp_foo and true/false
// on simpleTest; the other thirteen are facet-free). Because comp_foo and simpleTest can
// carry DIFFERENT own facets, execListCase reads each element's type graph
// independently (readListCohortCase), synthesizing the fixture's TWO derivation
// steps per tested element — an anonymous constructed list over the item type
// with xs:anySimpleType as base and the mandatory fixed whiteSpace=collapse
// §4.3.6.1 as its whole {facets} (cos-st-restricts clause 2.2.1.2), then the
// named leaf restricting it and carrying any enumeration —
// and deciding every tested value through the ordinary value.ValidateLexical
// pipeline. That pipeline resolves a list-variety type end to end (issue #75):
// value.governingMapping wraps the item TYPE in a generic list mapping
// (cvc-datatype-valid clause dv_list §4.1.4 cl.2.2), so each lexical is split into
// items, each item is Datatype-Valid against the item type, and the leaf's
// enumeration is checked over the LIST value space by "equal or identical"
// (cvc-enumeration-valid §4.3.5.4, §2.2.1/§2.2.2 pairwise over items; length, had
// any fixture one, would count list items §4.3.1.3). The case is valid iff every
// value across comp_foo and every simpleTest sibling validates (float039.xml's
// nine simpleTest siblings — spanning ±MIN/MAX float, NaN, ±INF, ±0 — are each
// checked). No new backend mapping and no new exported identifier: the list
// mapping is unexported (value.listMapping/listValue), consuming the existing
// strict primitive mappings via the item type.
//
// The item type is DERIVED, not a primitive, in the seven fixtures issue #224
// added, and that distinction is load-bearing: xs:byte's {primitive type
// definition} is xs:decimal, so the lexical mapping alone accepts every decimal
// literal while byte's OWN effective facets (pattern, fractionDigits=0,
// minInclusive=-128, maxInclusive=127) are what reject "128" or "1.5". dv_list
// requires the FULL cvc-datatype-valid rule per item, clause 3 (dv_vfacets)
// included, so value.listMapping decides each item by recursing through
// validateLexical against the item type rather than parsing it through the item
// type's governing mapping — the #224 fix that closed that false-accept before
// these fixtures were claimed.
//
// The msData UNION-variety Facets fixtures are still unclaimed by THIS
// reader — value.ValidateLexical decides a union end to end since issue #223
// (cvc-datatype-valid clause dv_union, §4.1.4 cl.2.3, value/union.go), and the
// D3_3_4 cohort's union case d3_3_4v17 rides it, but no msData union fixture has
// a reader here yet.
//
// # The anyURI a*/b* multi-leaf cohort (issue #190)
//
// The lane also decides the eight anyURI instance cases under
// msData/datatypes/Facets/anyURI/anyURI_{a001,a002,a004,b001,b002,b004,b005,b006}.xml.
// They share the Facets directory and filename form with the single-<foo> cohort
// but not its instance shape, which is why they were honest declines from the
// #124 landing until now (see readFacetsCase's exactly-one-<foo> guard, still in
// force for everything that reaches it). Two shape differences and one type
// difference:
//
//   - The a* instances declare their schema with the namespace-qualified
//     xsi:schemaLocation form (§2.6.3, a "<namespace> <location>" pair) rather than
//     xsi:noNamespaceSchemaLocation, since their schemas have a target namespace —
//     the same form the IBM D3_3_4 cohort uses, so qualifiedSchemaLocation resolves
//     both.
//   - The tested values are PLURAL and live wherever the schema puts them: a lone
//     <bar> element's text (a001/a002), a <root> wrapper's unqualified attribute
//     plus repeated <bar> children (a004, whose bar is a LOCAL declaration inside
//     the wrapper's inline complexType), repeated <bar> children reached by ref=
//     (b001/b004/b005), <bar> children of a simpleContent complexType that adds an
//     attribute so text AND attribute are both tested (b002), or a <choice>-repeated
//     mix of <foo> and <bar> (b006).
//   - In b006 those <foo> and <bar> leaves have DIFFERENT types: <foo> is typed
//     xsd:anyURI directly (facet-free) while <bar> carries the enumeration-restricted
//     named type, so each leaf must be validated against the type its own
//     declaration binds, not against one type for the whole document.
//
// readAnyURIShapeCase therefore reads element and attribute declarations from the
// schema, resolves each instance leaf to its declared simple type, and
// execAnyURIShapeCase ANDs value.ValidateLexical over every leaf — the same
// whole-document polarity execPDecimalCase and execD34Case use. That leaf-AND
// model decides LEXICAL validity only and validates no content model: occurrence
// constraints (minOccurs/maxOccurs) and element order go unchecked, so an
// instance that is invalid ONLY structurally would still score valid here. This
// is a cohort-wide convention rather than anything specific to anyURI —
// execFacetsCase and execD34Case take the same shortcut — and it is sound for
// these eight fixtures, which all satisfy their content models (maxOccurs
// 100/1000/unbounded against 3/6/7/26 children). No new spec rule
// and no new backend mapping is involved: all eight fixtures restrict xsd:anyURI
// with enumeration only, so the rules in play are cvc-enumeration-valid (§4.3.5.4,
// "equal or identical to one of the values specified in {value}") over
// cvc-datatype-valid (§4.1.4), both already wired, decided through the strict
// anyURI mapping whose value space is the identity on Char* (§3.3.17.2 — the reason
// the a004/b004/b006 SCHEMAS, invalid per 1.0, are valid per 1.1). Two spec facts
// carry specific fixtures: anyURI's fixed whiteSpace=collapse (§4.3.6) turns
// b005's "http://a/x  y" into "http://a/x y", which is still not the enumeration's
// "http://a/x%20y" because the mapping does no percent-decoding (§3.3.17.2 Note:
// "if two 'equivalent' URIs or IRIs are different character sequences, they map to
// different values"), so b005 is invalid; the same Note makes b006's "\a" distinct
// from every enumeration member, so b006 is invalid on that ONE leaf even though
// its twenty-five others match — exactly the reason its testGroup annotation gives.
// anyURI_a004_1339.i is the cohort's one recorded gap and NOT a shape limit: its
// seven leaves are all enumeration members, so the spec-correct verdict is valid
// against a suite expectation of invalid that the fixture's own annotation
// contradicts and that has carried status="queried" since 2010 — see
// execAnyURIShapeCase.
//
// # Still deferred
//
// Facets over QName (the Facets/QName dir) are now FULLY claimed (issue #125,
// completed by #152): the length/minLength/maxLength cases (vacuous per clause 1.3),
// the pattern case, AND the enumeration cases all decide through the ordinary
// pipeline. A QName enumeration member's prefix resolves against the DECLARING
// schema's in-scope bindings (§3.3.18), which the facet now carries per member
// (xsd.EnumerationMember): decodeRestriction snapshots each <enumeration>
// element's namespace context down the schema document's ancestor chain, and
// buildOwnFacets threads it into xsd.NewEnumerationMember, so value.newEnumFacet
// resolves each member against the right scope (an unresolvable member is a
// src-enumeration-value schema-construction defect, §4.3.5.3). Facets over
// NOTATION (the Facets/NOTATION dir) are now claimed too (issue #153): unlike
// every other cohort member their fixtures use a TWO-STEP restriction — a named
// simpleType restricts xsd:NOTATION with the jpeg/mpeg/g enumerations (the only
// way to make NOTATION usable, §3.3.19 enumeration-required-notation), then an
// anonymous attribute simpleType restricts THAT with one more tested facet
// (length/minLength/maxLength/pattern/enumeration), with the tested value carried
// in the <foo> element's attrTest attribute. A bespoke reader+decoder+executor
// (readNotationFacetsCase/decodeNotationRestriction/execNotationFacetsCase, kept
// separate from readFacetsCase/decodeRestriction/execFacetsCase to avoid any
// regression to the single-step cohort) synthesizes the two-level chain and
// decides it through the ordinary value.ValidateLexical pipeline; the sibling
// <xsd:notation> declarations are not load-bearing for any instance verdict
// (§3.14.1 is a schema-construction SCC satisfied by every fixture) and are not
// parsed (STYLE D4). xsd:boolean facets (no Facets dir exists for it), the plural
// list-typed dirs (IDREFS, NMTOKENS), the NIST corpus, and UNION variety remain
// out of scope until their backends land. LIST variety over a lexical-cohort item
// primitive is now decided (issue #75, "The list-variety cohort" above): the
// boolean018/float038/float039/anyURI011/hexBinary002/base64Binary002/duration027
// fixtures flipped from recorded gaps to decided passes.
// string_pattern002_1031.i (issue #146) is a list-variety case that stays deferred
// for two reasons: its Facets/string/string_pattern002.xml restricts via
// <xsd:list itemType="Hex"/> where "Hex" is a USER-DEFINED pattern-restricted
// simpleType, not a seeded strict-mapped primitive (execListCase declines a
// non-seeded item type), and its instance shape (a <Xml xmlns="TestNamespace">
// root with three <Hex> list-valued children) matches neither readFacetsCase's
// single-<foo> shape nor readListCohortCase's comp_foo/simpleTest shape, so it is
// honestly declined (Fail), never false-accepted.
// Within the integer family, the odd
// multi-element cases (e.g. Facets/int/test111092.xml, two named restriction
// steps under distinct elements) do not fit the single-<foo> instance shape and
// fall through to the instance lane as recorded gaps (issue #331 left them
// there: this is a READER-shape limit, not the routing gap that issue closed).
// The family's LIST-variety fixtures (byte009/long009/short009/unsignedByte007/
// unsignedInt007/unsignedLong007/unsignedShort007.xml) were claimed and decided
// by issue #224. Their forty-eight NON-list siblings (byte001–008, long001–008,
// short001–008, unsignedByte/Int/Long/Short001–006, each testing a value against
// xs:byte/xs:long/… through the shared byte.xsd/long.xsd) are claimed and decided
// too since issue #331: execLexicalCase no longer demands a DIRECT backend
// Mapping for the tested type but routes a seeded-but-unmapped one through
// value.ValidateLexical (decideLexicalByFacets), where governingMapping walks the
// base chain to xs:decimal's mapping and the type's own effective facets decide
// the value — the same pipeline xs:dateTimeStamp has taken since issue #140 and
// the list path has taken per item since #224. Their xs:int and xs:integer
// siblings (int001–008 and integer001–016 — twenty-four files, not the twenty
// #331's prose miscounted: integer013–016 exist and carry the identical
// comp_foo/simpleTest shape) joined the claim with issue #365, which widened
// datatypesCase's alternation and changed no engine code; they ride that same
// route unchanged. The last four integer-family sub-families under
// msData/datatypes — negativeInteger001–005, nonNegativeInteger001–005,
// nonPositiveInteger001–005 and positiveInteger001–005, twenty files in all —
// joined the claim with issue #449, again by widening datatypesCase's
// alternation and changing no engine code. Each carries the identical
// comp_foo/simpleTest shape against its own negativeInteger.xsd/
// nonNegativeInteger.xsd/nonPositiveInteger.xsd/positiveInteger.xsd. They are
// the route's first HALF-bounded arm: #331's forty-eight all carry BOTH
// min- and maxInclusive and #365's xs:integer carries NEITHER, while each of
// these four carries exactly one bound (§3.4.14.3 maxInclusive=0,
// §3.4.15.3 maxInclusive=-1, §3.4.20.3 minInclusive=0, §3.4.25.3
// minInclusive=1), so a single-sided value-facet check is the only thing
// between a well-formed integer literal and acceptance. Two of them are also
// the route's first TWO-HOP base chains: negativeInteger restricts
// nonPositiveInteger and positiveInteger restricts nonNegativeInteger, so
// st-restrict-facets §3.16.6.4's overlay walk crosses two restriction steps
// before reaching xs:integer and then xs:decimal's mapping — it does, and
// TestDatatypesLexicalHalfBoundedIntegerFamily pins it.
// What is still undecided is therefore enumerable rather than open-ended
// (STYLE P3a). Within the integer family, exactly one kind of case remains:
// the reader-shape limits named above — Facets/int/test111092.xml's
// two-named-step, two-element shape and any sibling of that kind, which
// readFacetsCase's exactly-one-<foo> reader declines. Outside it, the lane's
// standing exclusions are unchanged and named elsewhere in this comment: the
// NIST corpus, UNION variety, the plural list-typed dirs (IDREFS, NMTOKENS),
// string_pattern002's user-defined list item type, and
// time_minInclusive006_1163.i. Every one of those is an honest decline (Fail,
// recorded in the instance lane), never a false accept.
// time_minInclusive006_1163.i (issue #123) is a
// recorded gap for a different reason: its instance file carries no
// xsi:noNamespaceSchemaLocation (a defect in that one suite file), so
// readFacetsCase cannot resolve its schema and declines it (Fail) rather than
// guessing the base — an honest decline, not a false accept. The anyURI
// Facets/anyURI/anyURI_a*.xml and anyURI_b*.xml cases, honest declines from the
// #124 landing, are now DECIDED by their own reader and executor — see "The anyURI
// a*/b* multi-leaf cohort (issue #190)" above; readFacetsCase itself still decodes
// only the canonical single-<foo> shape and still declines everything else,
// including those eight files should they ever reach it. Of the anyURI/hexBinary/
// base64Binary cases, readFacetsCase decides the length/minLength/maxLength/
// enumeration ones in the canonical <test><foo> shape.

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
// dateTimeStamp (§3.4.28) is listed and, though it has ZERO cases in the current
// W3C checkout (no msData/datatypes/dateTimeStampNNN.xml), is now decided
// COMPLETELY: unlike every other cohort type its Parse-only path is not a complete
// check (it fixes explicitTimezone=required, a VALUE-based facet checked at
// cvc-datatype-valid §4.1.4 clause 3, cvc-explicitTimezone-valid §4.3.14.3, not a
// lexical/pattern check), so execLexicalCase routes any type whose EffectiveFacets
// fix a non-optional explicitTimezone (fixesTimezone) through the facet cohort's
// value.ValidateLexical path (decideLexicalByFacets) rather than parseOK. A
// tz-ABSENT dateTimeStamp literal is therefore correctly REJECTED (issue #140),
// closing the former fail-open; a future tz-absent case cannot regress the ratchet.
//
// The seven DERIVED integer-family members (byte, long, short, unsignedByte,
// unsignedInt, unsignedLong, unsignedShort — the families issue #331 claimed,
// not every family the checkout carries plain lexical cases for) joined the
// claim with that issue. They are
// not primitives, so the strict backend maps none of them, and it is exactly
// that miss which now ROUTES them to decideLexicalByFacets instead of declining
// them: value.governingMapping walks each one's base chain to xs:decimal's
// mapping and value.ValidateLexical then applies the type's OWN effective facets
// — the fixed pattern [\-+]?[0-9]+ (cvc-pattern-valid §4.3.4.4), fractionDigits=0
// (cvc-fractionDigits-valid §4.3.12.3) and the per-type bounds
// (cvc-min/maxInclusive-valid §4.3.10.3/§4.3.7.3) — which is what makes
// "128" against xs:byte a rejection rather than the false accept a Parse against
// xs:decimal's mapping alone would have produced (cvc-datatype-valid §4.1.4
// clauses 1 and 3, §3.4.13–§3.4.25). The same alternation also covers each
// family's LIST-variety sibling (byte009, long009, short009, unsignedByte007,
// unsignedInt007, unsignedLong007, unsignedShort007 — claimed by issue #224,
// which needed a named-file selector precisely because a family widening would
// then have dragged in the undecidable non-list siblings). Those seven still
// route through execLexicalCase's non-seeded fallback to execListCase, since
// their tested type decodes as the user-defined "myList", not a builtin.
// xs:int and xs:integer joined the same alternation with issue #365: their
// twenty-four plain lexical cases (int001–008, integer001–016 — #331's prose
// said "integer001–012", an undercount; integer013–016 are real files of the
// identical shape) ride the route above with NO engine change, only a wider
// selector. xs:integer is the first member to exercise that route's UNBOUNDED
// arm: it carries no min/maxInclusive at all (§3.4.13.3), just the fixed
// fractionDigits=0 and the pattern [\-+]?[0-9]+, so those two alone decide it.
// The pattern gate runs FIRST (cvc-datatype-valid §4.1.4 clause 1 before clause
// 3, and clause 3's V is "as determined by" clause 2), so integer012–016
// ("-1E4", "INF", "-INF", "NaN", "ABCDEF") are rejected as cvc-pattern-valid
// §4.3.4.4 failures, never as cvc-fractionDigits-valid §4.3.12.3 ones: none is
// in xs:decimal's lexical space (§3.3.3.1, no exponent/INF/NaN production)
// either, so no value V is ever established for the value-based facets to test.
// nonPositiveInteger, negativeInteger, nonNegativeInteger and positiveInteger
// joined it with issue #449 — their twenty plain lexical cases (each family's
// 001–005) — again with NO engine change, only a wider selector. They are the
// route's first HALF-bounded arm (exactly one of min/maxInclusive each,
// §3.4.14.3/§3.4.15.3/§3.4.20.3/§3.4.25.3) and, for negativeInteger and
// positiveInteger, its first TWO-HOP base chains (negativeInteger →
// nonPositiveInteger → integer → decimal; positiveInteger → nonNegativeInteger
// → integer → decimal), which governingMapping's st-restrict-facets §3.16.6.4
// walk crosses unchanged.
// The alternation lists "integer" before "int", and the two "non*" forms
// before the "negativeInteger"/"positiveInteger" they contain as suffixes, for
// the reader's sake only. Ordering cannot matter: RE2 has no leftmost-FIRST
// alternation, and more importantly the alternation is anchored between the
// literal msData/datatypes/ and [0-9]+\.xml$, so an alternative must consume
// the name from the character immediately after that literal — "negativeInteger"
// cannot reach nonNegativeInteger001.xml (that position reads n, o, n) and the
// bare "int" alternative can reach int<digits>.xml and nothing else. That was
// checked empirically for #449, not merely argued: a whole-tree match diff of
// the old pattern against the new one over testdata/xsdtests adds exactly the
// twenty files above and removes none. Facets/int/test111092.xml in particular
// stays claimed by no selector (TestDatatypesSelectorClaimsOnlyCohort pins the
// selector from both sides, including the cohort's own .xsd siblings).
var datatypesCase = regexp.MustCompile(`msData/datatypes/(boolean|decimal|string|float|double|anyURI|hexBinary|base64Binary|duration|dateTime|dateTimeStamp|time|date|gYearMonth|gYear|gMonthDay|gDay|gMonth|QName|NOTATION|unsignedByte|unsignedInt|unsignedLong|unsignedShort|nonNegativeInteger|nonPositiveInteger|negativeInteger|positiveInteger|byte|long|short|integer|int)[0-9]+\.xml$`)

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
// bindings — a context now carried per member on the facet (issue #152,
// xsd.EnumerationMember): decodeRestriction snapshots each <enumeration>'s namespace
// context down the schema document's ancestor chain and buildOwnFacets threads it
// into xsd.NewEnumerationMember, so value.newEnumFacet resolves each member against
// the right scope. QName's TESTED literals, by contrast, resolve their prefixes against the
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
// types per schema) is a distinct multi-type shape, claimed separately by
// d34Case/execD34Case (issue #162), not this selector.
var pdecimalCase = regexp.MustCompile(`saxonData/PDecimal/pdecimal[0-9]+\.[vn][0-9]+\.xml$`)

// d34Case matches an IBM D3_3_4 precisionDecimal instance case (issue #162):
// ibmData/{valid,instance_invalid}/D3_3_4/*.xml, discovered via the
// ibmMeta/precisionDecimal.testSet auxiliary index (runner.go). Unlike the Saxon
// PDecimal cohort's single-leaf model, one such schema declares SEVERAL simple
// types (single-step builtin restrictions, and — since issue #223 — lists and
// unions over them) and the instance's <root> carries MULTIPLE children, each
// dispatched to its own type by the child's declared type= attribute or by the
// global element its ref= names; execD34Case reads and decides that multi-type
// shape. The schema_invalid D3_3_4 cases are schema-kind and never
// reach this selector (kindInstance gate). NaN.xml in valid/D3_3_4 is referenced
// by no testGroup, so parseSuite emits no case for it and it never reaches here.
var d34Case = regexp.MustCompile(`ibmData/(valid|instance_invalid)/D3_3_4/.*\.xml$`)

// notationFacetsCase matches a NOTATION Facets-cohort instance (issue #153):
// msData/datatypes/Facets/NOTATION/NOTATION_<facet>NNN.xml. These fixtures use a
// two-step restriction shape (a named simpleType restricts xsd:NOTATION with the
// three enumerations jpeg/mpeg/g — §3.3.19 enumeration-required-notation makes a
// bare xsd:NOTATION unusable, so only such a derived type may be restricted
// further — then an anonymous attribute simpleType restricts THAT named type with
// one more tested facet), incompatible with facetsCase/decodeRestriction's
// single-step builtin-base shape, so they get their own selector and executor
// (execNotationFacetsCase) rather than being folded into facetsBaseTypes.
var notationFacetsCase = regexp.MustCompile(`msData/datatypes/Facets/NOTATION/NOTATION_[A-Za-z]+[0-9]+\.xml$`)

// anyURIShapeCase matches an anyURI a*/b* Facets-cohort instance (issue #190):
// msData/datatypes/Facets/anyURI/anyURI_{a,b}NNN.xml. These eight fixtures
// (a001/a002/a004, b001/b002/b004/b005/b006) share facetsCase's directory and
// filename form but NOT its instance shape — the tested values live in one or
// several <bar>/<foo> elements and in unqualified attributes, and the a* files
// declare their schema with the namespace-qualified xsi:schemaLocation — so they
// get their own reader and executor (readAnyURIShapeCase/execAnyURIShapeCase) and
// are dispatched BEFORE facetsCase, whose single-<foo> reader would decline them.
// See "The anyURI a*/b* multi-leaf cohort" above.
var anyURIShapeCase = regexp.MustCompile(`msData/datatypes/Facets/anyURI/anyURI_[ab][0-9]+\.xml$`)

// selectsDatatypes claims the instance cases of the lexical (integer family
// included since issue #331), facet, precisionDecimal, NOTATION-facet and
// anyURI-multi-leaf cohorts. It is a cheap
// path predicate; the executor does the real document reading. The
// anyURIShapeCase disjunct is stated even though facetsCase's pattern happens to
// match those eight paths too, so the claim rests on the cohort's own selector
// rather than on that overlap.
//
// Issue #224's separate integerListCase disjunct (the seven LIST-variety
// integer fixtures, named one by one so the family widening it feared could not
// drag in the then-undecidable non-list siblings) is gone: datatypesCase's
// integer-family alternation now covers those seven files too, and a second
// selector matching a strict subset of the first would be redundant state
// (STYLE D3). Their execution route is unchanged — see datatypesCase.
func selectsDatatypes(c caseSpec) bool {
	if c.kind != kindInstance {
		return false
	}
	doc := filepath.ToSlash(c.doc)
	return datatypesCase.MatchString(doc) || facetsCase.MatchString(doc) ||
		pdecimalCase.MatchString(doc) || notationFacetsCase.MatchString(doc) ||
		d34Case.MatchString(doc) || anyURIShapeCase.MatchString(doc)
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
		if d34Case.MatchString(doc) {
			return execD34Case(strictBackend, sym, c)
		}
		if notationFacetsCase.MatchString(doc) {
			return execNotationFacetsCase(strictBackend, sym, c)
		}
		// Before facetsCase: the eight anyURI a*/b* fixtures live under the Facets
		// directory and match facetsCase's filename form, but their multi-leaf shape
		// needs the dedicated reader (issue #190).
		if anyURIShapeCase.MatchString(doc) {
			return execAnyURIShapeCase(strictBackend, sym, c)
		}
		if facetsCase.MatchString(doc) {
			return execFacetsCase(strictBackend, sym, c)
		}
		return execLexicalCase(strictBackend, sym, c)
	}
}

// execLexicalCase decides a lexical-cohort case: an instance is valid iff every
// tested leaf value is Datatype Valid (cvc-datatype-valid §4.1.4) against the
// tested type. A type the backend maps DIRECTLY is its own {primitive type
// definition} and carries no own constraining facets in this cohort, so clause
// 2.1 alone decides it and value.Parse (parseOK) is a complete check; a type
// with no direct mapping — every derived builtin, the integer family foremost —
// is decided by its OWN effective facets over its nearest mapped ancestor's
// lexical space, which is the full value.ValidateLexical pipeline
// (decideLexicalByFacets), not Parse. See the routing comment below.
//
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
	st, seeded := sym[qn]
	if !seeded {
		// A non-seeded tested type is the list-variety cohort's signature: these
		// fixtures restrict a user-defined "myList" (an <xsd:list itemType=..>),
		// so decodeTestedPrimitive returns "myList" (or another non-builtin) here.
		// Fall back to execListCase, which does its own independent read of the
		// instance+schema (comp_foo and simpleTest can carry DIFFERENT own facets,
		// so each resolves its own type graph). A shape that is not a list case
		// declines there (Fail), so a genuinely non-list non-seeded case is
		// unaffected — the same honest-decline the other fallbacks use.
		return execListCase(backend, sym, c)
	}
	m, mapped := backend.Mapping(qn)
	// Context-dependent primitives (QName/NOTATION, §3.3.18/§3.3.19) resolve a
	// prefix against the in-scope namespace bindings at the literal, so they take
	// the contextual path with a real value.Context rather than the nil-context
	// value scan below (whose whiteSpace-only reading suffices for the
	// context-free primitives). Both ARE backend-mapped primitives (strict maps
	// all 20, guarded by TestDatatypesBackendSeeds); an unmapped one could not be
	// decided by either path below either — decideLexicalByFacets' nil context is
	// wrong for it — so it declines honestly rather than being mis-decided.
	if isContextDependent(prim) {
		if !mapped {
			return Fail()
		}
		return execContextualCase(m, prim, c)
	}
	// A tested type with no mapping ANYWHERE up its base chain has no lexical
	// space to be a member of, so neither path can decide it: an honest decline
	// (Fail). Since issue #331 an unmapped type reaches this decline only when it
	// is genuinely ungoverned — no seeded builtin is, since every one resolves to
	// a strict-mapped primitive, so this is a backend-gap net, not a live branch.
	if !mapped && !strictGoverns(backend, st) {
		return Fail()
	}
	// TWO INDEPENDENT reasons to leave the Parse-only path — keep them as an OR,
	// do not fold either into the other (issue #331; the oracle grounding on that
	// issue establishes the independence, and xs:dateTimeStamp satisfying both
	// today is a coincidence, not a subsumption):
	//
	//   - !mapped: the tested type is not its own {primitive type definition}, so
	//     Parse against the ancestor mapping that governs it (value.governingMapping,
	//     the widest-space rule of st-restrict-facets §3.16.6.4) satisfies only
	//     cvc-datatype-valid clause 2.1 — the type's OWN facets, clause 1's pattern
	//     and clause 3's value-based facets, are what actually decide it. For the
	//     integer family that is the fixed pattern [\-+]?[0-9]+ (cvc-pattern-valid
	//     §4.3.4.4), fractionDigits=0 (cvc-fractionDigits-valid §4.3.12.3) and the
	//     per-type bounds (cvc-min/maxInclusive-valid §4.3.10.3/§4.3.7.3): Parse
	//     against xs:decimal's mapping would FALSE-ACCEPT "128" as an xs:byte.
	//   - fixesTimezone: the type's effective facets fix explicitTimezone to a
	//     non-optional value (xs:dateTimeStamp fixes it to required, §3.4.28;
	//     §4.3.14.4 permits any date/time restriction to fix it too).
	//     cvc-explicitTimezone-valid (§4.3.14.3) is a VALUE-based facet checked at
	//     clause 3 AFTER the lexical mapping, so parseOK would FALSE-ACCEPT a
	//     tz-absent literal. This arm is about facet completeness, not about being
	//     unmapped: explicitTimezone lives only on the eight temporal primitives'
	//     applicable-facet rows (cos-applicable-facets §4.1.5), every one of them
	//     DIRECTLY mapped, so a user-defined date/time restriction that is itself
	//     mapped would still need it. Conversely the integer family never touches
	//     explicitTimezone (decimal's row has no such entry), so the first arm
	//     cannot be expressed through this one.
	if !mapped || fixesTimezone(st) {
		return decideLexicalByFacets(backend, st, values, c)
	}
	observedValid := true
	for _, v := range values {
		if !parseOK(m, prim, v, nil) {
			observedValid = false
			break
		}
	}
	if observedValid == c.expect.wantsValid() {
		return Pass()
	}
	return Fail()
}

// fixesTimezone reports whether st's effective facets fix explicitTimezone to a
// non-optional value (required or prohibited) — the class (xs:dateTimeStamp
// §3.4.28, or any date/time restriction that fixes it, §4.3.14.4) whose validity
// is NOT decided by lexical-space membership alone. cvc-explicitTimezone-valid
// (§4.3.14.3) is a VALUE-based facet, checked at cvc-datatype-valid §4.1.4 clause
// 3 after the lexical mapping, so the lexical cohort's parseOK does not enforce
// it. The fact "does this type constrain the timezone" already lives in
// EffectiveFacets (the seeded builtin carries the generated typespec's fixed
// facet), so this reads it there rather than re-deriving it from a type name
// (STYLE D1/D3). explicitTimezone is a supersede-kind facet, so at most one
// effective entry survives; an "optional" value (the date/time primitives'
// default) leaves lexical membership complete and takes the ordinary parseOK path.
func fixesTimezone(st *xsd.SimpleType) bool {
	for _, ef := range st.EffectiveFacets() {
		f := ef.Facet()
		if f.Kind() != xsd.FacetExplicitTimezone {
			continue
		}
		vals := f.Values()
		return len(vals) == 1 && vals[0] != "optional"
	}
	return false
}

// decideLexicalByFacets decides a lexical-cohort case whose tested type is not
// decided by lexical-space membership alone — it has no direct backend mapping,
// or it fixes a non-optional explicitTimezone, or both (execLexicalCase's routing
// comment explains why those two reasons stay independent, issue #331). Each
// tested value runs through the SAME pipeline the facet cohort uses
// (value.ValidateLexical): the type's own pattern (cvc-pattern-valid §4.3.4.4),
// then its {primitive type definition}'s lexical mapping reached by walking the
// base chain (value.governingMapping, cvc-datatype-valid §4.1.4 clause 2.1), then
// its own value-based facets (clause 3) — bounds, fractionDigits,
// explicitTimezone. The seeded builtin already carries all of them in its
// EffectiveFacets (the generated typespec's fixed facets), so no leaf synthesis
// is needed. Every type reaching here maps context-free (the two
// context-dependent primitives, §3.3.18/§3.3.19, are routed away upstream), so a
// nil value.Context suffices. The instance is valid iff every tested value
// validates, mirroring the parseOK path's whole-instance polarity.
func decideLexicalByFacets(backend value.Backend, st *xsd.SimpleType, values []string, c caseSpec) Status {
	observedValid := true
	for _, v := range values {
		if _, err := value.ValidateLexical(backend, st, v, nil); err != nil {
			mustNotBeFacetPrecondition(err, c, v)
			observedValid = false
			break
		}
	}
	if observedValid == c.expect.wantsValid() {
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
	if observedValid == c.expect.wantsValid() {
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
	if observedValid == c.expect.wantsValid() {
		return Pass()
	}
	return Fail()
}

// listTest is one tested element of a list-variety cohort case (issue #75): the
// local name of the list's item type (itemType, a builtin — a primitive for the
// #75 fixtures, a derived integer-family type for the #224 ones), the leaf restriction's own
// enumeration facet children (children, empty for the facet-free fixtures), and
// every tested lexical carried by that element (values). comp_foo carries one
// value; a simpleTest element may repeat (float039.xml has nine siblings), each
// captured so none is silently under-tested.
type listTest struct {
	itemType string
	children []facetChild
	values   []string
}

// execListCase decides a list-variety Facets cohort case (issues #75, #224): the
// msData/datatypes/{boolean018,float038,float039,anyURI011,hexBinary002,
// base64Binary002,duration027,byte009,long009,short009,unsignedByte007,
// unsignedInt007,unsignedLong007,unsignedShort007}.xml fixtures, each a
// <list itemType=..> (a
// user-defined "myList") reached through comp_foo and one-or-more simpleTest
// elements. For each tested element it resolves the item type to a seeded,
// strict-governed *xsd.SimpleType, synthesizes the list leaf (the item type as
// {item type definition}, xs:anySimpleType as base, the mandatory fixed
// whiteSpace=collapse plus any enumeration as own facets), and decides every
// tested value through the real facet pipeline (value.ValidateLexical): the
// list-variety mapping (cvc-datatype-valid clause dv_list §4.1.4, list.go)
// splits each lexical into items, each item is Datatype-Valid against the item
// type, and the leaf's enumeration is checked over the list value space by
// value-space "equal or identical" (cvc-enumeration-valid §4.3.5.4, §2.2.1/
// §2.2.2), its length (had any fixture one) in list items (cvc-length-valid
// §4.3.1.3). The whole case is valid iff EVERY tested value across comp_foo and
// every simpleTest sibling validates — the suite's whole-instance polarity. A
// case whose schema does not decode to the list shape, whose item type is
// not a seeded strict-governed builtin, or that carries an unrecognized facet is
// declined (Fail, an honest recorded gap) rather than mis-decided.
//
// The item type may be DERIVED, not only a primitive: the seven #224 fixtures
// list xs:byte/xs:long/xs:short/xs:unsignedByte/xs:unsignedInt/xs:unsignedLong/
// xs:unsignedShort, whose {primitive type definition} is xs:decimal. strictGoverns
// admits them (it resolves through the base chain, issue #81) and, because
// value.listMapping decides each item by the whole cvc-datatype-valid rule
// against the item type, the item type's OWN minInclusive/maxInclusive/
// fractionDigits/pattern facets are enforced per item (dv_list → dv_vfacets,
// §4.1.4) — not merely xs:decimal's lexical space.
//
// No msData UNION-variety fixture is decidable here: the value pipeline
// decides a union end to end since issue #223 (value/union.go), but this reader
// still recognizes only the <list itemType=..> "myList" shape.
func execListCase(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	tests, ok := readListCohortCase(c.doc)
	if !ok {
		return Fail()
	}
	observedValid := true
	for _, lt := range tests {
		qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: lt.itemType}
		item, seeded := sym[qn]
		if !seeded {
			return Fail()
		}
		if !strictGoverns(backend, item) {
			return Fail()
		}
		ownFacets, ok := buildListRestrictionFacets(lt.children)
		if !ok {
			return Fail()
		}
		constructed, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{},
			xsd.NewList(item), xsd.AnySimpleType(), constructedListFacets(), nil)
		if err != nil {
			return Fail()
		}
		leaf, err := xsd.NewSimpleType(xsderr.Loc{},
			xsd.QName{Space: synthNS, Local: "myList-" + lt.itemType},
			xsd.NewList(item), constructed, ownFacets, nil)
		if err != nil {
			return Fail()
		}
		for _, v := range lt.values {
			if _, verr := value.ValidateLexical(backend, leaf, v, nil); verr != nil {
				mustNotBeFacetPrecondition(verr, c, v)
				observedValid = false
				break
			}
		}
		if !observedValid {
			break
		}
	}
	if observedValid == c.expect.wantsValid() {
		return Pass()
	}
	return Fail()
}

// constructedListFacets is the {facets} of a CONSTRUCTED list — the
// <list itemType=..> step itself, whose {base type definition} is
// xs:anySimpleType. Structures §3.16.2.1 map.std.common case 3 manufactures
// exactly one member there, the whiteSpace facet with {value} = collapse and
// {fixed} = true (§4.3.6.1 f-w-fixed), and cos-st-restricts clause 2.2.1.2
// admits nothing else. It is spelled here because this cohort does not route
// through the generated builtin table, and a list-variety type with no whiteSpace
// mode in force violates value.ValidateLexical's facet precondition
// (value.effectiveWhiteSpace), which mustNotBeFacetPrecondition turns into a run
// failure rather than a scored verdict.
func constructedListFacets() []xsd.Facet {
	return []xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}
}

// buildListRestrictionFacets translates a list leaf restriction's facet children
// into the own facets of the SECOND derivation step — the <restriction base=..>
// hop over the constructed list, which is where the fixtures actually write
// them (boolean018.xsd: <simpleType name="myList"><list itemType="xsd:boolean"/>
// then <restriction base="myList"><enumeration …>). The two steps must stay
// apart: a constructed list may carry nothing but constructedListFacets
// (cos-st-restricts clause 2.2.1.2), and the enumeration belongs to the
// restricting type anyway, so this is the spec-accurate shape as well as the
// constructible one.
//
// Each enumeration child is collected via the shared enumerationMember helper and
// folded into one xsd.NewEnumerationFacet (§4.3.5.4); the fixtures' item types
// (boolean/float/anyURI/hexBinary/base64Binary/duration and the seven integer-family
// builtins) are never QName/NOTATION,
// so an enumeration member needs no namespace context (facetChild.bindings is
// nil here). Any non-enumeration facet kind is declined (ok=false), the cohort's
// honest-decline convention — no other facet kind appears in these fourteen fixtures.
// No enumeration child yields no own facets, a restriction step that narrows
// nothing.
func buildListRestrictionFacets(children []facetChild) ([]xsd.Facet, bool) {
	var enumMembers []xsd.EnumerationMember
	for _, ch := range children {
		if ch.name != "enumeration" {
			return nil, false
		}
		enumMembers = append(enumMembers, enumerationMember(ch))
	}
	if len(enumMembers) == 0 {
		return nil, true
	}
	return []xsd.Facet{xsd.NewEnumerationFacet(enumMembers)}, true
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
		xsd.NewAtomic(primitiveOfType(builtinType)), builtinType, ownFacets, nil)
	if err != nil {
		return Fail()
	}
	_, verr := value.ValidateLexical(backend, leaf, raw, ctx)
	mustNotBeFacetPrecondition(verr, c, raw)
	observedValid := verr == nil
	if observedValid == c.expect.wantsValid() {
		return Pass()
	}
	return Fail()
}

// execNotationFacetsCase decides a NOTATION Facets-cohort case (issue #153). The
// fixture shape is a two-step restriction: a named simpleType restricts
// xsd:NOTATION with the enumerations jpeg/mpeg/g (the ONLY way to make NOTATION
// usable, §3.3.19 enumeration-required-notation), and an anonymous attribute
// simpleType restricts THAT with one more tested facet
// (length/minLength/maxLength/pattern/enumeration). The tested value lives in the
// <foo> element's attrTest attribute. The executor synthesizes the corresponding
// two-level chain — a middle type (seeded NOTATION as base and {primitive type
// definition}, the step-0 enumerations as ownFacets) and a leaf (middle as base,
// the step-1 tested facet as ownFacets) — and decides validity through the real
// facet pipeline (value.ValidateLexical). xsd.EffectiveFacets' existing
// supersede-overlay realizes every verdict: an own enumeration supersedes the
// middle's (so a leaf enumeration=[mpeg] narrows to {mpeg} alone), while a
// length/pattern leaf inherits the middle's {jpeg,mpeg,g} enumeration unchanged;
// the length facets are vacuous over NOTATION (§4.3.1.3 clause 1.3, the QName/
// NOTATION exemption value.lengthFacet already realizes). No cvc-* rule is new:
// cvc-datatype-valid (§4.1.4), cvc-pattern-valid (§4.3.4.4) and
// cvc-enumeration-valid (§4.3.5.4) are the only rules in play, all already wired.
// The <xsd:notation> component declarations are NOT load-bearing for any instance
// verdict (§3.14.1's "must name a declared notation" is a schema-construction SCC
// satisfied by every fixture and irrelevant to instance-side membership), so they
// are deliberately not parsed (STYLE D4). A case whose schema does not decode to
// the two-step shape, whose base step does not restrict NOTATION, or that pairs an
// inapplicable facet with NOTATION is declined (Fail, a recorded gap).
func execNotationFacetsCase(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	raw, baseChildren, leafChildren, ctx, ok := readNotationFacetsCase(c.doc)
	if !ok {
		return Fail()
	}
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: "NOTATION"}
	notationType, seeded := sym[qn]
	if !seeded {
		return Fail()
	}
	if !strictGoverns(backend, notationType) {
		return Fail()
	}
	// Middle: the named simpleType restricting xsd:NOTATION with jpeg/mpeg/g. Its
	// own facets and the leaf's are both governed by NOTATION's applicable-facet set
	// (cos-applicable-facets §4.1.5 — a restriction never widens what applies,
	// §3.16.6.3), so both buildOwnFacets calls check against "NOTATION".
	middleFacets, ok := buildOwnFacets("NOTATION", baseChildren)
	if !ok {
		return Fail()
	}
	middle, err := xsd.NewSimpleType(xsderr.Loc{},
		xsd.QName{Space: synthNS, Local: "NOTATION-notation"},
		xsd.NewAtomic(primitiveOfType(notationType)), notationType, middleFacets, nil)
	if err != nil {
		return Fail()
	}
	leafFacets, ok := buildOwnFacets("NOTATION", leafChildren)
	if !ok {
		return Fail()
	}
	leaf, err := xsd.NewSimpleType(xsderr.Loc{},
		xsd.QName{Space: synthNS, Local: "NOTATION-facets"},
		xsd.NewAtomic(primitiveOfType(notationType)), middle, leafFacets, nil)
	if err != nil {
		return Fail()
	}
	_, verr := value.ValidateLexical(backend, leaf, raw, ctx)
	mustNotBeFacetPrecondition(verr, c, raw)
	observedValid := verr == nil
	if observedValid == c.expect.wantsValid() {
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
//
// # The IBM D3_3_4 precisionDecimal cohort (issue #162)
//
// The lane also claims the IBM precisionDecimal instance cases under
// ibmData/{valid,instance_invalid}/D3_3_4/*.xml (discovered via the same
// ibmMeta/precisionDecimal.testSet auxiliary index, runner.go). This is a
// SECOND, materially larger precisionDecimal shape than the Saxon cohort's
// single-leaf model: ONE schema declares SEVERAL simple types — single-step
// restrictions of a builtin (decType/decTotalDigits/decEnumeration/decPattern/
// decMinMaxInclusive/decMinMaxExclusive/decMinMaxScale), <list>s over them (v16)
// and <union>s over them (v17) — and the instance's <root> carries MULTIPLE
// children, each dispatched to its OWN type by the element's declared type=
// attribute (the element→type binding is NOT a naming convention — v14's
// decMinMaxScale type is carried by element elMinMaxScale — so the reader
// resolves each child's real declared type=, never guesses from the element
// name) or by the global element its ref= names. readD34Case resolves the schema
// from the instance's own xsi:schemaLocation (a namespace+location pair, unlike
// the Saxon cohort's filename-derived path — d3_3_4ii01a.xml shares
// d3_3_4ii01.xsd), collects every simple type the schema declares, finds
// <element name="root">'s sequence (inline, or through the named complexType its
// type= references) and routes each instance child to its declared type.
// execD34Case then builds one component per declaration it can back
// (buildD34Types) and ANDs value.ValidateLexical over every child's value — the
// document is valid iff EVERY child validates, the same whole-document polarity.
// The spec rules are exactly those the Saxon cohort already enforces plus the two
// {variety} clauses of cvc-datatype-valid the value package backs end to end, and
// no new library rule ID: cvc-datatype-valid (§4.1.4) with clause dv_list (cl.2.2,
// issue #75) for v16 and clause dv_union (cl.2.3, issue #223) for v17,
// cvc-pattern-valid (§4.3.4.4), cvc-enumeration-valid (§4.3.5.4, value-space
// equal-or-identical on ·numericalValue·: 10 == 1.0e1, NaN matches NaN by
// identity), the four bound facets over the PARTIAL order
// (cvc-min/maxInclusive/Exclusive-valid §4.3.7–4.3.10; NaN incomparable ⇒ fails
// every bound, §3.1), cvc-totalDigits-valid (xsd-precisionDecimal.md §4.1.1,
// zero AND the specials vacuously pass) and cvc-maxScale/minScale-valid
// (§4.2.3/§4.3.3; the specials' absent ·scale· is exempt, but zero is NOT — #133).
// Decidable: d3_3_4v14/v16/v17/v23/v24, d3_3_4v18 and v19–v22 (multi-step
// restriction chains, whose types restrict other SCHEMA types rather than a seeded
// builtin — decidable since issue #574 widened buildD34Type's base resolution),
// d3_3_4ii01[,a-f] and d3_3_4ii02 (the ii01/ii02 schemas are shared across sibling
// instance files). Declined honestly (a recorded gap in the datatypes lane, never
// mis-decided): v15, whose root's sequence children are typed by a COMPLEX type, so
// no simple-type declaration governs them — it falls out of scope naturally,
// without special-casing, because its types simply fail to build. That gap is
// recorded in the DATATYPES lane — this repo's most complete lane, where the
// residual failures number in the tens against over a thousand passes — so it costs
// a real point. (The INSTANCE lane, 0/26426, is the inert one; naming it here, as
// this comment once did, would tell a reader the decline is free.) NaN.xml in
// valid/D3_3_4 is not referenced by any testGroup, so parseSuite never emits a case
// for it and the selector never sees it.
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
		xsd.NewAtomic(primitiveOfType(builtinType)), builtinType, ownFacets, nil)
	if err != nil {
		return Fail()
	}
	// precisionDecimal maps context-free (§3.2), so a nil value.Context suffices —
	// unlike the QName cohort, no prefix resolution is involved.
	observedValid := true
	for _, v := range values {
		if _, verr := value.ValidateLexical(backend, leaf, v, nil); verr != nil {
			mustNotBeFacetPrecondition(verr, c, v)
			observedValid = false
			break
		}
	}
	if observedValid == c.expect.wantsValid() {
		return Pass()
	}
	return Fail()
}

// execD34Case decides an IBM D3_3_4 precisionDecimal cohort case (issue #162,
// widened to the list and union varieties by issue #223). Unlike
// execPDecimalCase's single-leaf model, one schema declares SEVERAL simple types
// and the instance's <root> holds MULTIPLE children, each bound to its OWN type by
// the child's declared type= (or by the global element its ref= names). readD34Case
// resolves those bindings; buildD34Types turns every declaration it can back into
// a real *xsd.SimpleType; execD34Case then ANDs value.ValidateLexical over every
// child's value. The instance is valid iff EVERY child validates — the suite's
// whole-document polarity.
//
// The pipeline realizes precisionDecimal's spec-exact facet semantics unchanged
// (the same rules the Saxon cohort enforces, no new library rule ID): totalDigits
// vacuously passes zero and the specials (§4.1.1), the four bound facets fail NaN
// over the partial order (§3.1), maxScale/minScale skip the specials' absent
// ·scale· (#133), and enumeration matches value-space equal-or-identical on
// ·numericalValue· (§4.3.5.4). The two variety cases add exactly the two clauses of
// cvc-datatype-valid the value package now backs end to end: dv_list (§4.1.4 cl.2.2,
// issue #75) for v16 and dv_union (cl.2.3, issue #223) for v17, whose unions mix
// precisionDecimal with xs:string and xs:integer members and so exercise
// first-accepting-member dispatch across three different value spaces.
//
// A case whose shape readD34Case declines, or ANY of whose types in use fails to
// build (v15's complexType-typed children), is declined — Fail, an honest recorded
// gap — rather than partially decided.
func execD34Case(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	decls, elems, ok := readD34Case(c.doc)
	if !ok {
		return Fail()
	}
	// Build up front, over the whole declaration set, so a type in use that cannot
	// be backed declines the case before any value is decided rather than silently
	// skipping that child. leaves is a lookup, never ranged into output (STYLE D2);
	// the validation loop below ranges elems, a slice in document order.
	leaves := buildD34Types(backend, sym, decls)
	for _, e := range elems {
		if _, built := leaves[e.typeKey]; !built {
			return Fail()
		}
	}
	// Every type in this cohort maps context-free (precisionDecimal §3.2, string,
	// integer), so a nil value.Context suffices — no prefix resolution is involved.
	observedValid := true
	for _, e := range elems {
		if _, verr := value.ValidateLexical(backend, leaves[e.typeKey], e.value, nil); verr != nil {
			mustNotBeFacetPrecondition(verr, c, e.value)
			observedValid = false
			break
		}
	}
	if observedValid == c.expect.wantsValid() {
		return Pass()
	}
	return Fail()
}

// buildD34Types turns the schema's simple-type declarations into real
// *xsd.SimpleType components, keyed by d34TypeDecl.key. A declaration whose shape
// this cohort cannot back — anything but a single-step restriction of a seeded
// builtin, a list, or a union over those — is simply absent from the result, which
// is how the out-of-scope D3_3_4 shapes decline without being special-cased.
//
// Declarations may reference one another (v16's decListC lists sv:decTotalDigits,
// v17's decUnionC unions sv:decMinScale), so the pass repeats over the ordered
// declarations until a round builds nothing new: a declaration whose references
// are not built YET simply fails this round and is retried in the next. That fixed
// point needs no cycle check and no depth counter — a reference cycle just stops
// making progress and leaves its types unbuilt, which the caller reads as a
// decline. Iterating the ORDERED slice (never the result map) keeps which
// declaration is attempted first deterministic (STYLE D2).
func buildD34Types(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, decls []d34TypeDecl) map[string]*xsd.SimpleType {
	built := make(map[string]*xsd.SimpleType, len(decls))
	for progress := true; progress; {
		progress = false
		for _, d := range decls {
			if _, done := built[d.key]; done {
				continue
			}
			st, ok := buildD34Type(backend, sym, decls, built, d.key, d.decl)
			if !ok {
				continue
			}
			built[d.key] = st
			progress = true
		}
	}
	return built
}

// buildD34Type builds ONE declared simple type. It backs exactly three shapes,
// each mapping onto a {variety} the value pipeline decides end to end:
//
//   - <list>: {item type definition} from itemType= or the inline <simpleType>,
//     carrying the fixed whiteSpace=collapse every list has (§4.3.6.1 f-w-fixed,
//     buildListOwnFacets) — decided through cvc-datatype-valid clause dv_list
//     (§4.1.4 cl.2.2).
//   - <union>: {member type definitions} from memberTypes= then the inline
//     <simpleType> children, in that order (map.std.union clause 1), facet-free
//     because a union constructed from xs:anySimpleType must be
//     (cos-st-restricts clause 3.2.1.2) — decided through clause dv_union (cl.2.3),
//     whose first-accepting-member scan reads exactly that order.
//   - <restriction base="B">, where B is resolved through d34TypeRef exactly as a
//     list's itemType= or a union's memberTypes= entry is: a type the schema itself
//     DECLARES (already built by an earlier round of the fixed point) or, failing
//     that, a seeded builtin. Either way the resolved component must be
//     strict-governed. B being a declared type is what admits a MULTI-STEP chain
//     (issue #574: v18 and v19–v22 restrict a restriction, one to three steps deep,
//     ultimately grounded in a builtin), with no new ordering logic — a base not
//     built YET declines this round and the caller's fixed point retries it.
//     dt-derived (§2.4.3) puts no cap on chain depth, and the facet overlay across
//     the chain is xsd.NewSimpleType's (st-restrict-facets/key-facets-overlay
//     §3.16.6.4), never re-implemented here. The base itself stays the leaf's
//     {base type definition}, but facet APPLICABILITY is computed from the chain's
//     ultimate primitive (primitiveOfType), because cos-applicable-facets (§4.1.5)
//     keys on {primitive type definition}, which st-restrict-facets clause 2
//     (§3.16.6.2) holds invariant at every step of an atomic chain.
//
// ok is false for any other shape, and equally for a reference that is declared but
// not yet built — the caller's fixed point retries those.
func buildD34Type(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, decls []d34TypeDecl, built map[string]*xsd.SimpleType, key string, decl d34SimpleType) (*xsd.SimpleType, bool) {
	name := xsd.QName{Space: synthNS, Local: "d34-" + key}
	if decl.List != nil {
		item, ok := d34ItemOrMember(backend, sym, decls, built, key+"-item", decl.List.ItemType, decl.List.Item)
		if !ok {
			return nil, false
		}
		return newD34SimpleType(name, xsd.NewList(item), xsd.AnySimpleType(), constructedListFacets())
	}
	if decl.Union != nil {
		members, ok := d34UnionMembers(backend, sym, decls, built, key, *decl.Union)
		if !ok {
			return nil, false
		}
		return newD34SimpleType(name, xsd.NewUnion(members...), xsd.AnySimpleType(), nil)
	}
	if decl.Restriction == nil {
		return nil, false
	}
	base, ok := d34TypeRef(backend, sym, decls, built, decl.Restriction.Base)
	if !ok || !strictGoverns(backend, base) {
		return nil, false
	}
	primitive := primitiveOfType(base)
	if primitive == nil {
		return nil, false
	}
	children := make([]facetChild, 0, len(decl.Restriction.Facets))
	for _, f := range decl.Restriction.Facets {
		children = append(children, facetChild{name: f.XMLName.Local, value: f.Value})
	}
	ownFacets, ok := buildOwnFacets(primitive.Name().Local, children)
	if !ok {
		return nil, false
	}
	return newD34SimpleType(name, xsd.NewAtomic(primitive), base, ownFacets)
}

// newD34SimpleType constructs one synthesized cohort type, turning the
// constructor's error into the cohort's honest ok=false decline: a component the
// xsd constructors reject (an inapplicable facet, a member blocked by {final})
// is a shape this executor cannot decide, not a verdict about instance data.
func newD34SimpleType(name xsd.QName, variety xsd.Variety, base *xsd.SimpleType, ownFacets []xsd.Facet) (*xsd.SimpleType, bool) {
	st, err := xsd.NewSimpleType(xsderr.Loc{}, name, variety, base, ownFacets, nil)
	if err != nil {
		return nil, false
	}
	return st, true
}

// d34UnionMembers resolves a <union>'s {member type definitions} in map.std.union
// clause 1 order: the memberTypes= references first, in attribute order, then the
// inline <simpleType> children in document order. That order is the one dv_union
// scans for the ·active member type· (§4.1.4 cl.2.3), so it is load-bearing, not
// cosmetic. ok is false if any member fails to resolve, so a partially resolved
// union is never built.
func d34UnionMembers(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, decls []d34TypeDecl, built map[string]*xsd.SimpleType, key string, u d34Union) ([]*xsd.SimpleType, bool) {
	refs := strings.Fields(u.MemberTypes)
	members := make([]*xsd.SimpleType, 0, len(refs)+len(u.Members))
	for _, ref := range refs {
		m, ok := d34TypeRef(backend, sym, decls, built, ref)
		if !ok {
			return nil, false
		}
		members = append(members, m)
	}
	for i, inline := range u.Members {
		m, ok := buildD34Type(backend, sym, decls, built, fmt.Sprintf("%s-member%d", key, i), inline)
		if !ok {
			return nil, false
		}
		members = append(members, m)
	}
	return members, true
}

// d34ItemOrMember resolves the single type a <list> names either way: by itemType=
// reference or as its one anonymous inline <simpleType> child (v16 uses both
// spellings). Exactly one must be present — a <list> with neither, or with both, is
// not a shape this executor decides.
func d34ItemOrMember(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, decls []d34TypeDecl, built map[string]*xsd.SimpleType, key, ref string, inline []d34SimpleType) (*xsd.SimpleType, bool) {
	if ref != "" && len(inline) == 0 {
		return d34TypeRef(backend, sym, decls, built, ref)
	}
	if ref == "" && len(inline) == 1 {
		return buildD34Type(backend, sym, decls, built, key, inline[0])
	}
	return nil, false
}

// d34TypeRef resolves a type REFERENCE — a list's itemType=, a union's memberTypes=
// entry — to a component. A name the schema itself DECLARES resolves only once that
// declaration is built (ok=false meanwhile, which the fixed point retries);
// otherwise it must be a seeded builtin the strict backend governs, so a reference
// to an unbacked type declines rather than silently validating through some other
// mapping. Schema declarations are consulted FIRST so a schema-local type can never
// be shadowed by a same-named builtin.
//
// Matching on the local name alone is sound here for the same reason it is
// throughout this cohort's readers: each fixture has one target namespace, one
// prefix bound to it, and the XML Schema namespace as its default.
func d34TypeRef(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, decls []d34TypeDecl, built map[string]*xsd.SimpleType, ref string) (*xsd.SimpleType, bool) {
	name := localName(ref)
	if d34Declared(decls, name) {
		st, done := built[name]
		return st, done
	}
	st, seeded := sym[xsd.QName{Space: xsd.XMLSchemaNS, Local: name}]
	if !seeded || !strictGoverns(backend, st) {
		return nil, false
	}
	return st, true
}

// execAnyURIShapeCase decides an anyURI a*/b* cohort case (issue #190): every
// tested leaf the instance carries — each element's character content and each
// unqualified attribute's value — is validated against the simple type its own
// declaration binds, and the instance is valid iff EVERY leaf is
// (cvc-enumeration-valid §4.3.5.4 over cvc-datatype-valid §4.1.4, the same
// whole-document polarity execPDecimalCase/execD34Case use). Per-leaf type
// resolution is load-bearing rather than cosmetic: anyURI_b006's <foo> children
// are typed xsd:anyURI DIRECTLY (facet-free, so every Char* value is valid per
// §3.3.17.2) while its <bar> siblings carry the enumeration-restricted named type,
// so validating both against one type would reach the right verdict for the wrong
// reason.
//
// A leaf whose type is not a single-step anyURI restriction (or the bare builtin),
// an element occurrence the schema does not declare, or an attribute the element's
// type does not declare all DECLINE the whole case (Fail, a recorded gap) — the
// cohort's honest-decline convention — rather than leaving a value unchecked.
//
// One claimed case, anyURI_a004_1339.i, is a KNOWN suite quirk: all seven of its
// leaves (six <bar> values plus the <root> att attribute) ARE enumeration members,
// so the spec-correct verdict is VALID, yet the suite declares the instance
// invalid — an expectation its own testGroup annotation contradicts ("Schema doc
// changed to guaranteed-to-fail URIs, but that does not make the schema (or
// instance) invalid") and which has carried status="queried"
// (bugzilla 4126) since 2010. Per the pdecimal006.n2 precedent the executor keeps
// the spec-correct verdict, so this one case records an honest Fail rather than a
// false Pass bought by mis-implementing the enumeration check.
func execAnyURIShapeCase(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	typeFacets, leaves, ok := readAnyURIShapeCase(c.doc)
	if !ok {
		return Fail()
	}
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyURI"}
	builtinType, seeded := sym[qn]
	if !seeded {
		return Fail()
	}
	if !strictGoverns(backend, builtinType) {
		return Fail()
	}
	// One synthesized leaf per DISTINCT named type in use, keyed by name and built up
	// front over the document-ordered leaves, so a construction failure for any type
	// in use declines the whole case before any value is decided. The bare builtin
	// key resolves to the SEEDED anyURI type itself — a facet-free element needs no
	// synthesis. synth is a lookup, never ranged into output (STYLE D2); the
	// validation loop below ranges leaves, a slice in document order.
	synth := map[string]*xsd.SimpleType{anyURIBuiltinKey: builtinType}
	for _, l := range leaves {
		if _, built := synth[l.typeName]; built {
			continue
		}
		ownFacets, ok := buildOwnFacets("anyURI", typeFacets[l.typeName])
		if !ok {
			return Fail()
		}
		leaf, err := xsd.NewSimpleType(xsderr.Loc{},
			xsd.QName{Space: synthNS, Local: "anyURI-" + l.typeName},
			xsd.NewAtomic(primitiveOfType(builtinType)), builtinType, ownFacets, nil)
		if err != nil {
			return Fail()
		}
		synth[l.typeName] = leaf
	}
	// anyURI maps context-free (§3.3.17.2, the identity on Char*), so a nil
	// value.Context suffices. Values are passed RAW: ValidateLexical's whiteSpace
	// stage applies anyURI's fixed collapse (§4.3.6), which anyURI_b005 turns on —
	// its "http://a/x  y" collapses to "http://a/x y", still not the enumeration's
	// "http://a/x%20y" (§3.3.17.2 Note: no percent-decoding), so the case is invalid.
	observedValid := true
	for _, l := range leaves {
		if _, verr := value.ValidateLexical(backend, synth[l.typeName], l.value, nil); verr != nil {
			mustNotBeFacetPrecondition(verr, c, l.value)
			observedValid = false
			break
		}
	}
	if observedValid == c.expect.wantsValid() {
		return Pass()
	}
	return Fail()
}

// mustNotBeFacetPrecondition converts this lane's applicability OWNERSHIP from prose
// into an executed assertion. Every leaf reaching value.ValidateLexical here is
// synthesized through buildOwnFacets, which declines a case pairing a facet with a
// primitive the generated builtin table says it does not apply to
// (cos-applicable-facets §4.1.5), or through buildListRestrictionFacets, which admits
// enumeration and nothing else — one of the seven facets §4.1.5 makes applicable to a
// list. Every leaf likewise carries the whiteSpace facet its variety requires
// (§3.16.7.4, §4.3.6.1, constructedListFacets). So no synthesized type can violate
// value.ValidateLexical's facet precondition, and value.IsFacetPrecondition must be
// false at every call site in this file.
//
// If it is ever true, the leaf-synthesis logic has a hole, and the case must NOT be
// scored: the fault is returned as an error, so a site that merely tested `err != nil`
// would read it as "this instance is invalid" and, for an .nK case, AGREE with the
// suite for entirely the wrong reason — a false pass the ratchet would then bank
// permanently. Panicking fails the run loudly instead, the same stance the lane's
// Seed precondition takes: a programming error in the harness, not a runtime
// condition.
func mustNotBeFacetPrecondition(err error, c caseSpec, lexical string) {
	if !value.IsFacetPrecondition(err) {
		return
	}
	panic(fmt.Sprintf("conformance: case %s: value.ValidateLexical reported a facet-pipeline precondition fault on %q, but this executor owns cos-applicable-facets applicability (buildOwnFacets), so the synthesized type is a harness bug and the case must not be scored: %v", c.id, lexical, err))
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
//
// Reading the row's OWN whiteSpace value is enough because prim always names a
// type the backend maps DIRECTLY (every parseOK call site gates on
// backend.Mapping), and strict maps only atomic primitives. The three
// list-variety builtins — whose rows deliberately carry no own whiteSpace value,
// since it belongs to their anonymous intermediate list (§3.16.2.1 case 3) — are
// unmapped and so never reach here.
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

// readListCohortCase reads one list-variety cohort instance+schema (issue #75)
// into one listTest per tested element (comp_foo and every simpleTest sibling).
// It does its OWN independent read — not readLexicalCase's shared prim — because
// comp_foo and simpleTest can resolve to DIFFERENT own facets (boolean018:
// comp_foo enumerates 0/1, simpleTest enumerates true/false), so each type graph
// must be resolved independently. ok is false when the instance or schema does
// not decode to this cohort's shape, an honest decline rather than a guess.
//
// The schema is a whole-document xml.Unmarshal (these fixtures are tiny). Type
// resolution walks: complexTest's type → its named complexType → the single
// comp_foo sequence child → either its inline anonymous simpleType or its named
// type; simpleTest's type → the named simpleType. Each then resolves to the
// <list itemType=..> through AT MOST one <restriction base=..> hop
// (resolveListLeaf), collecting that leaf restriction's own <enumeration>
// children; an intermediate step's facets are not merged (none of the seven
// in-scope fixtures need more than the single leaf restriction level with facets).
func readListCohortCase(instancePath string) (tests []listTest, ok bool) {
	inst, err := decodeListInstance(instancePath)
	if err != nil || inst.SchemaLoc == "" {
		return nil, false
	}
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(inst.SchemaLoc))
	sch, err := decodeListSchema(schemaPath)
	if err != nil {
		return nil, false
	}
	// Lookups built once from the parsed document (D2: maps are pure lookups; the
	// output order comes from appending comp_foo then simpleTest below).
	elementType := map[string]string{}
	for _, e := range sch.Elements {
		elementType[e.Name] = e.Type
	}
	complexTypes := map[string]listSchemaComplexType{}
	for _, ct := range sch.ComplexTypes {
		complexTypes[ct.Name] = ct
	}
	simpleTypes := map[string]listSchemaSimpleType{}
	for _, st := range sch.SimpleTypes {
		simpleTypes[st.Name] = st
	}

	// comp_foo: complexTest's complexType, its comp_foo sequence child, then that
	// child's inline simpleType or its named type.
	ct, found := complexTypes[localName(elementType["complexTest"])]
	if !found {
		return nil, false
	}
	var compStart listSchemaSimpleType
	switch {
	case ct.Element.SimpleType != nil:
		compStart = *ct.Element.SimpleType
	default:
		compStart, found = simpleTypes[localName(ct.Element.Type)]
		if !found {
			return nil, false
		}
	}
	compItem, compChildren, ok := resolveListLeaf(compStart, simpleTypes)
	if !ok {
		return nil, false
	}

	// simpleTest: its named simpleType.
	simpStart, found := simpleTypes[localName(elementType["simpleTest"])]
	if !found {
		return nil, false
	}
	simpItem, simpChildren, ok := resolveListLeaf(simpStart, simpleTypes)
	if !ok {
		return nil, false
	}

	return []listTest{
		{itemType: compItem, children: compChildren, values: []string{inst.ComplexTest.CompFoo}},
		{itemType: simpItem, children: simpChildren, values: inst.SimpleTest},
	}, true
}

// resolveListLeaf resolves a starting simpleType node (inline or named) to the
// local name of its list item type and the leaf restriction's own enumeration
// facet children.
// It collects the leaf's own <enumeration> children (only the first restriction
// step, never an intermediate one — §3.16.1: the item type is never itself a
// list, so the chain is a single restriction hop to the <list> in this cohort),
// then walks the <restriction base=..> chain (capped at a small fixed depth as a
// defensive guard against a malformed/cyclic fixture — trusted test inputs, so
// this is not load-bearing) until it reaches the <list itemType=..> declaration.
// ok is false for a node that is neither a list nor a recognized restriction, a
// restriction carrying a non-enumeration facet child, an unresolvable base, or an
// absent itemType.
func resolveListLeaf(start listSchemaSimpleType, simpleTypes map[string]listSchemaSimpleType) (itemType string, children []facetChild, ok bool) {
	cur := start
	collectedLeaf := false
	for depth := 0; depth < 4; depth++ {
		if cur.List != nil {
			item := localName(cur.List.ItemType)
			if item == "" {
				return "", nil, false
			}
			return item, children, true
		}
		if cur.Restriction == nil {
			return "", nil, false
		}
		if len(cur.Restriction.Other) > 0 {
			return "", nil, false
		}
		if !collectedLeaf {
			for _, e := range cur.Restriction.Enumerations {
				children = append(children, facetChild{name: "enumeration", value: e.Value})
			}
			collectedLeaf = true
		}
		next, found := simpleTypes[localName(cur.Restriction.Base)]
		if !found {
			return "", nil, false
		}
		cur = next
	}
	return "", nil, false
}

// listInstance mirrors the list-variety cohort's instance shape: a root carrying
// one complexTest/comp_foo value and one-or-more simpleTest values. SimpleTest is
// a []string so EVERY sibling is captured (float039.xml has nine); a scalar field
// would silently keep only the last, under-testing the case.
type listInstance struct {
	SchemaLoc   string `xml:"http://www.w3.org/2001/XMLSchema-instance noNamespaceSchemaLocation,attr"`
	ComplexTest struct {
		CompFoo string `xml:"comp_foo"`
	} `xml:"complexTest"`
	SimpleTest []string `xml:"simpleTest"`
}

func decodeListInstance(path string) (listInstance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return listInstance{}, err
	}
	var inst listInstance
	if err := xml.Unmarshal(data, &inst); err != nil {
		return listInstance{}, err
	}
	return inst, nil
}

// listSchema mirrors the list-variety cohort's schema shape: top-level element
// declarations, named complexTypes (each a sequence with one element child that
// carries a type= or an inline simpleType), and named simpleTypes (each a <list>
// or a <restriction>). Only direct schema children are captured, so the anonymous
// complexType nested inside <element name='root'> is not (it carries element
// refs, not a tested type). Slice fields preserve document order (D2).
type listSchema struct {
	Elements []struct {
		Name string `xml:"name,attr"`
		Type string `xml:"type,attr"`
	} `xml:"element"`
	ComplexTypes []listSchemaComplexType `xml:"complexType"`
	SimpleTypes  []listSchemaSimpleType  `xml:"simpleType"`
}

// listSchemaComplexType is a named complexType whose one sequence element child
// (comp_foo) carries either a named type= or an inline anonymous simpleType.
type listSchemaComplexType struct {
	Name    string `xml:"name,attr"`
	Element struct {
		Name       string                `xml:"name,attr"`
		Type       string                `xml:"type,attr"`
		SimpleType *listSchemaSimpleType `xml:"simpleType"`
	} `xml:"sequence>element"`
}

// listSchemaSimpleType is a named-or-inline simpleType: either a <list itemType=..>
// or a <restriction base=..> with zero-or-more <enumeration> children. Other
// captures any restriction child that is NOT an enumeration, so resolveListLeaf can
// decline a fixture carrying an unrecognized facet rather than guess.
type listSchemaSimpleType struct {
	Name string `xml:"name,attr"`
	List *struct {
		ItemType string `xml:"itemType,attr"`
	} `xml:"list"`
	Restriction *struct {
		Base         string `xml:"base,attr"`
		Enumerations []struct {
			Value string `xml:"value,attr"`
		} `xml:"enumeration"`
		Other []struct {
			XMLName xml.Name
		} `xml:",any"`
	} `xml:"restriction"`
}

func decodeListSchema(path string) (listSchema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return listSchema{}, err
	}
	var s listSchema
	if err := xml.Unmarshal(data, &s); err != nil {
		return listSchema{}, err
	}
	return s, nil
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
// its element local name (e.g. "length"), its value attribute, and — for an
// <enumeration> over QName/NOTATION — the namespace bindings in scope where that
// element was written (§3.3.18), so buildOwnFacets can build an
// xsd.EnumerationMember carrying the DECLARING schema's context. bindings is the
// innermost-wins snapshot decodeRestriction accumulates down the schema
// document's ancestor chain (empty-prefix key "" holds the default namespace); it
// is nil for facet children that carry no context (every non-enumeration facet,
// and the precisionDecimal path, whose members are context-free). It is a
// lookup-only map, never ranged into output (STYLE D2).
type facetChild struct {
	name     string
	value    string
	bindings map[string]string
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

// mergesRepeatedChildren reports whether several children of ONE <restriction>
// naming kind map to a SINGLE facet of that kind. Only pattern and enumeration
// do: §4.3.4.2 case 2 concatenates every <pattern> sibling's value into one
// regular expression with multiple branches, and §4.3.5.2 case 2 collects every
// <enumeration> sibling's value into one set. A repeated child of any other kind
// is a SECOND facet of that kind, which st-props-correct (§3.16.6.1) clause 4
// forbids.
func mergesRepeatedChildren(kind xsd.FacetKind) bool {
	return kind == xsd.FacetPattern || kind == xsd.FacetEnumeration
}

// buildOwnFacets translates the schema's facet children into the leaf's
// ownFacets, grouping same-kind children (pattern/enumeration carry a set of
// {value}s) into one facet in first-seen order (D2: the map is a lookup, output
// order comes from the order slice). It returns ok=false — declining the case —
// when a child names an unrecognized facet or a facet inapplicable to base
// (cos-applicable-facets §4.1.5), so the synthesized leaf never carries a facet
// that would violate ValidateLexical's facet precondition
// (mustNotBeFacetPrecondition asserts the "never").
//
// A repeated child of a kind that does NOT merge is declined for a different
// reason: the schema is st-props-correct clause 4 invalid, and the same-kind
// grouping below would fold the duplicate into one facet, hiding it from the
// clause-4 check xsd.NewSimpleType runs. The case would then be decided against a
// type the schema does not define — an agreement with the suite reached for the
// wrong reason, which the ratchet would lock in (warden's advisory on #75).
func buildOwnFacets(base string, children []facetChild) ([]xsd.Facet, bool) {
	spec, ok := typeSpecOf(base)
	if !ok {
		return nil, false
	}
	// Enumeration over QName/NOTATION compares §3.2.18 {namespace name, local name}
	// tuples, so a prefixed enum member (e.g. "foo:fo") must resolve against the
	// DECLARING SCHEMA's in-scope bindings — now carried per member on the facet
	// (issue #152, xsd.EnumerationMember). Each enumeration child brings its own
	// snapshot of those bindings (facetChild.bindings, decoded from the schema
	// document's ancestor chain), which enumerationMember threads into the member.
	// A context-free base (string/decimal/precisionDecimal/…) simply carries no
	// bindings and resolves identically to before.
	var order []xsd.FacetKind
	seen := map[xsd.FacetKind]bool{}
	values := map[xsd.FacetKind][]string{}
	var enumMembers []xsd.EnumerationMember
	for _, ch := range children {
		kind, ok := facetKindOf(ch.name)
		if !ok {
			return nil, false
		}
		if !spec.Applies(builtin.FacetName(kind.String())) {
			return nil, false
		}
		if seen[kind] && !mergesRepeatedChildren(kind) {
			return nil, false
		}
		if !seen[kind] {
			seen[kind] = true
			order = append(order, kind)
		}
		if kind == xsd.FacetEnumeration {
			enumMembers = append(enumMembers, enumerationMember(ch))
			continue
		}
		values[kind] = append(values[kind], ch.value)
	}
	facets := make([]xsd.Facet, 0, len(order))
	for _, kind := range order {
		if kind == xsd.FacetEnumeration {
			facets = append(facets, xsd.NewEnumerationFacet(enumMembers))
			continue
		}
		facets = append(facets, xsd.NewFacet(kind, values[kind], false))
	}
	return facets, true
}

// enumerationMember builds an xsd.EnumerationMember from an <enumeration> facet
// child: its lexical {value} plus the namespace context in scope where the
// element was written (§3.3.18). ch.bindings is the innermost-wins snapshot
// decodeRestriction accumulates down the schema document; its empty-prefix key ""
// is the {default namespace}, split out from the named-prefix bindings. The named
// bindings are emitted in a deterministic prefix-sorted order (STYLE D2 — the map
// is a lookup, not output, and QName resolution is order-independent, so any
// stable order serves). A context-free child (no bindings) yields a member with
// empty context, resolving identically to a nil context.
func enumerationMember(ch facetChild) xsd.EnumerationMember {
	var def *string
	prefixes := make([]string, 0, len(ch.bindings))
	for p := range ch.bindings {
		if p == "" {
			ns := ch.bindings[""]
			def = &ns
			continue
		}
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)
	var binds []xsd.NamespaceBinding
	for _, p := range prefixes {
		binds = append(binds, xsd.NewNamespaceBinding(p, ch.bindings[p]))
	}
	return xsd.NewEnumerationMember(ch.value, binds, def)
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
// out-of-cohort shape carrying zero <foo> leaves or several is honestly declined
// rather than mis-read as a single empty or last-wins tested value. The anyURI
// a*/b* fixtures that motivated this guard are now claimed earlier, by
// anyURIShapeCase's dedicated multi-leaf reader (issue #190), so they no longer
// reach here; the guard stays because it is what keeps any FUTURE out-of-cohort
// shape from being mis-read.
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
// document order (P4: token stream, no whole-document buffer). Each facet child
// captures the namespace bindings in scope where it was written — accumulated
// down the schema document's ancestor chain including the child's own xmlns
// declarations (§3.3.18) — so buildOwnFacets can resolve a QName/NOTATION
// enumeration member's prefix against the DECLARING schema's context (issue
// #152), the schema-side analogue of readQNameContexts' instance-side walk. ok is
// false when no restriction is found.
func decodeRestriction(path string) (base, attrName string, children []facetChild, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", nil, false
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the parsed result
	dec := xml.NewDecoder(bufio.NewReader(f))
	var frames []map[string]string // one innermost-wins snapshot per open element
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
			if len(frames) > 0 {
				frames = frames[:len(frames)-1]
			}
			continue
		}
		se, isStart := tok.(xml.StartElement)
		if !isStart {
			continue
		}
		frames = append(frames, childBindings(frames, se.Attr))
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
			children = append(children, facetChild{
				name:     se.Name.Local,
				value:    attrValue(se, "value"),
				bindings: frames[len(frames)-1],
			})
		}
	}
	if inRestriction {
		return base, attrName, children, true
	}
	return "", "", nil, false
}

// notationStep is one <xsd:restriction> step of a NOTATION Facets-cohort schema
// (issue #153): its base primitive (prefix stripped), the name of the enclosing
// <xsd:attribute> if any (empty for the outer named-type step, "attrTest" for the
// inner attribute step), and its constraining-facet children in document order.
type notationStep struct {
	base     string
	attrName string
	children []facetChild
}

// readNotationFacetsCase reads one NOTATION Facets-cohort instance (issue #153):
// the tested value (the <foo> element's attrTest attribute — never its text
// content, which is a fixed placeholder) and, from the schema at the instance's
// noNamespaceSchemaLocation, the two restriction steps' facet children. ok is
// false when either document does not decode to this shape, the base step does not
// restrict NOTATION, or the leaf step carries no attribute name (so no instance
// attribute names the tested value) — an honest decline, never a guess.
func readNotationFacetsCase(instancePath string) (raw string, baseChildren, leafChildren []facetChild, ctx value.Context, ok bool) {
	inst, err := decodeFacetsInstance(instancePath)
	if err != nil || inst.SchemaLoc == "" || len(inst.Foos) != 1 {
		return "", nil, nil, nil, false
	}
	foo := inst.Foos[0]
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(inst.SchemaLoc))
	baseStep, leafStep, ok := decodeNotationRestriction(schemaPath)
	if !ok || baseStep.base != "NOTATION" || leafStep.attrName == "" {
		return "", nil, nil, nil, false
	}
	// The instance's root-level namespace bindings, threaded to ValidateLexical so
	// NOTATION's context-dependent Parse (§3.3.19, "as given for QName", §3.3.18) can
	// resolve the tested value's prefix — the same root-only snapshot the QName facet
	// cohort proved sufficient (childBindings is a lookup-only map, never output).
	ctx = nsContext{bindings: childBindings(nil, inst.Attrs)}
	v, found := foo.attr(leafStep.attrName)
	if !found {
		return "", nil, nil, nil, false
	}
	return v, baseStep.children, leafStep.children, ctx, true
}

// decodeNotationRestriction streams a NOTATION Facets-cohort schema and returns
// its two <xsd:restriction> steps: the outer named-type step (restricting
// xsd:NOTATION with the jpeg/mpeg/g enumerations) and the inner attribute step
// (restricting the named type with the one tested facet). Modeled on
// decodeRestriction's single forward streaming pass (P4: token stream, no
// whole-document buffer), but instead of returning at the FIRST </restriction>
// close it collects EVERY restriction step in document order — the shape
// guarantees exactly two in fixed order, so no local-name symbol table is needed
// (grounding #153). Each facet child captures the namespace bindings in scope
// where it was written (§3.3.18), so buildOwnFacets can resolve a NOTATION
// enumeration member's prefix against the declaring schema's context, exactly as
// decodeRestriction does. ok is false unless exactly two steps are found with the
// second carrying a non-empty attrName (the leaf) — an honest decline.
func decodeNotationRestriction(path string) (baseStep, leafStep notationStep, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return notationStep{}, notationStep{}, false
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the parsed result
	dec := xml.NewDecoder(bufio.NewReader(f))
	var frames []map[string]string // one innermost-wins snapshot per open element
	var steps []notationStep
	inRestriction := false
	lastAttr := ""
	var cur notationStep
	for {
		tok, terr := dec.Token()
		if terr != nil {
			break
		}
		if end, isEnd := tok.(xml.EndElement); isEnd {
			if inRestriction && end.Name.Local == "restriction" && end.Name.Space == xsd.XMLSchemaNS {
				steps = append(steps, cur)
				inRestriction = false
				lastAttr = "" // consumed: never leak this attribute name to a later step
			}
			if len(frames) > 0 {
				frames = frames[:len(frames)-1]
			}
			continue
		}
		se, isStart := tok.(xml.StartElement)
		if !isStart {
			continue
		}
		frames = append(frames, childBindings(frames, se.Attr))
		if inRestriction {
			if se.Name.Space == xsd.XMLSchemaNS {
				cur.children = append(cur.children, facetChild{
					name:     se.Name.Local,
					value:    attrValue(se, "value"),
					bindings: frames[len(frames)-1],
				})
			}
			continue
		}
		if se.Name.Local == "attribute" && se.Name.Space == xsd.XMLSchemaNS {
			lastAttr = attrValue(se, "name")
		}
		if se.Name.Local == "restriction" && se.Name.Space == xsd.XMLSchemaNS {
			inRestriction = true
			cur = notationStep{base: localName(attrValue(se, "base")), attrName: lastAttr}
		}
	}
	if len(steps) != 2 || steps[1].attrName == "" {
		return notationStep{}, notationStep{}, false
	}
	return steps[0], steps[1], true
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

// d34Element pairs one instance child of an IBM D3_3_4 <root> (issue #162) with
// the KEY of the simple type it is declared as (resolved from the schema's
// sequence via the child's type= attribute, or via the global element its ref=
// names) and its text value, so execD34Case validates each child against the leaf
// built for its type.
type d34Element struct {
	typeKey string
	value   string
}

// d34TypeDecl is one simple-type declaration a D3_3_4 schema offers, paired with
// the key elements bind it by: a top-level <simpleType>'s own name, or — for the
// anonymous inline <simpleType> of a global <element> — d34InlineKey's synthesized
// key. Declarations are carried as a SLICE in document order, not a map: the
// build pass below iterates them repeatedly, and a map would make which type is
// attempted first (and so which failure is reported) nondeterministic (STYLE D2).
type d34TypeDecl struct {
	key  string
	decl d34SimpleType
}

// d34InlineKey keys the anonymous inline <simpleType> of the global element named
// name. The "element:" prefix contains a colon, which no NCName type name can, so
// an inline key never collides with a top-level named simpleType (the
// anyURIBuiltinKey trick).
func d34InlineKey(name string) string { return "element:" + name }

// d34Declared reports whether decls carries a declaration under key. It is a
// linear scan over the schema's handful of type declarations, which is what keeps
// the declarations one ordered slice rather than a slice plus a parallel index
// (STYLE D3).
func d34Declared(decls []d34TypeDecl, key string) bool {
	for _, d := range decls {
		if d.key == key {
			return true
		}
	}
	return false
}

// readD34Case reads one IBM D3_3_4 precisionDecimal cohort instance (issue #162)
// and returns every simple type the schema DECLARES (decls, in document order)
// plus every instance child paired with the key of its resolved type, in document
// order (elems). The schema is resolved from the instance's own
// xsi:schemaLocation (a whitespace-separated namespace/location pair — NOT the
// Saxon cohort's filename-derived path, since d3_3_4ii01a.xml shares
// d3_3_4ii01.xsd), located relative to the instance's directory.
//
// The reader answers only "what does the schema declare, and which declaration
// governs each instance child" — never "can this shape be decided", which is
// execD34Case's buildD34Types pass. Splitting them is what lets the list (v16) and
// union (v17) varieties enter through the same door as the atomic one (issue
// #223): the reader admits any declared simple type, and a shape the value
// pipeline cannot back simply fails to build.
//
// ok is false — declining the WHOLE case, never partially deciding — when: the
// instance or schema cannot be read; the schema has no <element name="root">
// carrying a <sequence> (inline, or through the named complexType its type= names);
// that sequence is empty; a child binds neither a type= nor a resolvable ref=; a
// child's type is not among the schema's simple-type declarations (v15, whose
// sequence children are typed by a COMPLEX type, and v19–v22, whose named types
// are still declared and so still reach the build pass); or an instance child's
// local name is not bound by the sequence (an unexpected element that could carry
// an undecidable literal).
func readD34Case(instancePath string) (decls []d34TypeDecl, elems []d34Element, ok bool) {
	inst, err := decodeD34Instance(instancePath)
	if err != nil {
		return nil, nil, false
	}
	loc, ok := qualifiedSchemaLocation(inst.XMLName.Space, inst.SchemaLoc)
	if !ok {
		return nil, nil, false
	}
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(loc))
	schema, err := decodeD34Schema(schemaPath)
	if err != nil {
		return nil, nil, false
	}
	decls = d34TypeDecls(schema)
	seq, ok := d34RootSequence(schema)
	if !ok {
		return nil, nil, false
	}
	elemTypes, ok := d34SequenceTypes(schema, seq, decls)
	if !ok {
		return nil, nil, false
	}
	if len(inst.Children) == 0 {
		return nil, nil, false
	}
	for _, ch := range inst.Children {
		key, bound := elemTypes[ch.XMLName.Local]
		if !bound {
			return nil, nil, false
		}
		elems = append(elems, d34Element{typeKey: key, value: ch.Text})
	}
	return decls, elems, true
}

// qualifiedSchemaLocation extracts the schema location for the root element's
// namespace from an xsi:schemaLocation value (whitespace-separated
// namespace/location pairs, §2.6.3). It returns the location paired with rootNS,
// or — since both cohorts that use this form, IBM D3_3_4 (issue #162) and the
// anyURI a* fixtures (issue #190), always carry exactly one pair, for the
// instance's own namespace — the first pair's location as a fallback. ok is false
// when no pair is present.
func qualifiedSchemaLocation(rootNS, schemaLoc string) (loc string, ok bool) {
	fields := strings.Fields(schemaLoc)
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i] == rootNS {
			return fields[i+1], true
		}
	}
	if len(fields) >= 2 {
		return fields[1], true
	}
	return "", false
}

// d34Instance mirrors an IBM D3_3_4 instance: a namespace-qualified <root> whose
// xsi:schemaLocation names the schema and whose direct child elements each carry
// one tested literal in their text content. XMLName captures the root's namespace
// (to pick the matching schemaLocation pair); Children collects every direct child
// element (,any) in document order, keyed by local name to the schema's sequence.
type d34Instance struct {
	XMLName   xml.Name
	SchemaLoc string     `xml:"http://www.w3.org/2001/XMLSchema-instance schemaLocation,attr"`
	Children  []d34Child `xml:",any"`
}

// d34Child is one direct child of the D3_3_4 <root>: its qualified name (its local
// part binds to the schema sequence's element name) and its text value.
type d34Child struct {
	XMLName xml.Name
	Text    string `xml:",chardata"`
}

func decodeD34Instance(path string) (d34Instance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return d34Instance{}, err
	}
	var inst d34Instance
	if err := xml.Unmarshal(data, &inst); err != nil {
		return d34Instance{}, err
	}
	return inst, nil
}

// d34Schema mirrors a D3_3_4 schema: its top-level simpleTypes and complexTypes
// plus its top-level element declarations, one of which is <element name="root">
// carrying — inline, or through the named complexType its type= names — the
// sequence that binds child element names to types.
type d34Schema struct {
	SimpleTypes  []d34SimpleType  `xml:"simpleType"`
	ComplexTypes []d34ComplexType `xml:"complexType"`
	Elements     []d34ElementDecl `xml:"element"`
}

// d34SimpleType mirrors one <simpleType> — top-level and named, or anonymous
// inside a <list>, a <union> or an <element>. At most one of the three variety
// children is present, and buildD34Type decides which shapes it can back; a
// declaration with none (or with an unsupported one) simply never builds. The
// three are pointers so "absent" is distinguishable from "present but empty" (a
// facet-free <restriction>, an itemType-only <list>, a memberTypes-only <union>).
// The type is recursive through its <list>/<union> children, which is exactly the
// nesting v16 (<list><simpleType>) and v17 (<union><simpleType>) use.
type d34SimpleType struct {
	Name        string          `xml:"name,attr"`
	Restriction *d34Restriction `xml:"restriction"`
	List        *d34List        `xml:"list"`
	Union       *d34Union       `xml:"union"`
}

// d34Restriction is a <restriction base=..> with its facet children, captured by
// element name (,any) exactly as the pre-#223 reader did.
type d34Restriction struct {
	Base   string `xml:"base,attr"`
	Facets []struct {
		XMLName xml.Name
		Value   string `xml:"value,attr"`
	} `xml:",any"`
}

// d34List is a <list>: its {item type definition} named by itemType= or given as
// an anonymous inline <simpleType> child (§3.16.2.3 map.std.list). An <annotation>
// child, which v16's decListC carries, is deliberately not captured.
type d34List struct {
	ItemType string          `xml:"itemType,attr"`
	Item     []d34SimpleType `xml:"simpleType"`
}

// d34Union is a <union>: its {member type definitions} named by memberTypes= and/or
// given as anonymous inline <simpleType> children. Order is load-bearing —
// map.std.union clause 1 takes "(a) resolved to by the items in the actual value of
// the memberTypes attribute, if any, and (b) corresponding to the <simpleType>s
// among the children, if any, in order" — so buildD34Type concatenates them in
// exactly that order, which is the order dv_union scans (§4.1.4 cl.2.3).
type d34Union struct {
	MemberTypes string          `xml:"memberTypes,attr"`
	Members     []d34SimpleType `xml:"simpleType"`
}

// d34ComplexType is a <complexType> reduced to what this cohort needs: its name
// (absent when inline on an element) and its <sequence> of child element
// declarations.
type d34ComplexType struct {
	Name     string `xml:"name,attr"`
	Sequence struct {
		Elements []d34SeqElement `xml:"element"`
	} `xml:"sequence"`
}

// d34SeqElement is one <element> inside a sequence: a LOCAL declaration binding a
// name to a type=, or a ref= to a global element declaration (v16's eldecListC,
// v18's elEnumerationA/B).
type d34SeqElement struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
	Ref  string `xml:"ref,attr"`
}

// d34ElementDecl is a top-level <element>: <root> (whose complexType, inline or
// named by type=, carries the tested sequence) or a global element a sequence
// reaches by ref=, whose own type is a type= reference or an anonymous inline
// <simpleType>.
type d34ElementDecl struct {
	Name        string          `xml:"name,attr"`
	Type        string          `xml:"type,attr"`
	SimpleType  *d34SimpleType  `xml:"simpleType"`
	ComplexType *d34ComplexType `xml:"complexType"`
}

func decodeD34Schema(path string) (d34Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return d34Schema{}, err
	}
	var s d34Schema
	if err := xml.Unmarshal(data, &s); err != nil {
		return d34Schema{}, err
	}
	return s, nil
}

// d34TypeDecls collects every simple type the schema declares, in document order:
// each top-level named <simpleType> under its own name, then each global
// <element>'s anonymous inline <simpleType> under d34InlineKey. It filters
// NOTHING on variety or base — that judgement belongs to buildD34Types, which is
// what lets the list and union varieties (v16/v17) and the multi-step restriction
// chains (v18, v19–v22) reach the value pipeline while a shape it cannot back — a
// base, itemType= or memberTypes= entry naming neither a declaration of this schema
// nor a seeded, strict-governed builtin — still declines by failing to build.
func d34TypeDecls(s d34Schema) []d34TypeDecl {
	decls := make([]d34TypeDecl, 0, len(s.SimpleTypes)+len(s.Elements))
	for _, st := range s.SimpleTypes {
		if st.Name == "" {
			continue
		}
		decls = append(decls, d34TypeDecl{key: st.Name, decl: st})
	}
	for _, el := range s.Elements {
		if el.Name == "" || el.SimpleType == nil {
			continue
		}
		decls = append(decls, d34TypeDecl{key: d34InlineKey(el.Name), decl: *el.SimpleType})
	}
	return decls
}

// d34RootSequence returns the child element declarations of <element name="root">:
// from its INLINE <complexType><sequence>, or — v16's shape — from the named
// complexType its type= attribute references. ok is false when there is no root
// element, when its type= names no declared complexType, or when it carries
// neither (v15's root type= names a complexType whose content is a simpleContent
// extension, so its sequence decodes empty and the caller declines on the
// empty-sequence check).
func d34RootSequence(s d34Schema) ([]d34SeqElement, bool) {
	for _, el := range s.Elements {
		if el.Name != "root" {
			continue
		}
		if el.ComplexType != nil {
			return el.ComplexType.Sequence.Elements, true
		}
		if el.Type == "" {
			return nil, false
		}
		for _, ct := range s.ComplexTypes {
			if ct.Name == localName(el.Type) {
				return ct.Sequence.Elements, true
			}
		}
		return nil, false
	}
	return nil, false
}

// d34SequenceTypes maps each instance element name the root sequence binds to the
// key of the simple type declared for it. ok is false — declining the case — when
// the sequence is empty (v15, whose root complexType carries no sequence at all),
// or when any single child fails to resolve, so the case is never partially
// decided.
func d34SequenceTypes(s d34Schema, seq []d34SeqElement, decls []d34TypeDecl) (map[string]string, bool) {
	if len(seq) == 0 {
		return nil, false
	}
	elemTypes := make(map[string]string, len(seq))
	for _, child := range seq {
		name, key, ok := d34ChildType(s, child, decls)
		if !ok {
			return nil, false
		}
		elemTypes[name] = key
	}
	return elemTypes, true
}

// d34ChildType resolves ONE root-sequence child to the instance element name it
// binds and the key of the simple type declared for it. A ref= child defers to
// d34RefType (the global element it names carries the type); a local child must
// carry both a name= and a type= naming a DECLARED simple type — v15's children are
// typed by a complexType, which is no such declaration, so that shape declines
// here.
func d34ChildType(s d34Schema, child d34SeqElement, decls []d34TypeDecl) (name, key string, ok bool) {
	if child.Ref != "" {
		return d34RefType(s, localName(child.Ref), decls)
	}
	if child.Name == "" || child.Type == "" {
		return "", "", false
	}
	tn := localName(child.Type)
	return child.Name, tn, d34Declared(decls, tn)
}

// d34RefType resolves an <element ref="..."> to the referenced GLOBAL element's
// name — which is the name the instance child carries — and the key of the simple
// type governing it: the anonymous inline <simpleType> the element declares
// (v16's eldecListC, v18's elEnumerationA/B) or the named type its type= points
// at. ok is false when no global element bears that name or it declares neither.
func d34RefType(s d34Schema, ref string, decls []d34TypeDecl) (name, key string, ok bool) {
	for _, el := range s.Elements {
		if el.Name != ref {
			continue
		}
		if el.SimpleType != nil {
			inline := d34InlineKey(el.Name)
			return el.Name, inline, d34Declared(decls, inline)
		}
		if el.Type == "" {
			return "", "", false
		}
		tn := localName(el.Type)
		return el.Name, tn, d34Declared(decls, tn)
	}
	return "", "", false
}

// anyURIBuiltinKey keys the bare xsd:anyURI builtin in the anyURI a*/b* cohort's
// type index (issue #190): anyURI_b006's <foo> children are typed xsd:anyURI
// DIRECTLY — facet-free — while its <bar> siblings carry the enumeration-restricted
// named type, so the two must not be conflated. The key contains a colon, which no
// NCName type name can, so it never collides with a schema's own named type.
const anyURIBuiltinKey = "xsd:anyURI"

// anyURILeaf is one tested value in an anyURI a*/b* instance: the key of the
// simple type governing it (a named single-step anyURI restriction, or
// anyURIBuiltinKey) and its RAW lexical, un-normalized — ValidateLexical's
// whiteSpace stage applies anyURI's fixed collapse.
type anyURILeaf struct {
	typeName string
	value    string
}

// anyURINode is one element occurrence in an anyURI a*/b* instance: its LOCAL
// name, its DIRECT character data (descendant text belongs to the descendant's own
// node), and its unqualified attributes in document order. Matching on the local
// name alone is sound for this cohort exactly as it is for readD34Case: each
// fixture's schema has one target namespace (or none) and declares each name once,
// a fact indexAnyURIElements re-checks rather than assumes.
type anyURINode struct {
	name  string
	text  string
	attrs []xml.Attr
}

// anyURIBinding is a resolved element (or named complexType) declaration: the key
// of the simple type governing its TEXT content — "" when the content is
// element-only, like the <root> wrapper — plus the simple-type key of every
// attribute it declares, keyed by attribute name. attrTypes is a lookup, never
// ranged into output (STYLE D2).
type anyURIBinding struct {
	textType  string
	attrTypes map[string]string
}

// anyURIIndex is one cohort schema decoded and indexed for leaf resolution: every
// named simpleType that single-step restricts xsd:anyURI (by local name, carrying
// its facet children), every named complexType whose simpleContent resolves to
// such a type, and every element declaration in the document — top-level or local
// — by local name. All three are lookups, never ranged into output (STYLE D2).
type anyURIIndex struct {
	simpleTypes  map[string][]facetChild
	complexTypes map[string]anyURIBinding
	elements     map[string]anyURIElemDecl
}

// readAnyURIShapeCase reads one anyURI a*/b* cohort instance (issue #190) and
// returns the facet children of every named anyURI restriction the schema declares
// (typeFacets, keyed by local name — execAnyURIShapeCase synthesizes one leaf per
// type in use) plus every tested value in document order paired with the key of
// the type governing it (leaves): each element's character content, and each
// unqualified attribute's value.
//
// ok is false — declining the WHOLE case, never partially deciding it — when: the
// instance or its schema cannot be read; the instance root declares neither
// xsi:noNamespaceSchemaLocation (the b* form) nor an xsi:schemaLocation pair for
// its own namespace (the a* form); two element declarations share a name but
// reference different types; an element occurrence is not declared by the schema;
// an element's or attribute's type is not a single-step anyURI restriction nor the
// bare builtin (a list/union variety or a multi-step chain never enters the index);
// an element with element-only content carries real character data; an attribute
// is not declared by its element's type; or no tested value is found at all.
func readAnyURIShapeCase(instancePath string) (typeFacets map[string][]facetChild, leaves []anyURILeaf, ok bool) {
	schemaLoc, nodes, ok := decodeAnyURIInstance(instancePath)
	if !ok {
		return nil, nil, false
	}
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(schemaLoc))
	schema, err := decodeAnyURISchema(schemaPath)
	if err != nil {
		return nil, nil, false
	}
	simpleTypes := indexAnyURISimpleTypes(schema)
	elements, ok := indexAnyURIElements(schema)
	if !ok {
		return nil, nil, false
	}
	idx := anyURIIndex{
		simpleTypes:  simpleTypes,
		complexTypes: indexAnyURIComplexTypes(schema, simpleTypes),
		elements:     elements,
	}
	for _, n := range nodes {
		decl, declared := idx.elements[n.name]
		if !declared {
			return nil, nil, false
		}
		bind, resolved := bindAnyURIElem(idx, decl)
		if !resolved {
			return nil, nil, false
		}
		if bind.textType == "" && strings.TrimSpace(n.text) != "" {
			// Element-only content carrying real character data is a shape this reader
			// does not model: decline rather than drop a value that may decide the case.
			// The cohort's <root> wrappers carry only inter-element whitespace, which is
			// permitted by their element-only content type and is not a tested value.
			return nil, nil, false
		}
		if bind.textType != "" {
			leaves = append(leaves, anyURILeaf{typeName: bind.textType, value: n.text})
		}
		for _, a := range n.attrs {
			key, attrDeclared := bind.attrTypes[a.Name.Local]
			if !attrDeclared {
				return nil, nil, false
			}
			leaves = append(leaves, anyURILeaf{typeName: key, value: a.Value})
		}
	}
	if len(leaves) == 0 {
		return nil, nil, false
	}
	return simpleTypes, leaves, true
}

// decodeAnyURIInstance streams an anyURI a*/b* instance and returns the schema
// location its root declares plus every element occurrence in document order (P4:
// token stream, no whole-document tree). Character data is charged to the
// innermost open element, so a wrapper's inter-element whitespace never merges
// into a leaf's tested value. ok is false when the file cannot be opened, is not
// well-formed, declares no usable schema location, or carries no element at all.
func decodeAnyURIInstance(path string) (schemaLoc string, nodes []anyURINode, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, false
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the parsed result
	dec := xml.NewDecoder(bufio.NewReader(f))
	var open []int // indexes into nodes, innermost last: one per currently-open element
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", nil, false
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if len(nodes) == 0 {
				loc, declared := anyURISchemaLocation(t)
				if !declared {
					return "", nil, false
				}
				schemaLoc = loc
			}
			nodes = append(nodes, anyURINode{name: t.Name.Local, attrs: testedAttrs(t)})
			open = append(open, len(nodes)-1)
		case xml.EndElement:
			if len(open) == 0 {
				return "", nil, false
			}
			open = open[:len(open)-1]
		case xml.CharData:
			if len(open) > 0 {
				nodes[open[len(open)-1]].text += string(t)
			}
		}
	}
	if len(nodes) == 0 || len(open) != 0 {
		return "", nil, false
	}
	return schemaLoc, nodes, true
}

// anyURISchemaLocation extracts the schema location an instance root declares. The
// b* fixtures use xsi:noNamespaceSchemaLocation (their schema has no target
// namespace); the a* fixtures use the namespace-qualified xsi:schemaLocation form
// (§2.6.3), whose pair for the root's OWN namespace names the schema —
// qualifiedSchemaLocation resolves that form. ok is false when neither is present.
func anyURISchemaLocation(root xml.StartElement) (loc string, ok bool) {
	noNamespace, qualified := "", ""
	for _, a := range root.Attr {
		if a.Name.Space != xsd.XMLSchemaInstanceNS {
			continue
		}
		if a.Name.Local == "noNamespaceSchemaLocation" {
			noNamespace = a.Value
		}
		if a.Name.Local == "schemaLocation" {
			qualified = a.Value
		}
	}
	if noNamespace != "" {
		return noNamespace, true
	}
	if qualified != "" {
		return qualifiedSchemaLocation(root.Name.Space, qualified)
	}
	return "", false
}

// testedAttrs returns se's UNQUALIFIED attributes — the ones an
// attributeFormDefault="unqualified" schema declares and this cohort tests. The
// xmlns declarations (default and prefixed) and the namespace-qualified xsi:*
// instance attributes carry no tested value and are dropped.
func testedAttrs(se xml.StartElement) []xml.Attr {
	var attrs []xml.Attr
	for _, a := range se.Attr {
		if a.Name.Space != "" || a.Name.Local == "xmlns" {
			continue
		}
		attrs = append(attrs, a)
	}
	return attrs
}

// anyURISchema mirrors an anyURI a*/b* cohort schema: its top-level element
// declarations, its named simpleTypes (each a candidate single-step anyURI
// restriction) and its named complexTypes (anyURI_b002's simpleContent extension,
// which adds an attribute to a restricted anyURI content type). Everything else
// these schemas carry — <include>/<import>/<redefine> with deliberately
// unresolvable locations, <notation>, <annotation> — is not load-bearing for any
// instance verdict and is deliberately not decoded (STYLE D4).
type anyURISchema struct {
	Elements     []anyURIElemDecl    `xml:"element"`
	SimpleTypes  []anyURISimpleType  `xml:"simpleType"`
	ComplexTypes []anyURIComplexType `xml:"complexType"`
}

// anyURIElemDecl is one element declaration, top-level or local: its name, the
// type it references, and — when it declares an INLINE complexType instead (the
// <root> wrapper) — that complexType's attribute declarations (a004's att) and its
// local element children under a <sequence> (a004/b001/b002/b004/b005) or a
// <choice> (b006). The nesting is recursive because a local child may declare an
// inline complexType of its own; no cohort fixture does, but walking it costs
// nothing and cannot mis-read one. A <element ref="…"/> child carries no name and
// contributes nothing: the top-level declaration it references is decoded here in
// its own right.
type anyURIElemDecl struct {
	Name        string `xml:"name,attr"`
	Type        string `xml:"type,attr"`
	ComplexType struct {
		Attributes []anyURIAttrDecl `xml:"attribute"`
		Sequence   []anyURIElemDecl `xml:"sequence>element"`
		Choice     []anyURIElemDecl `xml:"choice>element"`
	} `xml:"complexType"`
}

// anyURIAttrDecl is one <xsd:attribute name= type=> declaration: the name an
// instance attribute must match and the simple type its value is tested against.
type anyURIAttrDecl struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

// anyURISimpleType is a named simpleType and its single-step restriction: the base
// it restricts and that restriction's facet children in document order.
type anyURISimpleType struct {
	Name        string `xml:"name,attr"`
	Restriction struct {
		Base   string `xml:"base,attr"`
		Facets []struct {
			XMLName xml.Name
			Value   string `xml:"value,attr"`
		} `xml:",any"`
	} `xml:"restriction"`
}

// anyURIComplexType is a named complexType with simpleContent (anyURI_b002's
// "ct"): the extension's base names the simple type governing an element's TEXT,
// and its attribute declarations name the types of the attributes it adds.
type anyURIComplexType struct {
	Name      string `xml:"name,attr"`
	Extension struct {
		Base       string           `xml:"base,attr"`
		Attributes []anyURIAttrDecl `xml:"attribute"`
	} `xml:"simpleContent>extension"`
}

func decodeAnyURISchema(path string) (anyURISchema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return anyURISchema{}, err
	}
	var s anyURISchema
	if err := xml.Unmarshal(data, &s); err != nil {
		return anyURISchema{}, err
	}
	return s, nil
}

// indexAnyURISimpleTypes indexes every named simpleType that is a DIRECT
// single-step <restriction base="(prefix:)?anyURI"> by its local name, carrying its
// facet children. A named type whose restriction base's local name is anything else
// — another named type (a multi-step chain) or a list/union variety, whose
// simpleType carries no such restriction so Base is empty — is NOT indexed, which
// is how those shapes fall out of scope with no special-casing: every leaf that
// would have used one declines. The facet children carry no namespace bindings
// (facetChild.bindings nil): anyURI maps context-free (§3.3.17.2), so an
// enumeration member needs no declaring-schema context to resolve.
func indexAnyURISimpleTypes(s anyURISchema) map[string][]facetChild {
	index := make(map[string][]facetChild, len(s.SimpleTypes))
	for _, st := range s.SimpleTypes {
		if st.Name == "" || localName(st.Restriction.Base) != "anyURI" {
			continue
		}
		var children []facetChild
		for _, f := range st.Restriction.Facets {
			children = append(children, facetChild{name: f.XMLName.Local, value: f.Value})
		}
		index[st.Name] = children
	}
	return index
}

// indexAnyURIComplexTypes indexes every named complexType whose simpleContent
// extension resolves end to end: its base names an indexed anyURI restriction (or
// the bare builtin) and each attribute it adds does too. A complexType that does
// not — element-only content, a non-anyURI content type, an unresolvable attribute
// type — is NOT indexed, so an element declared with it declines rather than being
// read against a guessed type.
func indexAnyURIComplexTypes(s anyURISchema, simpleTypes map[string][]facetChild) map[string]anyURIBinding {
	index := make(map[string]anyURIBinding, len(s.ComplexTypes))
	for _, ct := range s.ComplexTypes {
		if ct.Name == "" {
			continue
		}
		text, resolved := anyURISimpleKey(simpleTypes, ct.Extension.Base)
		if !resolved {
			continue
		}
		attrs, ok := anyURIAttrTypes(simpleTypes, ct.Extension.Attributes)
		if !ok {
			continue
		}
		index[ct.Name] = anyURIBinding{textType: text, attrTypes: attrs}
	}
	return index
}

// indexAnyURIElements indexes every element declaration in the schema document —
// top-level and the local children of an inline complexType — by its local name.
// ok is false when two declarations share a name but name DIFFERENT types in
// their type= attribute, which would make an instance occurrence ambiguous under
// readAnyURIShapeCase's local-name matching; no cohort fixture does.
//
// The check compares type= ATTRIBUTE STRINGS, not fully resolved shapes, so one
// conflict escapes it: two same-named declarations that BOTH carry an inline
// complexType have Type == "" and so compare equal even when their inline shapes
// differ, and the second silently overwrites the first — last-wins, unflagged. No
// cohort fixture does that either, but anyURIShapeCase is a FAMILY regexp
// (anyURI_[ab][0-9]+\.xml), not an enumeration of the eight files, so any future
// fixture dropped into msData/datatypes/Facets/anyURI/ joins the cohort with no
// review gate and could reach it. Comparing decoded inline shapes is the fix when
// one does; until then this states the invariant the code actually holds rather
// than paying for a structural equality nothing exercises (issue #290).
func indexAnyURIElements(s anyURISchema) (map[string]anyURIElemDecl, bool) {
	index := map[string]anyURIElemDecl{}
	for _, el := range s.Elements {
		if !collectAnyURIElems(index, el) {
			return nil, false
		}
	}
	return index, true
}

// collectAnyURIElems adds el to index under its name (a ref= child, which declares
// none, contributes only its subtree) and recurses into the local element children
// of its inline complexType. It reports false on the conflicting-name condition
// indexAnyURIElements documents — a prior.Type != el.Type comparison of type=
// strings, which by construction cannot see the inline-complexType conflict that
// comment names.
func collectAnyURIElems(index map[string]anyURIElemDecl, el anyURIElemDecl) bool {
	if el.Name != "" {
		prior, seen := index[el.Name]
		if seen && prior.Type != el.Type {
			return false
		}
		index[el.Name] = el
	}
	for _, child := range el.ComplexType.Sequence {
		if !collectAnyURIElems(index, child) {
			return false
		}
	}
	for _, child := range el.ComplexType.Choice {
		if !collectAnyURIElems(index, child) {
			return false
		}
	}
	return true
}

// bindAnyURIElem resolves one element declaration to its tested-value binding. A
// declaration with type= names either a simple type (its text is the tested value)
// or a simpleContent complexType (text plus the attributes it adds); a declaration
// carrying an INLINE complexType instead (the <root> wrapper) has element-only
// content, so its text is no tested value and only its own attribute declarations
// are (a004's att). ok is false when the referenced type is not indexed, declining
// the case rather than guessing a base.
func bindAnyURIElem(idx anyURIIndex, decl anyURIElemDecl) (anyURIBinding, bool) {
	if decl.Type == "" {
		attrs, ok := anyURIAttrTypes(idx.simpleTypes, decl.ComplexType.Attributes)
		if !ok {
			return anyURIBinding{}, false
		}
		return anyURIBinding{attrTypes: attrs}, true
	}
	if ct, isComplex := idx.complexTypes[localName(decl.Type)]; isComplex {
		return ct, true
	}
	text, resolved := anyURISimpleKey(idx.simpleTypes, decl.Type)
	if !resolved {
		return anyURIBinding{}, false
	}
	return anyURIBinding{textType: text}, true
}

// anyURISimpleKey resolves a type reference to the key of the simple type it names:
// an indexed named anyURI restriction, or the bare xsd:anyURI builtin
// (anyURIBuiltinKey, anyURI_b006's <foo>). Only the local part is matched — every
// cohort schema binds its xsd: prefix to the XML Schema namespace (verified) — and
// the schema's OWN named types are consulted first, so a user type sharing the
// local name anyURI could not be mistaken for the builtin. ok is false for any
// other reference (a non-anyURI base, a list/union variety, a multi-step chain),
// which declines every leaf that would have used it.
func anyURISimpleKey(simpleTypes map[string][]facetChild, ref string) (string, bool) {
	local := localName(ref)
	if _, indexed := simpleTypes[local]; indexed {
		return local, true
	}
	if local == "anyURI" {
		return anyURIBuiltinKey, true
	}
	return "", false
}

// anyURIAttrTypes maps each declared attribute's name to the key of the simple type
// its value is tested against. ok is false when any declaration is nameless or its
// type is not resolvable (anyURISimpleKey), which declines the owning type.
func anyURIAttrTypes(simpleTypes map[string][]facetChild, decls []anyURIAttrDecl) (map[string]string, bool) {
	if len(decls) == 0 {
		return nil, true
	}
	attrs := make(map[string]string, len(decls))
	for _, a := range decls {
		key, resolved := anyURISimpleKey(simpleTypes, a.Type)
		if a.Name == "" || !resolved {
			return nil, false
		}
		attrs[a.Name] = key
	}
	return attrs, true
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
