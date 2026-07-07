package loader

import (
	"fmt"
	"io"
	"net/http"
)

// httpResolver dereferences locations as HTTP(S) URLs; see HTTP.
type httpResolver struct {
	client *http.Client
}

// HTTP returns a Resolver that dereferences each location as an absolute
// URL over GET.
//
// location is used verbatim as the request URL, built with
// http.NewRequest(http.MethodGet, location, nil); there is no implicit base
// resolution, because resolving a relative reference against a base URI is
// the caller's job (§4.3.2 clause 4), not this seam's. Timeout, redirect,
// and transport behavior are wholly delegated to client — HTTP adds no
// package-level timeout logic (D5/no-hidden-state). A nil client falls back
// to http.DefaultClient.
//
// Status mapping (engineering decision, not spec-derived): a 404 maps to
// ErrNotFound, since only 404 unambiguously means "not found" the way Chain
// fall-through requires; every other non-2xx status is a real, wrapped
// error that short-circuits a Chain rather than being treated as absence.
func HTTP(client *http.Client) Resolver {
	return httpResolver{client: client}
}

// Resolve implements Resolver.
func (r httpResolver) Resolve(namespace, location string) (io.ReadCloser, string, error) {
	client := r.client
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest(http.MethodGet, location, nil)
	if err != nil {
		return nil, "", fmt.Errorf("loader: building request for %q: %w", location, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("loader: fetching %q: %w", location, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		// Body is discarded on the error path; its Close error cannot matter.
		_ = resp.Body.Close()
		return nil, "", fmt.Errorf("loader: %q returned 404: %w", location, ErrNotFound)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Body is discarded on the error path; its Close error cannot matter.
		_ = resp.Body.Close()
		return nil, "", fmt.Errorf("loader: %q returned status %q", location, resp.Status)
	}

	// resp.Body streams the response; the caller closes it (P4 — no ReadAll).
	return resp.Body, location, nil
}
