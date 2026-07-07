# <a id="title"></a>An XSD datatype for IEEE floating-point decimal

<a id="title"></a>
## <a id="title"></a><a id="w3c-doctype"></a>W3C Working Group Note 9 June 2011

<a id="w3c-doctype"></a>
<a id="w3c-doctype"></a>
**This version:**
: <a id="w3c-doctype"></a>[http://www.w3.org/TR/2011/NOTE-xsd-precisionDecimal-20110609/](https://www.w3.org/TR/2011/NOTE-xsd-precisionDecimal-20110609/)

**Latest version:**
: [http://www.w3.org/TR/xsd-precisionDecimal/](https://www.w3.org/TR/xsd-precisionDecimal/)

**Editors:**
: David Peterson, invited expert (SGML*Works!*) [<davep@iit.edu>](mailto:davep@iit.edu)

: C. M. Sperberg-McQueen, Black Mesa Technologies LLC [<cmsmcq@blackmesatech.com>](mailto:cmsmcq@blackmesatech.com)

[Copyright](https://www.w3.org/Consortium/Legal/ipr-notice#Copyright)© 2011[W3C](https://www.w3.org/)® ([MIT](http://www.csail.mit.edu/), [ERCIM](http://www.ercim.eu/), [Keio](http://www.keio.ac.jp/)), All Rights Reserved. W3C [liability](https://www.w3.org/Consortium/Legal/ipr-notice#Legal_Disclaimer), [trademark](https://www.w3.org/Consortium/Legal/ipr-notice#W3C_Trademarks) and [document use](https://www.w3.org/Consortium/Legal/copyright-documents) rules apply.

---

## <a id="abstract"></a>Abstract

<a id="abstract"></a> This document defines a datatype designed for compatibility with IEEE 754 floating-point decimal data, which can be supported by XSD 1.1 processors as an [implementation-defined](https://www.w3.org/TR/xmlschema11-2/#key-impl-def) datatype.

## <a id="status"></a>Status of This Document

<a id="status"></a>
<a id="status"></a>*This section describes the status of this document at the time of its publication. Other documents may supersede this document. A list of current W3C publications and the latest revision of this technical report can be found in the **[W3C technical reports index](https://www.w3.org/TR/) at http://www.w3.org/TR/.*

This document is a W3C [Working Group Note](https://www.w3.org/2005/10/Process-20051014/tr.html#maturity-levels) as described in the [World Wide Web Consortium Process Document](https://www.w3.org/2005/10/Process-20051014/cover.html). It contains a definition of a precisionDecimal datatype designed for compatibility with IEEE 754 floating-point decimal numbers.

In its current state, this document contains all the material specific to the [precisionDecimal](#precisionDecimal) datatype that has appeared in working drafts of [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1), including some revisions made since the most recent public working draft. It is substantially complete as a specification of the datatype, though some further changes (listed in [To-do list (non-normative) (§D)](#to-do)) may be made in a future revision of this document.

Comments on this document should be sent to the W3C XML Schema comments mailing list, [www-xml-schema-comments@w3.org](mailto:www-xml-schema-comments@w3.org) ([archive](http://lists.w3.org/Archives/Public/www-xml-schema-comments/)). Each email message should contain only one comment.

Publication as a Working Group Note does not imply endorsement by the W3C Membership. This is a draft document and may be updated, replaced or obsoleted by other documents at any time. It is inappropriate to cite this document as other than work in progress.

This document has been produced by the [W3C XML Schema Working Group](https://www.w3.org/XML/Schema) as part of the W3C [XML Activity](https://www.w3.org/XML/Activity). The authors of this document are the members of the XML Schema Working Group.

This document was produced by a group operating under the [5 February 2004 W3C Patent Policy](https://www.w3.org/Consortium/Patent-Policy-20040205/). W3C maintains a [public list of any patent disclosures](https://www.w3.org/2004/01/pp-impl/19482/status) made in connection with the deliverables of the group; that page also includes instructions for disclosing a patent. An individual who has actual knowledge of a patent which the individual believes contains [Essential Claim(s)](https://www.w3.org/Consortium/Patent-Policy-20040205/#def-essential) must disclose the information in accordance with [section 6 of the W3C Patent Policy](https://www.w3.org/Consortium/Patent-Policy-20040205/#sec-Disclosure).

## <a id="contents"></a>Table of Contents

<a id="contents"></a>1 [Introduction](#intro)
2 [Definitions](#terminology)
3 [The precisionDecimal datatype](#precisionDecimal)
3.1 [Value Space](#sec-vs-pD)
3.2 [Lexical Mapping](#pD-lexical-mapping)
3.3 [Facets](#sec-f-pD)
4 [Facets for constraining precisionDecimal values](#facets)
4.1 [totalDigits](#rf-totalDigits)[totalDigits Validation Rules](#totalDigits-validation-rules) 4.2 [maxScale](#rf-maxScale)[The maxScale Schema Component](#dc-maxScale) · [XML Representation of maxScale Schema Components](#xr-maxScale) · [maxScale Validation Rules](#maxScale-validation-rules) · [Constraints on maxScale Schema Components](#maxScale-coss) 4.3 [minScale](#rf-minScale)[The minScale Schema Component](#dc-minScale) · [XML Representation of minScale Schema Components](#xr-minScale) · [minScale Validation Rules](#minScale-validation-rules) · [Constraints on minScale Schema Components](#minScale-coss) 5 [Implementation issues](#implementation-notes)
5.1 [Implementation limits](#implementation-limits)
5.2 [Interfacing with XPath](#implementing-operators)
6 [Mapping functions](#mappings)
### <a id="appendices"></a>Appendices

<a id="appendices"></a>A [Normative references](#normative-refs)
B [Non-normative references](#non-normative-refs)
C [Acknowledgements (non-normative)](#acknowledgments)
D [To-do list (non-normative)](#to-do)

---

## <a id="intro"></a>1 Introduction

<a id="intro"></a>This document defines an XSD datatype intended to support the floating-point decimal defined by IEEE 754.

<a id="intro"></a> IEEE 754 defines both floating-point binary and floating-point decimal formats. The binary formats have been widely adopted since their initial introduction; if the floating-point decimal formats are also widely adopted for applications, it will be convenient to be able to represent values of that type in XML documents or in other contexts where XSD datatypes are used.

<a id="intro"></a>
## <a id="intro"></a><a id="terminology"></a>2 Definitions

<a id="terminology"></a>The following terms are used in this specification with the meanings indicated.

<a id="terminology"></a>Except as specified below, in this specification terms defined in [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1) have the meanings given there.

<a id="dt-constraining-facet"></a>[Definition:]**constraining facet**A schema component whose value may be set or changed during [derivation](https://www.w3.org/TR/xmlschema11-2/#dt-derived) (subject to facet-specific constraints) to control various aspects of a derived datatype. <a id="dt-fundamental-facet"></a>[Definition:]**fundamental facet**A schema component that provides a limited piece of information about some aspect of a datatype. <a id="dt-specialvalue"></a>[Definition:]**special value**One of the possible values of the [·numericalValue·](#vp-pd-numVal) property of [precisionDecimal](#precisionDecimal) values, whose only relevent property, for purposes of this document and of [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1), lie in its being distinct from the other possible values; specifically, ***positiveInfinity***, ***negativeInfinity***, and ***notANumber***.Informally, any [precisionDecimal](#precisionDecimal) value whose [·numericalValue·](#vp-pd-numVal) property is such a special value.**Note:**The names of these special values are shared with the [special values](https://www.w3.org/TR/xmlschema11-2/#dt-specialvalue) of the float and double datatypes.
## <a id="precisionDecimal"></a>3 The precisionDecimal datatype

<a id="precisionDecimal"></a><a id="dt-precisionDecimal"></a>[Definition:]The **precisionDecimal**datatype represents decimal numbers which retain precision; it also includes values for positive and negative infinity and for "not a number", and it differentiates between "positive zero" and "negative zero".  This datatype is introduced to provide a variant of decimal from which may be derived datatypes closely corresponding to the floating-point decimal datatypes described by [[IEEE 754-2008]](#ieee754-2008).

**Note:**The [precisionDecimal](#precisionDecimal) datatype also permits derivation of a datatype closely corresponding to Java BigDecimal, although implementation is permitted to be confined to a much smaller value space.**Note:**Users wishing to implement useful operations for this datatype (beyond the equality and order specified herein) are urged to consult [[IEEE 754-2008]](#ieee754-2008).
The datatype [precisionDecimal](#precisionDecimal) draws its name from a common usage of the term 'precision' to mean the degree of accuracy with which a quantity is recorded. In this usage, writing a number as '`2`' or as '`2.00`' is taken as recording the value with less or more precision.

**Note:**See the [conformance note](https://www.w3.org/TR/xmlschema11-2/#partial-implementation) in [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1), which applies to this datatype.
### <a id="sec-vs-pD"></a>3.1 Value Space

<a id="sec-vs-pD"></a><a id="sec-vs-pD"></a>Properties of [precisionDecimal](#precisionDecimal) Values**<a id="vp-pd-numVal"></a>*·numericalValue·***a decimal number, ***positiveInfinity***, ***negativeInfinity***or ***notANumber*****<a id="vp-pd-precision"></a>*·scale·***an integer or ***absent***; ***absent***if and only if [·numericalValue·](#vp-pd-numVal) is a special value.**<a id="vp-pd-sign"></a>*·sign·******positive***, ***negative***, or ***absent***; must be ***positive***if [·numericalValue·](#vp-pd-numVal) is positive or ***positiveInfinity***, must be ***negative***if [·numericalValue·](#vp-pd-numVal) is negative or ***negativeInfinity***, must be ***absent***if and only if [·numericalValue·](#vp-pd-numVal) is ***notANumber*****Note:**The [·sign·](#vp-pd-sign) property is redundant except when [·numericalValue·](#vp-pd-numVal) is zero; in other cases, the [·sign·](#vp-pd-sign) value is fully determined by the [·numericalValue·](#vp-pd-numVal) value.**Note:**As explained below, '`NaN`' is the lexical representation of the [precisionDecimal](#precisionDecimal) value whose [·numericalValue·](#vp-pd-numVal) property has the special value ***notANumber***.  Accordingly, in English text we use 'NaN' to refer to that value.  Similarly we use 'INF' and '−INF' to refer to the two values whose [·numericalValue·](#vp-pd-numVal) properties have the special values ***positiveInfinity***and ***negativeInfinity***.  These three [precisionDecimal](#precisionDecimal) values are also informally called "not-a-number", "positive infinity", and "negative infinity". The latter two together are called "the infinities".**Note:**The datatype defined here is intended to allow the derivation of less general datatypes corresponding to the decimal formats defined by [[IEEE 754-2008]](#ieee754-2008). Those formats can be viewed as representing values other than the infinities and NaN as triples (*s*, *q*, *m*), where *s*is the *sign*bit, *q*is the *exponent*(an integer), and *m*is the *significand*(also an integer). (The "*q*" form of IEEE's *exponent*is used when treating the *significand*as an integer.) The [precisionDecimal](#precisionDecimal)[·numericalValue·](#vp-pd-numVal) is (–1 ^ *s*) × (10 ^ *q*) × *m*and the [·scale·](#vp-pd-precision) is *q*. Conversely, for nonzero finite values the *sign**s*is the [·numericalValue·](#vp-pd-numVal) divided by its absolute value, the integer *significand**m*is |[·numericalValue·](#vp-pd-numVal)| / (10 ^ [·scale·](#vp-pd-precision)), and, of course, the exponent *q*is the [·scale·](#vp-pd-precision). The single NaN of [precisionDecimal](#precisionDecimal) corresponds both to the signaling NaN and to the quiet NaN of [IEEE 754-2008], which permits the use of a single NaN when values are being transmitted from one system to another via [lexical representations](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-representation). The individual decimal formats defined by [[IEEE 754-2008]](#ieee754-2008) are characterized by the range of values allowed for the *exponent*and by the number of decimal digits available for the *significand*, which IEEE terms the *precision*of the format. In any given IEEE 754 decimal format there may be multiple representations representing the same numerical value, one with its significand using the maximum number of significant digits available in the format, and the others with fewer significant digits (when that is possible). Note that in some cases these distinct representations will result in distinct results in operations defined by IEEE 754. For datatypes derived from [precisionDecimal](#precisionDecimal), setting the [facet is equivalent to restricting the number of decimal digits available for the significand, and setting the](https://www.w3.org/TR/xmlschema11-2/#dt-totalDigits)[·minScale·](#dt-minScale) and [·maxScale·](#dt-maxScale) facets amounts to controlling the possible values of the exponent *q*. **Note:**[precisionDecimal](#precisionDecimal) also allows the derivation of the Java BigDecimal class, which corresponds in essentials to a datatype derived from [precisionDecimal](#precisionDecimal) by limiting the allowed scale, by eliminating the special values, and by ignoring the difference between +0 and –0. Equality and order for [precisionDecimal](#precisionDecimal) are defined as follows:
- Two numerical [precisionDecimal](#precisionDecimal) values are ordered (or equal) as their [·numericalValue·](#vp-pd-numVal) values are ordered (or equal).  (This means that two zeroes with different [·sign·](#vp-pd-sign) properties are *equal*; negative zeroes are *not*ordered less than positive zeroes.)
- INF is equal only to itself, and is greater than −INF and all numerical [precisionDecimal](#precisionDecimal) values.
- −INF is equal only to itself, and is less than INF and all numerical [precisionDecimal](#precisionDecimal) values.
- NaN is [incomparable](https://www.w3.org/TR/xmlschema11-2/#dt-incomparable) with all values, *including itself*.
### <a id="pD-lexical-mapping"></a>3.2 Lexical Mapping

<a id="pD-lexical-mapping"></a>The lexical space of [precisionDecimal](#precisionDecimal) is the set of all decimal numerals with or without a decimal point, numerals in scientific (exponential) notation, and the character strings '`INF`', '`+INF`', '`-INF`', and '`NaN`'. Lexical Space<a id="nt-precDecRep"></a>[1] *pDecimalRep*::= [noDecimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-noDecNuml) | [decimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-decNuml) | [scientificNotationNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-sciNuml) | [numericalSpecialRep](https://www.w3.org/TR/xmlschema11-2/#nt-numSpecReps)**Note:**The four non-terminals referred to on the right-hand side of the [pDecimalRep](#nt-precDecRep) are defined in [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1). The [pDecimalRep](#nt-precDecRep) production is equivalent (after whitespace is removed) to the following regular expression:
> `(\+|-)?([0-9]+(\.[0-9]*)?|\.[0-9]+)([Ee](\+|-)?[0-9]+)? |(\+|-)?INF|NaN`

The [lexical mapping](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-mapping) for [precisionDecimal](#precisionDecimal) is [·precisionDecimalLexicalMap·](#f-precDecLexmap).  The [canonical mapping](https://www.w3.org/TR/xmlschema11-2/#dt-canonical-mapping) is [·precisionDecimalCanonicalMap·](#f-precDecCanmap).

For example, each of the [lexical representations](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-representation) shown below is followed by its corresponding value triple ([·numericalValue·](#vp-pd-numVal), [·scale·](#vp-pd-precision), and [·sign·](#vp-pd-sign)) and [canonical representation](https://www.w3.org/TR/xmlschema11-2/#dt-canonical-representation):
- '`3`'   ( 3 ,  0 , ***positive***)   '`3`'
- '`3.00`'   ( 3 ,  2 , ***positive***)   '`3.00`'
- '`03.00`'   ( 3 ,  2 , ***positive***)   '`3.00`'
- '`300`'   ( 300 ,  0 , ***positive***)   '`300`'
- '`3.00e2`'   ( 300 ,  0 , ***positive***)   '`300`'
- '`3.0e2`'   ( 300 ,  −1 , ***positive***)   '`3.0E2`'
- '`30e1`'   ( 300 ,  −1 , ***positive***)   '`3.0E2`'
- '`.30e3`'   ( 300 ,  −1 , ***positive***)   '`3.0E2`'
Note that the last three examples not only show different [lexical representations](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-representation) for the same value, but are of particular interest because values with negative precision can *only*have [lexical representations](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-representation) in scientific notation.**Note:**[[IEEE 754-2008]](#ieee754-2008) expects [lexical representations](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-representation) whose exact value is not in the [value space](https://www.w3.org/TR/xmlschema11-2/#dt-value-space) to be mapped to the nearest value that is in the [value space](https://www.w3.org/TR/xmlschema11-2/#dt-value-space). When [precisionDecimal](#precisionDecimal) is restricted, all [lexical representations](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-representation) of values dropped from the value space are dropped from the lexical space. One result is that when [precisionDecimal](#precisionDecimal) is restricted using the [or](https://www.w3.org/TR/xmlschema11-2/#dt-totalDigits)[·maxScale·](#dt-maxScale) facets, non-zero digits beyond those required to exactly represent the intended value are not permitted by this specification. [[IEEE 754-2008]](#ieee754-2008) permits all case variants of '`INF`' and '`NaN`', as well as those of '`INFINITY`'; in many cases it permits language definitions to prescribe which variants are used. This specification explicitly chooses only '`INF`' and '`NaN`'. 754 also permits language definitions to prescribe whether '`+`' shall be used with positive values; this specification makes the '`+`' optional.**Note:**Note: The [lexical representations](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-representation) with "unnecessary least significant digits" representations are the only ones lost when [precisionDecimal](#precisionDecimal) is restricted; shorter and simpler [lexical representations](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-representation) will not be eliminated by use of [or](https://www.w3.org/TR/xmlschema11-2/#dt-totalDigits)[·maxScale·](#dt-maxScale). (In contrast, if facets for controlling total digits and scale were available for the floating-point binary types, the effect of restriction would often be inconvenient. In the floating-point binary types, a simple decimal numeral will sometimes have no exact value and so the number will be rounded to a binary approximation. Any restriction of the value space which dropped that approximate value would automatically also drop the simple decimal numeral from the lexical space. For example, the number one-tenth is in the value space neither of float nor of double; the string '`0.1`' maps, in double, to 0.1000000000000000055511151231257827021181583404541015625. If the [,](https://www.w3.org/TR/xmlschema11-2/#dt-totalDigits)[·minScale·](#dt-minScale), and [·maxScale·](#dt-maxScale) facets were available for double (they are not) and were used to define the float type (again, they are not), the value just mentioned would be dropped, and the [literal](https://www.w3.org/TR/xmlschema11-2/#dt-literal) '`0.1`' would be dropped along with it, instead of mapping (as in fact it does) to the value 0.100000001490116119384765625. This interaction between datatype restriction and rounding has as a consequence that it will typically be more convenient for users if restricted-precision numeric types are derived from [precisionDecimal](#precisionDecimal) than it would be if they were derived from [float](https://www.w3.org/TR/xmlschema11-2/#float) or [double](https://www.w3.org/TR/xmlschema11-2/#double).
### <a id="sec-f-pD"></a>3.3 Facets

<a id="sec-f-pD"></a>The [precisionDecimal](#precisionDecimal) datatype and all datatypes derived from it by restriction have the following [·constraining facets·](#dt-constraining-facet) with ***fixed***values; these facets must not be changed from the values shown:

- <a id="precisionDecimal.whiteSpace"></a>[<a id="precisionDecimal.whiteSpace"></a>whiteSpace](https://www.w3.org/TR/xmlschema11-2/#rf-whiteSpace) = ***collapse***(fixed)
Datatypes derived by restriction from [precisionDecimal](#precisionDecimal)may also specify values for the following [·constraining facets·](https://www.w3.org/TR/xmlschema11-2/#dt-constraining-facet):

- [totalDigits](https://www.w3.org/TR/xmlschema11-2/#rf-totalDigits)
- [maxScale](#rf-maxScale)
- [minScale](#rf-minScale)
- [pattern](https://www.w3.org/TR/xmlschema11-2/#rf-pattern)
- [enumeration](https://www.w3.org/TR/xmlschema11-2/#rf-enumeration)
- [maxInclusive](https://www.w3.org/TR/xmlschema11-2/#rf-maxInclusive)
- [maxExclusive](https://www.w3.org/TR/xmlschema11-2/#rf-maxExclusive)
- [minInclusive](https://www.w3.org/TR/xmlschema11-2/#rf-minInclusive)
- [minExclusive](https://www.w3.org/TR/xmlschema11-2/#rf-minExclusive)
- [assertions](https://www.w3.org/TR/xmlschema11-2/#rf-assertions)
The [precisionDecimal](#precisionDecimal) datatype has the following values for its [·fundamental facets·](#dt-fundamental-facet):

- [ordered](https://www.w3.org/TR/xmlschema11-2/#rf-ordered) = ***partial***
- [bounded](https://www.w3.org/TR/xmlschema11-2/#rf-bounded) = ***false***
- [cardinality](https://www.w3.org/TR/xmlschema11-2/#rf-cardinality) = ***countably infinite***
- [numeric](https://www.w3.org/TR/xmlschema11-2/#rf-numeric) = ***true***
## <a id="facets"></a>4 Facets for constraining precisionDecimal values

<a id="facets"></a> The [assertions](https://www.w3.org/TR/xmlschema11-2/#dt-assertions), [enumeration](https://www.w3.org/TR/xmlschema11-2/#dt-enumeration), [maxInclusive](https://www.w3.org/TR/xmlschema11-2/#dt-maxInclusive), [maxExclusive](https://www.w3.org/TR/xmlschema11-2/#dt-maxExclusive), [minExclusive](https://www.w3.org/TR/xmlschema11-2/#dt-minExclusive), [minInclusive](https://www.w3.org/TR/xmlschema11-2/#dt-minInclusive), and [pattern](https://www.w3.org/TR/xmlschema11-2/#dt-pattern) facets defined by [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1) can be used in deriving new types from [precisionDecimal](#precisionDecimal) by restriction; their meaning and use are as documented in [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1).

The [totalDigits](https://www.w3.org/TR/xmlschema11-2/#dt-totalDigits) facet defined by [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1) can also be used. Its meaning, when applied to values of type [precisionDecimal](#precisionDecimal), is described in [totalDigits (§4.1)](#rf-totalDigits). Except as otherwise specified in [totalDigits (§4.1)](#rf-totalDigits), all the constraints on the use of the [totalDigits](https://www.w3.org/TR/xmlschema11-2/#dt-totalDigits) facet described in [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1) continue to apply when the facet is used with [precisionDecimal](#precisionDecimal) values.

In addition, two facets not defined by [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1) can be used when restricting [precisionDecimal](#precisionDecimal). They are described in [maxScale (§4.2)](#rf-maxScale) and [minScale (§4.3)](#rf-minScale).

### <a id="rf-totalDigits"></a>4.1 totalDigits

<a id="rf-totalDigits"></a> For [precisionDecimal](#precisionDecimal) values with [·numericalValue·](#vp-pd-numVal) of *nV*and [·scale·](#vp-pd-precision) of *aP*, if the [value](https://www.w3.org/TR/xmlschema11-2/#f-td-value) of [is t, the effect of the](https://www.w3.org/TR/xmlschema11-2/#f-td)[facet is to require that (aP + 1 + log10(| nV |)](https://www.w3.org/TR/xmlschema11-2/#f-td)[div](https://www.w3.org/TR/xmlschema11-2/#dt-div) 1) ≤ *t*, for values other than zero, NaN, and the infinities. This means in effect that values are expressible in scientific notation using at most *t*digits for the coefficient.

#### <a id="totalDigits-validation-rules"></a>4.1.1 totalDigits Validation Rules

<a id="cvc-totalDigits-valid"></a>**Validation Rule: totalDigits Valid**
<a id="cvc-totalDigits-valid"></a><a id="cvc-totalDigits-valid"></a>A [precisionDecimal](#precisionDecimal) value *v*is facet-valid with respect to a [facet with a](https://www.w3.org/TR/xmlschema11-2/#f-td)[value](https://www.w3.org/TR/xmlschema11-2/#f-td-value) of *t*if and only if one of the following is true: 1 *v*is a [precisionDecimal](#precisionDecimal) value with [·numericalValue·](#vp-pd-numVal) of ***positiveInfinity***, ***negativeInfinity***, ***notANumber***, or zero. 2 *v*is a [precisionDecimal](#precisionDecimal) value with [·numericalValue·](#vp-pd-numVal) of *nV*and [·scale·](#vp-pd-precision) of *aP*, and *v*is not NaN, INF, -INF, or zero, and (*aP*+ 1 + log10(|*nV*|) [div](https://www.w3.org/TR/xmlschema11-2/#dt-div) 1) ≤ *t*.
### <a id="rf-maxScale"></a>4.2 maxScale

<a id="rf-maxScale"></a>4.2.1 [The maxScale Schema Component](#dc-maxScale)
4.2.2 [XML Representation of maxScale Schema Components](#xr-maxScale)
4.2.3 [maxScale Validation Rules](#maxScale-validation-rules)
4.2.4 [Constraints on maxScale Schema Components](#maxScale-coss)
<a id="dt-maxScale"></a>[Definition:]**maxScale**places an upper limit on the [·scale·](#vp-pd-precision) of [precisionDecimal](#precisionDecimal) values: if the [{value}](#f-ms-value) of **maxScale**= *m*, then only values with [·scale·](#vp-pd-precision) ≤ *m*are retained in the [value space](https://www.w3.org/TR/xmlschema11-2/#dt-value-space). As a consequence, every value in the value space will have [·numericalValue·](#vp-pd-numVal) equal to *i*/ 10*n*for some integers *i*and *n*, with *n*≤ *m*. The [{value}](#f-ms-value) of [maxScale](#f-ms)must be an [integer](https://www.w3.org/TR/xmlschema11-2/#integer). If it is negative, the numeric values of the datatype are restricted to multiples of 10 (or 100, or …).

The term 'maxScale' is chosen to reflect the fact that it restricts the [value space](https://www.w3.org/TR/xmlschema11-2/#dt-value-space) to those values that can be represented lexically in scientific notation using an integer coefficient and a scale (or negative exponent) no greater than [maxScale](#f-ms). (It has nothing to do with the use of the term 'scale' to denote the radix or base of a notation.) Note that [maxScale](#f-ms) does not restrict the [lexical space](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-space) directly; a lexical representation that adds non-significant leading or trailing zero digits, or that uses a lower exponent with a non-integer coefficient is still permitted.

Example The following is the definition of a user-defined datatype which could be used to represent a floating-point decimal datatype which allows seven decimal digits for the coefficient and exponents between −95 and 96. Note that the scale is −1 times the exponent.
```
<simpleType name='decimal32'>
  <restriction base='precisionDecimal'>
    <totalDigits value='7'/>
    <maxScale value='95'/>
    <minScale value='-96'/>
  </restriction>
</simpleType>
```

#### <a id="dc-maxScale"></a>4.2.1 The maxScale Schema Component

<a id="dc-maxScale"></a><a id="dc-maxScale"></a><a id="dc-maxScale"></a>Schema Component: <a id="f-ms"></a>maxScale<a id="f-ms-annotations"></a>{annotations}<a id="f-ms-annotations"></a> A sequence of [Annotation](https://www.w3.org/TR/xmlschema11-1/#a) components. <a id="f-ms-value"></a>{value} An xs:integer value. Required.<a id="f-ms-value"></a><a id="f-ms-value"></a><a id="f-ms-fixed"></a>{fixed} An xs:boolean value. Required.<a id="f-ms-fixed"></a><a id="f-ms-fixed"></a><a id="f-ms-fixed"></a><a id="f-ms-fixed"></a>
<a id="f-ms-fixed"></a> If [{fixed}](#f-ms-fixed) is ***true***, then types for which the current type is the [{base type definition}](https://www.w3.org/TR/xmlschema11-2/#std-base_type_definition)must not specify a value for [maxScale](#f-ms) other than [{value}](#f-ms-value).

#### <a id="xr-maxScale"></a>4.2.2 XML Representation of maxScale Schema Components

<a id="xr-maxScale"></a> The XML representation for a [maxScale](#f-ms) schema component is a [<maxScale>](https://www.w3.org/TR/xmlschema11-2/#element-maxScale) element information item. The correspondences between the properties of the information item and properties of the component are as follows:

XML Representation Summary: `maxScale`Element Information Item

[maxScale](#dc-maxScale)**Schema Component****Property****Representation**[{value}](#f-ms-value) The [actual value](https://www.w3.org/TR/xmlschema11-1/#key-vv) of the `value`[[attribute]](https://www.w3.org/TR/xml-infoset/#infoitem.element)[{fixed}](#f-ms-fixed) The [actual value](https://www.w3.org/TR/xmlschema11-1/#key-vv) of the `fixed`[[attribute]](https://www.w3.org/TR/xml-infoset/#infoitem.element), if present, otherwise ***false***[{annotations}](#f-ms-annotations) The [annotation mapping](https://www.w3.org/TR/xmlschema11-1/#key-am-one) of the [<maxScale>](https://www.w3.org/TR/xmlschema11-2/#element-maxScale) element, as defined in section [XML Representation of Annotation Schema Components](https://www.w3.org/TR/xmlschema11-1/#declare-annotation) of [[XSD 1.1 Part 1: Structures]](#structures-1.1).
#### <a id="maxScale-validation-rules"></a>4.2.3 maxScale Validation Rules

<a id="cvc-maxScale-valid"></a>**Validation Rule: maxScale Valid**
<a id="cvc-maxScale-valid"></a><a id="cvc-maxScale-valid"></a> A [precisionDecimal](#precisionDecimal) value *v*is facet-valid with respect to [maxScale](#f-ms) if and only if one of the following is true: 1 *v*has [·scale·](#vp-pd-precision) less than or equal to the [{value}](#f-ms-value) of [maxScale](#f-ms).2 The [·scale·](#vp-pd-precision) of *v*is ***absent***.
#### <a id="maxScale-coss"></a>4.2.4 Constraints on maxScale Schema Components

<a id="maxScale-valid-restriction"></a>**Schema Component Constraint: maxScale valid restriction**
<a id="maxScale-valid-restriction"></a><a id="maxScale-valid-restriction"></a> It is an [error](https://www.w3.org/TR/xmlschema11-2/#dt-error) if [maxScale](#f-ms) is among the members of [{facets}](https://www.w3.org/TR/xmlschema11-2/#std-facets) of [{base type definition}](https://www.w3.org/TR/xmlschema11-2/#std-base_type_definition) and [{value}](#f-ms-value) is greater than the [{value}](#f-ms-value) of that [maxScale](#f-ms).
### <a id="rf-minScale"></a>4.3 minScale

<a id="rf-minScale"></a>4.3.1 [The minScale Schema Component](#dc-minScale)
4.3.2 [XML Representation of minScale Schema Components](#xr-minScale)
4.3.3 [minScale Validation Rules](#minScale-validation-rules)
4.3.4 [Constraints on minScale Schema Components](#minScale-coss)
<a id="dt-minScale"></a>[Definition:]**minScale**places a lower limit on the [·scale·](#vp-pd-precision) of [precisionDecimal](#precisionDecimal) values. If the [{value}](#f-mns-value) of **minScale**is *m*, then the value space is restricted to values with [·scale·](#vp-pd-precision) ≥ *m*. As a consequence, every value in the value space will have [·numericalValue·](#vp-pd-numVal) equal to *i*/ 10*n*for some integers *i*and *n*, with *n*≥ *m*.

The term **minScale**is chosen to reflect the fact that it restricts the [value space](https://www.w3.org/TR/xmlschema11-2/#dt-value-space) to those values that can be represented lexically in exponential form using an integer coefficient and a scale (negative exponent) at least as large as *minScale*. Note that it does not restrict the [lexical space](https://www.w3.org/TR/xmlschema11-2/#dt-lexical-space) directly; a lexical representation that adds additional leading zero digits, or that uses a larger exponent (and a correspondingly smaller coefficient) is still permitted.

Example The following is the definition of a user-defined datatype which could be used to represent amounts in a decimal currency; it corresponds to a SQL column definition of `DECIMAL(8,2)`. The effect is to allow values between -999,999.99 and 999,999.99, with a fixed interval of 0.01 between values.
```
<simpleType name='price'>
  <restriction base='precisionDecimal'>
    <totalDigits value='8'/>
    <minScale value='2'/>
    <maxScale value='2'/>
  </restriction>
</simpleType>
```

#### <a id="dc-minScale"></a>4.3.1 The minScale Schema Component

<a id="dc-minScale"></a><a id="dc-minScale"></a><a id="dc-minScale"></a>Schema Component: <a id="f-mns"></a>minScale<a id="f-mns-annotations"></a>{annotations}<a id="f-mns-annotations"></a> A sequence of [Annotation](https://www.w3.org/TR/xmlschema11-1/#a) components. <a id="f-mns-value"></a>{value} An xs:integer value. Required.<a id="f-mns-value"></a><a id="f-mns-value"></a><a id="f-mns-fixed"></a>{fixed} An xs:boolean value. Required.<a id="f-mns-fixed"></a><a id="f-mns-fixed"></a><a id="f-mns-fixed"></a><a id="f-mns-fixed"></a>
<a id="f-mns-fixed"></a> If [{fixed}](#f-mns-fixed) is ***true***, then types for which the current type is the [{base type definition}](https://www.w3.org/TR/xmlschema11-2/#std-base_type_definition)must not specify a value for [minScale](#f-mns) other than [{value}](#f-mns-value).

#### <a id="xr-minScale"></a>4.3.2 XML Representation of minScale Schema Components

<a id="xr-minScale"></a> The XML representation for a [minScale](#f-mns) schema component is a [<minScale>](https://www.w3.org/TR/xmlschema11-2/#element-minScale) element information item. The correspondences between the properties of the information item and properties of the component are as follows:

XML Representation Summary: `minScale`Element Information Item

[minScale](#dc-minScale)**Schema Component****Property****Representation**[{value}](#f-mns-value) The [actual value](https://www.w3.org/TR/xmlschema11-1/#key-vv) of the `value`[[attribute]](https://www.w3.org/TR/xml-infoset/#infoitem.element)[{fixed}](#f-mns-fixed) The [actual value](https://www.w3.org/TR/xmlschema11-1/#key-vv) of the `fixed`[[attribute]](https://www.w3.org/TR/xml-infoset/#infoitem.element), if present, otherwise ***false***[{annotations}](#f-mns-annotations) The [annotation mapping](https://www.w3.org/TR/xmlschema11-1/#key-am-one) of the [<minScale>](https://www.w3.org/TR/xmlschema11-2/#element-minScale) element, as defined in section [XML Representation of Annotation Schema Components](https://www.w3.org/TR/xmlschema11-1/#declare-annotation) of [[XSD 1.1 Part 1: Structures]](#structures-1.1).
#### <a id="minScale-validation-rules"></a>4.3.3 minScale Validation Rules

<a id="cvc-minScale-valid"></a>**Validation Rule: minScale Valid**
<a id="cvc-minScale-valid"></a><a id="cvc-minScale-valid"></a> A [precisionDecimal](#precisionDecimal) value *v*is facet-valid with respect to [minScale](#f-mns) if and only if one of the following is true: 1 *v*has [·scale·](#vp-pd-precision) greater than or equal to the [{value}](#f-mns-value) of [minScale](#f-mns). 2 The [·scale·](#vp-pd-precision) of *v*is ***absent***.
#### <a id="minScale-coss"></a>4.3.4 Constraints on minScale Schema Components

<a id="minScale-totalDigits"></a>**Schema Component Constraint: minScale less than or equal to maxScale**
<a id="minScale-totalDigits"></a><a id="minScale-totalDigits"></a> It is an [error](https://www.w3.org/TR/xmlschema11-2/#dt-error) for [minScale](#f-mns) to be greater than [maxScale](#f-ms).
Note that it is *not*an error for [minScale](#f-mns) to be greater than [.](https://www.w3.org/TR/xmlschema11-2/#f-td)

<a id="minScale-valid-restriction"></a>**Schema Component Constraint: minScale valid restriction**
<a id="minScale-valid-restriction"></a><a id="minScale-valid-restriction"></a> It is an [error](https://www.w3.org/TR/xmlschema11-2/#dt-error) if [minScale](#f-mns) is among the members of [{facets}](https://www.w3.org/TR/xmlschema11-2/#std-facets) of [{base type definition}](https://www.w3.org/TR/xmlschema11-2/#std-base_type_definition) and [{value}](#f-mns-value) is less than the [{value}](#f-mns-value) of that [minScale](#f-mns).
## <a id="implementation-notes"></a>5 Implementation issues

<a id="implementation-notes"></a>
### <a id="implementation-notes"></a> <a id="implementation-limits"></a>5.1 Implementation limits

<a id="implementation-limits"></a>All [minimally conforming](https://www.w3.org/TR/xmlschema11-2/#dt-minimally-conforming) processors must support all [precisionDecimal](#precisionDecimal) values in the [value space](https://www.w3.org/TR/xmlschema11-2/#dt-value-space) of the otherwise unconstrained [derived](https://www.w3.org/TR/xmlschema11-2/#dt-derived) datatype for which [is set to sixteen,](https://www.w3.org/TR/xmlschema11-2/#f-td)[maxScale](#f-ms) to 369, and [minScale](#f-mns) to −398.

**Note:**The conformance limits given in the text correspond to those of the decimal64 type defined in [[IEEE 754-2008]](#ieee754-2008), which can be stored in a 64-bit field. The XML Schema Working Group recommends that implementors support limits corresponding to those of the decimal128 type. This entails supporting the values in the value space of the otherwise unconstrained datatype for which [is set to 34,](https://www.w3.org/TR/xmlschema11-2/#f-td)[maxScale](#f-ms) to 6111, and [minScale](#f-mns) to −6176.
### <a id="implementing-operators"></a>5.2 Interfacing with XPath

[[XPath 2.0]](#bib-xpath2) does not currently require support for the precisionDecimal datatype, but conforming XPath processors are allowed to support additional primitive data types, including precisionDecimal.

For interoperability, it is recommended that XPath processors intending to support precisionDecimal as an additional primitive data type follow the recommendations in [[Chamberlin 2006]](#bib-chamberlin-2006). If the XPath processor used to evaluate XPath expressions supports precisionDecimal, then any precisionDecimal values in the [post-schema-validation infoset](https://www.w3.org/TR/xmlschema11-1/#key-psvi)should be labeled as `xs:precisionDecimal`in the data model instance and handled accordingly in XPath.

If the XPath processor does not support precisionDecimal, then any precisionDecimal values in the [post-schema-validation infoset](https://www.w3.org/TR/xmlschema11-1/#key-psvi)should be mapped into [decimal](https://www.w3.org/TR/xmlschema11-2/#decimal), unless the [·numericalValue·](#vp-pd-numVal) is not a decimal number (for example, it is ***positiveInfinity***, ***negativeInfinity***, or ***notANumber***), in which case they should be mapped to [float](https://www.w3.org/TR/xmlschema11-2/#float). Whether this is done by altering the type information in the partial [post-schema-validation infoset](https://www.w3.org/TR/xmlschema11-1/#key-psvi), or by altering the usual rules for mapping from a [post-schema-validation infoset](https://www.w3.org/TR/xmlschema11-1/#key-psvi) to an [[XDM]](#bib-xdm) data model instance, or by treating precisionDecimal as an unknown type which is coerced as appropriate into decimal or float by the XPath processor, is [implementation-defined](https://www.w3.org/TR/xmlschema11-2/#key-impl-def) and out of scope for this specification.

As a consequence of the above variability, it is possible that XPath expressions that perform various kinds of type introspections will produce different results when different XPath processors are used. If the schema author wishes to ensure interoperable results, such introspections will need to be avoided.

## <a id="mappings"></a>6 Mapping functions

<a id="mappings"></a> The functions defined below make frequent reference to functions defined in [[XSD 1.1 Part 2: Datatypes]](#datatypes-1.1).

Auxiliary Functions for Reading Instances of [pDecimalRep](#nt-precDecRep)**<a id="vp-decPrecision"></a>*·decimalPtPrecision·***(*LEX*) → integer Maps a [decimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-decNuml) onto an integer; used in calculating the [·scale·](#vp-pd-precision) of a [precisionDecimal](#precisionDecimal) value.**Arguments:**
| *LEX* | : | matches decimalPtNumeral |
| --- | --- | --- |

**Result:**an integer**Algorithm:***LEX*necessarily contains a decimal point ('`.`') and may optionally contain a following [fracFrag](https://www.w3.org/TR/xmlschema11-2/#nt-fracFrag)*F*consisting of some number *n*of [digit](https://www.w3.org/TR/xmlschema11-2/#nt-digit)s. Return
- *n*when *F*is present, and
- 0   otherwise.
**<a id="vp-sciPrecision"></a>*·scientificPrecision·***(*LEX*) → integer Maps a [scientificNotationNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-sciNuml) onto an integer; used in calculating the [·scale·](#vp-pd-precision) of a [precisionDecimal](#precisionDecimal) value.**Arguments:**
| *LEX* | : | matches scientificNotationNumeral |
| --- | --- | --- |

**Result:**an integer**Algorithm:***LEX*necessarily contains a [noDecimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-noDecNuml) or [decimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-decNuml)*C*preceding an exponent indicator ('`E`' or '`e`', and a following [noDecimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-noDecNuml)*E*.Return
- −1 ×[noDecimalMap](https://www.w3.org/TR/xmlschema11-2/#f-noDecVal)(*E*)   when *C*is a [noDecimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-noDecNuml), and
- [·decimalPtPrecision·](#vp-decPrecision)(*C*) − [noDecimalMap](https://www.w3.org/TR/xmlschema11-2/#f-noDecVal)(*E*)   otherwise.
Lexical Mapping**<a id="f-precDecLexmap"></a>*·precisionDecimalLexicalMap·***(*LEX*) → [precisionDecimal](#precisionDecimal)Maps a [pDecimalRep](#nt-precDecRep) onto a complete [precisionDecimal](#precisionDecimal) value.**Arguments:**
| *LEX* | : | matches pDecimalRep |
| --- | --- | --- |

**Result:**a [precisionDecimal](#precisionDecimal) value**Algorithm:**
| Let | *pD*be a complete precisionDecimal value. |
| --- | --- |

1. Set *pD*'s [·numericalValue·](#vp-pd-numVal) to
  - [noDecimalMap](https://www.w3.org/TR/xmlschema11-2/#f-noDecVal)(*LEX*)   when *LEX*is an instance of [noDecimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-noDecNuml),
  - [decimalPtMap](https://www.w3.org/TR/xmlschema11-2/#f-decVal)(*LEX*)   when *LEX*is an instance of [decimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-decNuml),
  - [scientificMap](https://www.w3.org/TR/xmlschema11-2/#f-sciVal)(*LEX*)   when *LEX*is an instance of [scientificNotationNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-sciNuml) and
  - [specialRepValue](https://www.w3.org/TR/xmlschema11-2/#f-specRepVal)(*LEX*)   otherwise.

2. Set *pD*'s [·scale·](#vp-pd-precision) to
  - 0   when *LEX*is a [noDecimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-noDecNuml),
  - [·decimalPtPrecision·](#vp-decPrecision)(*LEX*)   when *LEX*is a [decimalPtNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-decNuml),
  - [·scientificPrecision·](#vp-sciPrecision)(*LEX*)   when *LEX*is a [scientificNotationNumeral](https://www.w3.org/TR/xmlschema11-2/#nt-sciNuml), and
  - ***absent***otherwise

3. Set *pD*'s [·sign·](#vp-pd-sign) to
  - ***absent***when *LEX*is '`NaN`'
  - ***negative***when the first character of *LEX*is '`-`', and
  - ***positive***otherwise.

4. Return *pD*.
Canonical Mapping**<a id="f-precDecCanmap"></a>*·precisionDecimalCanonicalMap·***(*pD*) → [pDecimalRep](#nt-precDecRep)Maps a [precisionDecimal](#precisionDecimal) to its [canonical representation](https://www.w3.org/TR/xmlschema11-2/#dt-canonical-representation), a [pDecimalRep](#nt-precDecRep).**Arguments:**
| *pD* | : | a precisionDecimal value |
| --- | --- | --- |

**Result:**a [literal](https://www.w3.org/TR/xmlschema11-2/#dt-literal) matching [pDecimalRep](#nt-precDecRep)**Algorithm:**
1. Let *nV*be the [·numericalValue·](#vp-pd-numVal) of *pD*.Let *aP*be the [·scale·](#vp-pd-precision) of *pD*.
2. If *pD*is one of NaN, INF, or -INF, then return [specialRepCanonicalMap](https://www.w3.org/TR/xmlschema11-2/#f-specValCanMap)(*nV*).
3. Otherwise, if *nV*is an integer and *aP*is zero and 1E-6 ≤ *nV*≤ 1E6, then return [noDecimalPtCanonicalMap](https://www.w3.org/TR/xmlschema11-2/#f-noDecCanMap)(*nV*).
4. Otherwise, if *aP*is greater than zero and 1E-6 ≤ *nV*≤ 1E6, then let *s*be [decimalPtCanonicalMap](https://www.w3.org/TR/xmlschema11-2/#f-decCanFragMap)(*nV*). Let *f*be the number of fractional digits in *s*; *f*will invariably be less than or equal to *aP*. Return the concatenation of *s*with *aP*−*f*occurrences of the digit '`0`'.
5. Otherwise, it will be the case that *nV*is less than 1E−6 or greater than 1E6, or that *aP*is less than zero.  Let
  - *s*be [scientificCanonicalMap](https://www.w3.org/TR/xmlschema11-2/#f-sciCanFragMap)(*nV*).
  - *m*be the part of *s*which precedes the "E".
  - *n*be the part of *s*which follows the "E".
  - *p*be the integer denoted by *n*.
  - *f*be the number of fractional digits in *m*; note that *f*will invariably be less than or equal to *aP*+*p*.
  - *t*be a string consisting of *aP*+*p*−*f*occurrences of the digit '`0`', preceded by a decimal point if and only if *m*contains no decimal point and *aP*+*p*−*f*is greater than zero.
Return the concatenation *m*& *t*& '`E`' & *n*.
## <a id="normative-refs"></a>A Normative references

**<a id="ieee754-2008"></a>IEEE 754-2008**
: <a id="ieee754-2008"></a> IEEE. *IEEE Standard for Floating-Point Arithmetic*. 29 August 2008. [http://ieeexplore.ieee.org/xpl/mostRecentIssue.jsp?punumber=4610933](http://ieeexplore.ieee.org/xpl/mostRecentIssue.jsp?punumber=4610933)

**<a id="datatypes-1.1"></a>XSD 1.1 Part 2: Datatypes**
: <a id="datatypes-1.1"></a> World Wide Web Consortium. *W3C XML Schema Definition Language (XSD) 1.1 Part 2: Structures*, ed. David Peterson et al. W3C Working Draft 3 December 2009. Available at: [http://www.w3.org/TR/xmlschema11-2/](https://www.w3.org/TR/xmlschema11-2/)

## <a id="non-normative-refs"></a>B Non-normative references

**<a id="bib-chamberlin-2006"></a>Chamberlin 2006**
: <a id="bib-chamberlin-2006"></a> Chamberlin, Don. *Impact of precisionDecimal on XPath and XQuery*Email to the W3C XML Query and W3C XSL Working Groups, 16 May 2006. Available online at [http://www.w3.org/XML/2007/dc.pd.xml](https://www.w3.org/XML/2007/dc.pd.xml) and [http://www.w3.org/XML/2007/dc.pd.html](https://www.w3.org/XML/2007/dc.pd.html)

**<a id="bib-xdm"></a>XDM**
: <a id="bib-xdm"></a> World Wide Web Consortium. *XQuery 1.0 and XPath 2.0 Data Model (XDM)*, ed. Mary Fernández et al. W3C Recommendation 23 January 2007. Available at: [http://www.w3.org/TR/xpath-datamodel/](https://www.w3.org/TR/xpath-datamodel/).

**<a id="bib-xpath2"></a>XPath 2.0**
: <a id="bib-xpath2"></a> World Wide Web Consortium. *XML Path Language 2.0*, ed. Anders Berglund et al. 23 January 2007. Available at: [http://www.w3.org/TR/2007/REC-xpath20-20070123/](https://www.w3.org/TR/2007/REC-xpath20-20070123/)

**<a id="structures-1.1"></a>XSD 1.1 Part 1: Structures**
: <a id="structures-1.1"></a> World Wide Web Consortium. *W3C XML Schema Definition Language (XSD) 1.1 Part 1: Structures*, ed. Shudi (Sandy) Gao 高殊镝, C. M. Sperberg-McQueen, and Henry S. Thompson. W3C Working Draft 3 December 2009. Available at: [http://www.w3.org/TR/xmlschema11-1/](https://www.w3.org/TR/xmlschema11-1/)

## <a id="acknowledgments"></a>C Acknowledgements (non-normative)

<a id="acknowledgments"></a>This document was prepared by the W3C XML Schema Working Group. The members at the time of publication were:

- <a id="acknowledgments"></a>Gioele Barabucci, University of Bologna
- <a id="acknowledgments"></a>Paul V. Biron, Invited expert
- <a id="acknowledgments"></a>David Ezell, National Association of Convenience Stores (NACS) (*chair*)
- <a id="acknowledgments"></a>Shudi (Sandy) Gao 高殊镝, IBM
- <a id="acknowledgments"></a>Mary Holstege, Mark Logic
- <a id="acknowledgments"></a>Sam Idicula, Oracle
- <a id="acknowledgments"></a>Michael Kay, Invited expert
- <a id="acknowledgments"></a>Nan Ma, China Electronics Standardization Institute
- <a id="acknowledgments"></a>Paolo Marinelli, University of Bologna
- <a id="acknowledgments"></a>Jim Melton, Oracle
- <a id="acknowledgments"></a>Noah Mendelsohn, Invited expert
- <a id="acknowledgments"></a>Dave Peterson, Invited expert
- <a id="acknowledgments"></a>Liam Quin, W3C
- <a id="acknowledgments"></a>C. M. Sperberg-McQueen, Black Mesa Technologies (for W3C) (*staff contact*)
- <a id="acknowledgments"></a>Henry S. Thompson, University of Edinburgh
- <a id="acknowledgments"></a>Scott Tsao, The Boeing Company
- <a id="acknowledgments"></a>Fabio Vitali, University of Bologna
- <a id="acknowledgments"></a>Stefano Zacchiroli, University of Bologna
- <a id="acknowledgments"></a>Kongyi Zhou, Oracle
<a id="acknowledgments"></a>
## <a id="acknowledgments"></a><a id="to-do"></a>D To-do list (non-normative)

<a id="to-do"></a>Some changes are expected to be made in future work on this document:

- <a id="to-do"></a>Draft fuller introduction.
- <a id="to-do"></a>Add section on conformance to this spec.
- <a id="to-do"></a>Add section claiming conformance to XSD 1.1 and pointing to the required information.
<a id="to-do"></a>