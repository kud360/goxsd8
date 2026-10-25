package goxsd8

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadmeFixtures holds each file of testdata/readme that README.md shows
// to the text README shows for it: order.xsd and order.json byte for byte as
// a fence's body, and the one-line twice.json, its newline dropped, as an
// inline code span. README's examples run against the committed files, so a
// fence edited without its file, or a file without its fence, would show the
// reader one document and run another.
func TestReadmeFixtures(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		file string
		want func(body string) string
	}{
		{"order.xsd", func(body string) string { return "```xml\n" + body + "```\n" }},
		{"order.json", func(body string) string { return "```json\n" + body + "```\n" }},
		{"twice.json", func(body string) string { return "`" + strings.TrimSuffix(body, "\n") + "`" }},
	} {
		body, err := os.ReadFile(filepath.Join("testdata", "readme", c.file))
		if err != nil {
			t.Errorf("%s: %v", c.file, err)
			continue
		}
		if want := c.want(string(body)); !strings.Contains(string(readme), want) {
			t.Errorf("README.md does not show testdata/readme/%s as\n%s", c.file, want)
		}
	}
}
