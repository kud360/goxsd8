package xmlsrc

import (
	"errors"
	"fmt"
	"io"

	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsderr"
)

// Option configures [Validate]. The zero set of options is a complete,
// usable configuration: options only replace defaults (STYLE T1).
type Option func(*config)

// config is the resolved [Validate] configuration.
type config struct {
	uri string
}

// newConfig applies opts over the defaults: the document is unnamed, and
// every Loc it produces renders "?:line:col".
func newConfig(opts []Option) config {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithURI names the document being read, so every [xsderr.Loc] the
// assessment cites identifies the file a reader has to open. [Validate]
// takes an [io.Reader], which carries no name of its own.
func WithURI(uri string) Option {
	return func(c *config) { c.uri = uri }
}

// Validate assesses the XML instance in r against v's schema and reports
// what the assessment found.
//
// The two error channels split on whether the assessment ran at all: it
// returns (nil, err) only when it never did — v or r is nil, or the
// document is malformed before its document element — and (result, nil) in
// every case where the walk began, with a source fault that stopped the
// walk mid-document living in [validate.Result.Err] alone and never also
// returned here.
//
// A nil argument yields a plain error rather than an [xsderr.Error], on
// [validate.New]'s reasoning about its own nil schema: it is a caller's
// bug, not a verdict about a document. A document that is malformed, or
// holds no document element, yields the reader's own *[xsderr.Error].
func Validate(v *validate.Validator, r io.Reader, opts ...Option) (*validate.Result, error) {
	if v == nil {
		return nil, fmt.Errorf("xmlsrc: Validate: nil *validate.Validator")
	}
	if r == nil {
		return nil, fmt.Errorf("xmlsrc: Validate: nil io.Reader")
	}
	cfg := newConfig(opts)
	w := newWalker(cfg.uri, r)
	root, err := w.root()
	if err != nil {
		return nil, err
	}
	// GAP(xml): content OUTSIDE the document element is not inspected.
	// Character data before it is dropped (see root), and anything after its
	// end tag is never read: Assess returns there, so trailing character
	// content and a second document element alike go unreported. XML 1.0
	// §2.1 admits only Misc in either position. Tracked by #753.
	return v.Assess(root), nil
}

// walker is the one token stream a whole assessment pulls from: a single
// xmltree.Reader, the depth it currently stands at, and the one source
// fault that can stop it.
type walker struct {
	r *xmltree.Reader
	// uri names the document, for the one location the reader cannot
	// supply itself (see root); the reader keeps its own copy unexported.
	uri string
	// depth is the number of elements open at the stream's current
	// position; 0 is the document level.
	depth int
	// n counts the tokens pulled so far, so an element can tell whether the
	// stream still stands where it was yielded (see element.Children).
	n int
	// err is the walk's single latched source fault. Every cursor reports
	// this one field and none keeps a copy of it (STYLE D3).
	err error
}

func newWalker(uri string, r io.Reader) *walker {
	return &walker{r: xmltree.NewReader(uri, r), uri: uri}
}

// next pulls the next node and reports at, the depth of the element whose
// [[children]] the node belongs to: for a start or end tag, the depth of the
// element containing the one it opens or closes; for character data, the
// depth of the element containing the run.
func (w *walker) next() (xmltree.Node, int, error) {
	node, err := w.r.Token()
	if err != nil {
		return nil, 0, err
	}
	w.n++
	switch node.(type) {
	case *xmltree.StartElement:
		w.depth++
		return node, w.depth - 1, nil
	case *xmltree.EndElement:
		w.depth--
		return node, w.depth, nil
	}
	return node, w.depth, nil
}

// root advances to the document element. Character data at the document
// level belongs to no element, so it is dropped rather than yielded to one
// — whatever it holds: nothing here asks whether it is the whitespace XML
// 1.0 §2.1 allows there, which is the GAP(xml) Validate marks.
func (w *walker) root() (*element, error) {
	for {
		node, at, err := w.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, xsderr.New(xsderr.RuleXMLWellFormed, xsderr.Loc{URI: w.uri}, "document has no document element")
			}
			return nil, err
		}
		start, ok := node.(*xmltree.StartElement)
		if !ok {
			continue
		}
		return &element{w: w, start: start, depth: at + 1, n: w.n}, nil
	}
}
