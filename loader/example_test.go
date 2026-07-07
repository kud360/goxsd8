package loader_test

import (
	"fmt"
	"io"
	"testing/fstest"

	"github.com/kud360/goxsd8/loader"
)

// Example_chain composes three resolvers with Chain: an fs.FS (e.g. an
// embed.FS of bundled schemas), a directory on disk, and an in-memory map.
// The first resolver that holds the location wins; the others are consulted
// only on ErrNotFound. Here the fs.FS lacks "xhtml.xsd" so the chain falls
// through to the Map.
func Example_chain() {
	bundled := loader.FS(fstest.MapFS{
		"xml.xsd": {Data: []byte("<!-- bundled xml.xsd -->")},
	})
	local := loader.Dir("/etc/xml/schemas") // may not exist; ErrNotFound falls through
	overrides := loader.Map(map[string]string{
		"xhtml.xsd": "<!-- in-memory xhtml.xsd -->",
	})

	resolver := loader.Chain(bundled, local, overrides)

	rc, resolved, err := resolver.Resolve("http://www.w3.org/1999/xhtml", "xhtml.xsd")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = rc.Close() }()

	body, _ := io.ReadAll(rc)
	fmt.Printf("resolved %q:\n%s\n", resolved, body)

	// Output:
	// resolved "xhtml.xsd":
	// <!-- in-memory xhtml.xsd -->
}
