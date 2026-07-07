package loader

import (
	"fmt"
	"io"
	"strings"
)

// mapResolver serves in-memory documents keyed by location; see Map.
type mapResolver struct {
	docs map[string]string
}

// Map returns a Resolver backed by an in-memory location→content map, for
// tests and tools.
//
// The map key is the location hint (the location argument of Resolve), NOT
// the namespace: a location-addressable store is the natural model for a
// hint→document lookup, and namespace is orthogonal (it is empty for
// no-namespace schemas, so it cannot be the sole key). namespace is ignored
// here. A location absent from the map maps to ErrNotFound.
//
// Map copies docs at construction so later mutation of the caller's map
// cannot change what the Resolver serves.
func Map(docs map[string]string) Resolver {
	m := make(map[string]string, len(docs))
	for k, v := range docs {
		m[k] = v
	}
	return mapResolver{docs: m}
}

// Resolve implements Resolver.
func (r mapResolver) Resolve(namespace, location string) (io.ReadCloser, string, error) {
	content, ok := r.docs[location]
	if !ok {
		return nil, "", fmt.Errorf("loader: %q not in map: %w", location, ErrNotFound)
	}
	return io.NopCloser(strings.NewReader(content)), location, nil
}
