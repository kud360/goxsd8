// Command fetchspecs (re)downloads the pristine spec HTML into
// docs/specs/html. Normally never needed — the HTML is committed. Run
// `go generate ./...` afterwards to regenerate the Markdown.
//
// The F&O and XDM specs are pinned to their dated URIs: the undated
// /TR/xpath-functions/ and /TR/xpath-datamodel/ shortnames have moved on
// to later major versions (3.x), but XPath 2.0 normatively binds to the
// 1.0 (Second Edition) documents. The other undated shortnames still
// resolve to the editions XSD 1.1 §1.4 cites (Namespaces in XML 1.0 Third
// Edition, XML 1.0 Fifth Edition, XML Information Set Second Edition), so
// they stay undated and track future in-place errata.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var specs = []struct {
	file string
	url  string
}{
	{"xmlschema11-1.html", "https://www.w3.org/TR/xmlschema11-1/"},
	{"xmlschema11-2.html", "https://www.w3.org/TR/xmlschema11-2/"},
	{"xpath20.html", "https://www.w3.org/TR/xpath20/"},
	{"xpath-functions.html", "https://www.w3.org/TR/2010/REC-xpath-functions-20101214/"},
	{"xpath-datamodel.html", "https://www.w3.org/TR/2010/REC-xpath-datamodel-20101214/"},
	{"xsd-precisionDecimal.html", "https://www.w3.org/TR/xsd-precisionDecimal/"},
	{"xml-names.html", "https://www.w3.org/TR/xml-names/"},
	{"xml.html", "https://www.w3.org/TR/xml/"},
	{"xml-infoset.html", "https://www.w3.org/TR/xml-infoset/"},
}

func main() {
	dir := filepath.Join("docs", "specs", "html")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "fetchspecs: creating %s: %v\n", dir, err)
		os.Exit(1)
	}
	for _, s := range specs {
		if err := fetch(filepath.Join(dir, s.file), s.url); err != nil {
			fmt.Fprintf(os.Stderr, "fetchspecs: %v\n", err)
			os.Exit(1)
		}
	}
}

func fetch(path, url string) error {
	fmt.Printf("fetching %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	// Read-side close; the body has been fully consumed or the copy failed.
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: HTTP %s", url, resp.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}
	return nil
}
