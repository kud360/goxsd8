package conformance

import (
	"bufio"
	"encoding/xml"
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
// # Still deferred
//
// Facets over the remaining primitive dirs not yet claimed here (QName,
// NOTATION), xsd:boolean facets (no Facets dir
// exists for it), the plural list-typed dirs (IDREFS, NMTOKENS), the NIST corpus,
// and list/union varieties remain out of scope until their backends land.
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

// datatypesCase matches an instance case in the lexical cohort.
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
var datatypesCase = regexp.MustCompile(`msData/datatypes/(boolean|decimal|string|float|double|anyURI|hexBinary|base64Binary|duration|dateTime|dateTimeStamp|time|date|gYearMonth|gYear|gMonthDay|gDay|gMonth)[0-9]+\.xml$`)

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
// special-casing is needed here. No
// new backend mapping is introduced in any case. ENTITY has no Facets cases in the
// current W3C checkout (no msData/datatypes/Facets/ENTITY dir); it is listed for
// spec parity and mechanism reuse (a zero-case regex alternative is harmless), so
// a future suite update carrying such cases is decided with no further code change.
// The list feeds both the
// directory and the filename-prefix alternation of facetsCase.
const facetsBaseTypes = `string|normalizedString|token|language|Name|NCName|NMTOKEN|` +
	`ID|IDREF|ENTITY|` +
	`anyURI|hexBinary|base64Binary|` +
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

// selectsDatatypes claims the instance cases of both cohorts. It is a cheap path
// predicate; the executor does the real document reading.
func selectsDatatypes(c caseSpec) bool {
	if c.kind != kindInstance {
		return false
	}
	doc := filepath.ToSlash(c.doc)
	return datatypesCase.MatchString(doc) || facetsCase.MatchString(doc)
}

// newDatatypesExec builds the lane's executor: it composes builtin/strict with
// a trivial fallback so builtin.Seed's all-primitives precondition is met,
// Seeds the builtins once (the M3 composition step), and captures the composed
// backend plus the seeded symbol table in the returned closure.
func newDatatypesExec() executor {
	// strict.New() now maps all 20 builtin primitives (decimal/precisionDecimal/
	// boolean/string/anyURI/float/double/hexBinary/base64Binary/duration/dateTime
	// plus the six seven-property siblings time/date/gYearMonth/gYear/gMonthDay/
	// gDay/gMonth and QName/NOTATION); Seed requires all 20, so the fallback — once
	// needed to cover precisionDecimal — is now fully redundant, retained only as a
	// defensive floor. strict wins where it maps (Override
	// yields partial first), so those fallback mappings are never actually
	// exercised: the lane now claims boolean/decimal/string/float/double/anyURI/
	// hexBinary/base64Binary/duration/dateTime/time/date/gYearMonth/gYear/
	// gMonthDay/gDay/gMonth
	// (lexical cohort) and string/decimal/float/double plus the integer and
	// derived-string (normalizedString/token, #85; language/Name/NCName/NMTOKEN,
	// #106; ID/IDREF/ENTITY, #116) families, anyURI/hexBinary/base64Binary (#124)
	// and the temporal primitives (#123) (facet cohort) cases
	// (float/double added in #80, anyURI in #82, hexBinary/base64Binary in #83,
	// duration in #84, dateTime in #103, the seven-property siblings in #109),
	// every one of which resolves (directly or via a base ancestor) to a strict
	// mapping — the no-op fallback still never runs for a claimed case.
	strictBackend := strict.New()
	backend := value.Override(fallbackPrimitives{}, strictBackend)

	// Seed proves the composed backend satisfies the precondition and yields
	// the builtin components; the executor confirms a claimed case's type is a
	// seeded builtin before validating it. The composed backend is complete by
	// construction (every primitive covered by the fallback, guarded by
	// TestDatatypesBackendSeeds), so a Seed error here is a programming error,
	// not a runtime condition — panic rather than drop it.
	types, err := builtin.Seed(backend)
	if err != nil {
		panic("conformance: datatypes lane backend must Seed by construction: " + err.Error())
	}
	sym := make(map[xsd.QName]*xsd.SimpleType, len(types))
	for _, t := range types {
		sym[t.Name()] = t
	}

	return func(c caseSpec) Status {
		if facetsCase.MatchString(filepath.ToSlash(c.doc)) {
			return execFacetsCase(backend, strictBackend, sym, c)
		}
		return execLexicalCase(backend, sym, c)
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
	observedValid := true
	for _, v := range values {
		if !parseOK(m, prim, v) {
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
		if !parseOK(m, lit.prim, lit.value) {
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
func execFacetsCase(backend, strictBackend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	raw, base, children, ok := readFacetsCase(c.doc)
	if !ok {
		return Fail()
	}
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: base}
	builtinType, seeded := sym[qn]
	if !seeded {
		return Fail()
	}
	// Authoritative cohort guard: the leaf's governing mapping (its own or a base
	// ancestor's, widest-space rule st-restrict-facets §3.16.6.4) must be strict's,
	// so ValidateLexical parses through a spec-exact mapping and the no-op fallback
	// (which "maps" every primitive) can never route a case through and mis-decide
	// it. Directly-mapped primitives (string/decimal/float/double) satisfy this at
	// the first step; the integer family resolves to its xs:decimal ancestor (#81).
	if !strictGoverns(strictBackend, builtinType) {
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
	_, verr := value.ValidateLexical(backend, leaf, raw, nil)
	observedValid := verr == nil
	if observedValid == c.expectValid {
		return Pass()
	}
	return Fail()
}

// strictGoverns reports whether st's governing mapping — its own or that of a
// base ancestor (widest-space rule, st-restrict-facets §3.16.6.4) — is supplied
// by the strict backend, so ValidateLexical parses through a spec-exact mapping
// rather than the no-op fallback. The integer family (xs:integer and its
// narrowings) has no strict mapping of its own; its nearest mapped ancestor is
// xs:decimal, which strict supplies (#81).
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

// fallbackPrimitives maps every builtin primitive with a no-op identity mapping.
// It once satisfied builtin.Seed's all-primitives precondition for the single
// primitive strict.New() did not cover (precisionDecimal, mapped as of #115). Now
// that strict maps all 20 primitives it is fully redundant — retained as a
// defensive floor beneath Override so Seed's precondition holds structurally rather
// than by relying on the strict cohort being complete. The datatypes selector never
// claims a case that would exercise these no-op mappings.
type fallbackPrimitives struct{}

func (fallbackPrimitives) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ.Space != xsd.XMLSchemaNS {
		return value.Mapping{}, false
	}
	for _, t := range builtin.Types {
		if t.IsPrimitive() && t.Name == typ.Local {
			return value.Mapping{
				Parse: func(lexical string, _ value.Context) (value.Value, error) { return lexical, nil },
			}, true
		}
	}
	return value.Mapping{}, false
}

// parseOK reports whether raw is in prim's lexical space, after applying prim's
// whiteSpace normalization (Datatypes §4.3.6) — collapse for boolean/decimal
// (their fixed whiteSpace facet), preserve for string. This is the lexical
// cohort's path only; the facet cohort normalizes inside value.ValidateLexical.
func parseOK(m value.Mapping, prim, raw string) bool {
	_, err := m.Parse(normalizeWhiteSpace(prim, raw), nil)
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

// facetChild is one constraining-facet element read from a Facets-cohort schema:
// its element local name (e.g. "length") and its value attribute.
type facetChild struct {
	name  string
	value string
}

// facetKinds is the set of facet kinds the facet cohort recognizes: the value-
// and pattern-facet kinds value.ValidateLexical decides for
// string/decimal/float/double (the bound facets also serve the
// partially-ordered float/double).
// whiteSpace (normalization, no cvc-* rule), assertions and explicitTimezone are
// deliberately excluded, so a schema carrying one is declined rather than
// silently ignored.
var facetKinds = []xsd.FacetKind{
	xsd.FacetLength, xsd.FacetMinLength, xsd.FacetMaxLength,
	xsd.FacetPattern, xsd.FacetEnumeration,
	xsd.FacetMaxInclusive, xsd.FacetMaxExclusive,
	xsd.FacetMinExclusive, xsd.FacetMinInclusive,
	xsd.FacetTotalDigits, xsd.FacetFractionDigits,
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
func readFacetsCase(instancePath string) (raw, base string, children []facetChild, ok bool) {
	inst, err := decodeFacetsInstance(instancePath)
	if err != nil || inst.SchemaLoc == "" || len(inst.Foos) != 1 {
		return "", "", nil, false
	}
	foo := inst.Foos[0]
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(inst.SchemaLoc))
	base, attrName, children, ok := decodeRestriction(schemaPath)
	if !ok || base == "" || len(children) == 0 {
		return "", "", nil, false
	}
	// The NMTOKEN cohort (unlike language/Name/NCName) carries the tested value
	// in a named attribute of <foo> rather than its content: the restriction is
	// declared on an <xsd:attribute>. When decodeRestriction reports that
	// attribute's name, read the matching instance attribute; otherwise the value
	// is <foo>'s element content.
	if attrName != "" {
		v, found := foo.attr(attrName)
		if !found {
			return "", "", nil, false
		}
		return v, base, children, true
	}
	return foo.Text, base, children, true
}

// facetsInstance mirrors the Facets cohort's instance shape: a <test> root whose
// single <foo> child holds the tested value in its content or a named attribute.
// Foos collects every <foo> child so readFacetsCase can require EXACTLY ONE: an
// out-of-cohort shape carrying zero <foo> leaves (e.g. the anyURI
// Facets/anyURI/anyURI_b001.xml case whose values live in repeated <bar>
// children) or several (e.g. anyURI_b006.xml, a list-style instance repeating
// many <foo> values against one enumeration) is honestly declined rather than
// mis-read as a single empty or last-wins tested value.
type facetsInstance struct {
	SchemaLoc string    `xml:"http://www.w3.org/2001/XMLSchema-instance noNamespaceSchemaLocation,attr"`
	Foos      []fooElem `xml:"foo"`
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
