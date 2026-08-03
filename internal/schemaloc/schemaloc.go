package schemaloc

import (
	"net/url"
	"path"
	"strings"
)

// Resolve resolves a schemaLocation URI reference against the base URI in scope
// at the element carrying it (§4.3.2 clause 4). The result is what the resolver
// is asked for, and — for a document that has its own <include>s — the base those
// are in turn resolved against, so it must stay in the same space as the location
// the caller handed the parser.
//
// Clause 4 fixes only WHICH base URI to resolve against; it mandates no
// resolution algorithm, so the split below is goxsd8's own strategy for honoring
// it. An absolute reference wins outright. Otherwise, when the base is itself an
// absolute URI, standard RFC 3986 reference resolution applies. When it is not —
// a bare relative path such as "schemas/main.xsd", which is exactly what a
// resolver rooted at a directory or an in-memory map is keyed by —
// [net/url.URL.ResolveReference] would root the result at "/" and turn a
// resolver-relative location into an absolute one that no such resolver can
// serve; path-wise resolution against the base's directory is used instead.
func Resolve(base, location string) string {
	if location == "" {
		return base
	}
	ref, refErr := url.Parse(location)
	if refErr == nil && ref.IsAbs() {
		return location
	}
	b, baseErr := url.Parse(base)
	if refErr == nil && baseErr == nil && b.IsAbs() {
		return b.ResolveReference(ref).String()
	}
	if strings.HasPrefix(location, "/") {
		return path.Clean(location)
	}
	return path.Join(path.Dir(base), location)
}
