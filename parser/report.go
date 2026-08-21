package parser

import (
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// AssemblyReport is what [ParseReport] observed while assembling one schema:
// every schema document it read, in discovery order, and every
// ·inter-schema-document reference· — <include>, <redefine>, <override>,
// <import> (§4.2.1) — it could not follow to one of them.
//
// It exists because §4.2.1's schema(D) is a property of a DOCUMENT SET, not of
// the components the set yields: the assembled [xsd.Schema] carries components
// with no record of which document each came from, and a consumer that needs to
// reason about the documents themselves (the conformance harness gates every
// document of a closure on its own decidable-shape allowlist) would otherwise
// have to re-walk §4.2's composition edges independently — a second
// implementation of this package's own discovery, which is exactly what it
// replaces.
//
// A report is always returned, and is populated as far as assembly got even
// when [ParseReport] returns an error: a report of the documents read BEFORE a
// composition rule rejected the assembly is precisely what makes the failure
// attributable.
//
// Its fields are unexported and it is constructed only by this package (STYLE
// T1): a caller-built report would claim an assembly that never happened.
type AssemblyReport struct {
	documents  []AssembledDocument
	unfollowed []UnfollowedDirective
}

// Documents returns the schema documents the assembly read, in DISCOVERY order:
// depth-first, pre-order, over each <schema>'s <include>, <override> and
// <import> children in one document-order pass — the same order their
// components enter the builder (STYLE D2; assembly.docs, never rebuilt from the
// load-once index).
//
// The same *[Document] can appear more than once. Document identity in an
// assembly is the resolved location together with the namespace it was reached
// under and the ·override pre-processing· applied to it (docKey), so one
// document reached both as a chameleon <include> and as a bare <import>, or
// overridden two different ways, is two distinct discoveries contributing two
// distinct component sets (§4.2.5's "duplicate and conflicting versions of some
// components"). This is a list of readings, not a set of files.
//
// The slice is the report's own; treat it as read-only.
func (r *AssemblyReport) Documents() []AssembledDocument { return r.documents }

// Unfollowed returns the ·inter-schema-document references· the assembly did
// not follow to a document, in the order they were encountered.
//
// Most entries are NOT errors: §4.2.3 clause 2.4 says of <include> that "it is
// not an error for the ·actual value· of the schemaLocation [attribute] to fail
// to resolve at all, in which case the corresponding inclusion must not be
// performed", §4.2.6.2 says as much of <import> and makes its schemaLocation a
// hint in the first place. They are reported because the resulting assembly is
// SHORT by whatever the unread document would have contributed, which a
// consumer cannot otherwise observe: a reference that named no document leaves
// no trace among [AssemblyReport.Documents] by construction.
//
// A NON-EMPTY <xs:redefine> is the one directive for which an unresolved
// location is also an error (src-redefine clause 1, §4.2.4): it is recorded here
// too, and [ParseReport] returns the verdict alongside. An EMPTY one keeps
// <include>'s non-error skip, since clause 1's antecedent does not fire for it.
//
// It records references the assembly ATTEMPTED and came back empty from, which
// is narrower than "every reference the document set contains":
// [ParseReport] stops at its first error, so directives it never reached are
// neither followed nor reported.
//
// The slice is the report's own; treat it as read-only.
func (r *AssemblyReport) Unfollowed() []UnfollowedDirective { return r.unfollowed }

// AssembledDocument is one schema document read while assembling a schema —
// one DISCOVERY of it, in the sense [AssemblyReport.Documents] describes.
type AssembledDocument struct {
	// Doc is the document as read.
	Doc *Document

	// Location is the RESOLVED location the [loader.Resolver] reported for it —
	// the load-once identity the Resolver contract defines, canonicalized as that
	// resolver sees fit. It is deliberately not Doc.URI(), which is the location
	// as REQUESTED (a schemaLocation resolved against its directive's base URI,
	// §4.3.2 clause 4), and not the schemaLocation attribute's own value either.
	// Compare locations through this field, exactly as the assembly's own dedup
	// index does.
	Location string
}

// UnfollowedDirective is one ·inter-schema-document reference· that yielded no
// assembled document, with the reason it yielded none and the position of the
// directive element itself.
type UnfollowedDirective struct {
	// Reason is why the directive was not followed.
	Reason UnfollowedReason

	// At is the location of the <include>/<redefine>/<override>/<import> element,
	// so a consumer can report the directive rather than the document.
	At xsderr.Loc
}

// UnfollowedReason is why an ·inter-schema-document reference· yielded no
// document. It is a closed set (STYLE T1, the xsd/closedsets.go idiom): a
// uint8 whose zero value is invalid, so an unset Reason is a caught bug rather
// than a valid one.
type UnfollowedReason uint8

// The UnfollowedReason values.
const (
	// UnfollowedLocationUnresolved is a schemaLocation that named no readable
	// document: the [loader.Resolver] reported loader.ErrNotFound for it — the
	// §4.2.3 clause 2.4 / §4.2.6.2 non-error — or failed for a reason of its own
	// (a transport or permission fault), which [ParseReport] also returns as an
	// error. Either way no document came back, so no composition was performed.
	UnfollowedLocationUnresolved UnfollowedReason = iota + 1

	// UnfollowedNoLocation is a bare <import>: a namespace declared with no
	// schemaLocation attribute at all, which §4.2.6.2 makes explicitly legal —
	// it declares that references into the namespace are expected without naming
	// a document to satisfy them from.
	UnfollowedNoLocation

	// UnfollowedNoSchemaLocation is an <include>, <override> or <redefine>
	// carrying no schemaLocation attribute. §4.2.1 makes those schemaLocations
	// mandatory — they "are not hints: conforming processors must attempt to
	// de-reference" — so this is a grammar fault, which [ParseReport] also
	// returns as an error, not a spec-sanctioned skip.
	UnfollowedNoSchemaLocation

	// UnfollowedUnreadable is a schemaLocation that resolved to a document which
	// could not be read: an XML well-formedness fault, an I/O failure, or a
	// document in an encoding this parser does not decode. src-include clause
	// 1.1 and src-import clause 2 both require a well-formed information set, so
	// [ParseReport] returns this as an error too.
	UnfollowedUnreadable
)

// String returns a stable identifier for the reason, or a diagnostic form for
// an invalid value (never panics — it is reached from logging and error
// formatting).
func (r UnfollowedReason) String() string {
	switch r {
	case UnfollowedLocationUnresolved:
		return "location-unresolved"
	case UnfollowedNoLocation:
		return "no-location"
	case UnfollowedNoSchemaLocation:
		return "no-schema-location"
	case UnfollowedUnreadable:
		return "unreadable"
	default:
		return "UnfollowedReason(" + strconv.Itoa(int(r)) + ")"
	}
}
