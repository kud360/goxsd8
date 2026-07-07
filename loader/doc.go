// Package loader is the IO seam: every schema document the parser reads
// arrives through a Resolver, so multi-schema composition, catalogs, and
// instance schemaLocation hints all have one home.
//
// # Contract (implemented in M2)
//
//	type Resolver interface {
//	    Resolve(namespace, location string) (io.ReadCloser, string, error)
//	}
//	    Answers "give me the schema document for (target namespace,
//	    location hint)". The returned string is the RESOLVED location —
//	    the dedup key: the loader loads each resolved location once, so
//	    a document named by several imports/hints composes instead of
//	    duplicating.
//
//	type ResolverFunc func(namespace, location string) (io.ReadCloser, string, error)
//	    Adapter, in the http.HandlerFunc idiom.
//
//	var ErrNotFound error
//	    Sentinel a Resolver returns when it has no answer; Chain uses
//	    errors.Is on it to fall through.
//
//	func Dir(path string) Resolver     // relative locations under a root
//	func FS(fsys fs.FS) Resolver       // any fs.FS (embed, testing)
//	func HTTP(client *http.Client) Resolver
//	func Map(docs map[string]string) Resolver  // in-memory, for tests/tools
//	func Chain(rs ...Resolver) Resolver
//	    First resolver that doesn't return ErrNotFound wins; attempts
//	    are aggregated with errors.Join when all fail.
//
// Multiple root schemas load into one set (the CLI accepts several
// schema arguments); xsi:schemaLocation / xsi:noNamespaceSchemaLocation
// hints found in instances route through the SAME Resolver, resolved
// relative to the instance document's location — hint loading and root
// loading must never diverge.
package loader
