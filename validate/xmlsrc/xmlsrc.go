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

// newConfig applies opts over the defaults: the document is unnamed, so
// every Loc it produces renders with "?" for its URI.
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
// The two error channels split on whether an assessment stands: it returns
// (nil, err) when none does — v or r is nil, the document is malformed
// before its document element, or the walk finished and the rest of the
// stream, which Validate then reads to its end, is malformed: a subtree the
// walk did not descend into, or what follows the document element, where
// only Misc may stand (XML 1.0 [1] document, [27] Misc), so that character
// data that is not literal white space and a second top-level element there are
// reported. A document that is not well-formed has no infoset to assess. It
// returns (result, nil) in every other case, with a source fault that stopped
// the walk mid-document living in [validate.Result.Err] alone and never also
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
	res := v.Assess(root)
	// A walk that stopped on a fault has it in res.Err already, and the
	// stream it stopped in is not read on.
	if w.err == nil {
		err = w.drain()
	}
	if err != nil {
		return nil, err
	}
	return res, nil
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
// level belongs to no element, so it is dropped rather than yielded to one;
// the reader has already refused any such run that is not S (XML 1.0 [1]
// document, [22] prolog, [27] Misc), so nothing the document holds is lost.
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

// drain reads the stream from wherever the walk left it to its end and returns
// the first source fault it meets: in a subtree the walk did not descend into,
// or after the document element's end tag, where the reader refuses all but
// Misc (XML 1.0 [1] document, [27] Misc) — character data that is not S, or a
// second top-level element.
func (w *walker) drain() error {
	for {
		_, _, err := w.next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
