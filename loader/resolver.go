package loader

import (
	"errors"
	"fmt"
	"io"
)

// ErrNotFound is the sentinel a Resolver returns when it has no answer for
// a (namespace, location) request. Chain uses errors.Is on it to fall
// through to the next resolver, and callers (the M4 parser) use it to
// decide that an unresolved schemaLocation hint is a normal, non-fatal
// outcome per §4.2.6.2 (src-import) and §4.3.2 clause 3 — this package
// does not decide fatality.
var ErrNotFound = errors.New("loader: schema document not found")

// Resolver answers "give me the schema document for (target namespace,
// location hint)". The returned string is the RESOLVED location — the
// dedup key: the loader loads each resolved location once, so a document
// named by several imports/hints composes instead of duplicating.
//
// The empty string is the well-defined "no namespace" state (absence of
// targetNamespace, §4.2.6.2 / §4.3.2), so namespace == "" is the sentinel
// for the no-namespace case, not a missing argument.
//
// The location argument is ALREADY RESOLVED against the owner element's
// base URI: §4.3.2 clause 4 makes relative-reference resolution the
// caller's (parser's) job, so Resolve is location-hint-in / reader-out and
// performs no URI-relative resolution of its own.
//
// There is deliberately no context.Context parameter: library code here
// runs no concurrency of its own (STYLE D5/L6), and callers bound blocking
// behavior through the injected dependency — for HTTP, the *http.Client's
// Transport timeouts; a caller needing cancellation wraps Resolve in its
// own goroutine, which is its business, not this seam's.
type Resolver interface {
	Resolve(namespace, location string) (io.ReadCloser, string, error)
}

// ResolverFunc adapts an ordinary function to Resolver, in the
// http.HandlerFunc idiom.
type ResolverFunc func(namespace, location string) (io.ReadCloser, string, error)

// Resolve calls f.
func (f ResolverFunc) Resolve(namespace, location string) (io.ReadCloser, string, error) {
	return f(namespace, location)
}

// chainResolver tries its resolvers in slice order; see Chain.
type chainResolver struct {
	resolvers []Resolver
}

// Chain returns a Resolver that tries rs in order. The first resolver that
// does not return ErrNotFound wins — its reader, resolved location, or real
// error is returned immediately.
//
// Only ErrNotFound triggers fall-through: a real (non-ErrNotFound) I/O
// error from any resolver short-circuits the chain and is returned as-is,
// so a permission-denied or malformed-response failure is never masked by a
// later resolver that happens to hold a copy. When every resolver returns
// ErrNotFound the per-resolver failures are aggregated with errors.Join
// (which still satisfies errors.Is(err, ErrNotFound)); an empty chain
// resolves nothing and returns ErrNotFound.
func Chain(rs ...Resolver) Resolver {
	return chainResolver{resolvers: rs}
}

// Resolve implements Resolver.
func (c chainResolver) Resolve(namespace, location string) (io.ReadCloser, string, error) {
	errs := make([]error, 0, len(c.resolvers))
	for _, r := range c.resolvers {
		rc, resolved, err := r.Resolve(namespace, location)
		if err == nil {
			return rc, resolved, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, "", err
		}
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return nil, "", fmt.Errorf("loader: resolving %q in namespace %q: %w", location, namespace, ErrNotFound)
	}
	return nil, "", errors.Join(errs...)
}
