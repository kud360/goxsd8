package xmltree_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/parser/xmltree"
)

// FuzzReader drives the reader over arbitrary input. Malformed XML must be an
// error value, never a panic (PRINCIPLES 24) — go test surfaces any panic as
// a failure, so the target only needs to keep draining until EOF or error.
func FuzzReader(f *testing.F) {
	seeds := []string{
		"<a/>",
		"<a>\n  <b>x</b>\n</a>",
		`<a xmlns:p="urn:P"><p:b p:x="1"/></a>`,
		`<r xmlns="urn:D"><c xml:lang="en"/></r>`,
		"<a><u:b/></a>",
		"<a></b>",
		"<a>\r\n<b/>\r\n</a>",
		"not xml at all",
		"<!-- c --><?pi ?><a>&amp;</a>",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, doc string) {
		r := xmltree.NewReader("fuzz.xml", strings.NewReader(doc))
		// Bound the loop: each Token consumes input, so a well-behaved
		// reader always terminates, but a bug must not hang the fuzzer.
		for i := 0; i < 1<<20; i++ {
			_, err := r.Token()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					// A located error is the contract; assert it does not
					// panic when rendered.
					_ = err.Error()
				}
				return
			}
		}
	})
}
