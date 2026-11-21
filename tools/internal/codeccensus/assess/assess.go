// Package assess is the half of the codec census (package codeccensus) that
// runs inside its driver: the program the census compiles, in a temporary
// module, beside the packages codegen emitted, so that codec.Decode and
// codec.Append can be instantiated with each generated type. Its importers
// are that driver's generated main package, which calls [Drive] or
// [DriveEncode] with one [CodecFor] Codec per type, and package codeccensus,
// which calls [Assemble] and [DocumentElement] and writes the [Request] each
// reads. Package codeccensus owns the population, the assembly and the
// drivers, and `go tool decodecensus` and `go tool encodecensus` each own
// their report; this package owns how one instance, and one round trip, is
// classified, so that the tests that pin each class run the classifier the
// driver runs.
package assess

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/codec"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// Assemble assembles the schema of one test group from docs, its
// <schemaDocument> list as slash paths relative to the suite at root, with
// backend: parser.ParseReport of the first document through loader.Dir over
// that document's own directory, under its base name, then, when a further
// document is one that closure did not read, parser.ParseSet of the first and
// each such further one, each named by its path relative to the first's
// directory. A further document outside that directory's tree does not
// resolve, and fails the assembly.
func Assemble(root string, docs []string, backend value.Backend) (*xsd.Schema, error) {
	if len(docs) == 0 {
		return nil, errors.New("assess: no schema document to assemble")
	}
	first := filepath.Join(root, filepath.FromSlash(docs[0]))
	dir := filepath.Dir(first)
	resolver := loader.Dir(dir)
	location := filepath.Base(first)
	schema, report, err := parser.ParseReport(location, parser.WithResolver(resolver), parser.WithBackend(backend))
	if err != nil {
		return nil, err
	}
	roots := []parser.Root{parser.RootAt(location)}
	for _, doc := range docs[1:] {
		rel, err := filepath.Rel(dir, filepath.Join(root, filepath.FromSlash(doc)))
		if err != nil {
			return nil, fmt.Errorf("assess: relating %s to %s: %w", doc, dir, err)
		}
		extra := filepath.ToSlash(rel)
		reached, err := closureReached(resolver, report, extra)
		if err != nil {
			return nil, err
		}
		if !reached {
			roots = append(roots, parser.RootAt(extra))
		}
	}
	if len(roots) == 1 {
		return schema, nil
	}
	schema, _, err = parser.ParseSet(roots, parser.WithResolver(resolver), parser.WithBackend(backend))
	return schema, err
}

// closureReached reports whether the document resolver resolves location to
// is one the assembly report read.
func closureReached(resolver loader.Resolver, report *parser.AssemblyReport, location string) (bool, error) {
	rc, resolved, err := resolver.Resolve("", location)
	if err != nil {
		return false, fmt.Errorf("assess: resolving schema document %s: %w", location, err)
	}
	if err := rc.Close(); err != nil {
		return false, fmt.Errorf("assess: closing schema document %s: %w", location, err)
	}
	for _, d := range report.Documents() {
		if d.Location == resolved {
			return true, nil
		}
	}
	return false, nil
}

// A Codec is one Go type's two directions, as [CodecFor] makes it: decode
// decodes the instance data against schema, compiled with backend, into that
// type and returns codec.Decode's *T, as an any, and its error, and encode
// appends a value decode returned to no bytes as the document element name
// names and returns codec.Append's result. A test makes a Codec of other
// functions to drive a class no codec answer reaches.
type Codec struct {
	decode func(schema *xsd.Schema, backend value.Backend, data []byte, opts ...codec.Option) (any, error)
	encode func(schema *xsd.Schema, backend value.Backend, name xsd.QName, v any) ([]byte, error)
}

// CodecFor returns the Codec of T: codec.Decode[T] under bound and then the
// options each call passes, and codec.Append[T] under bound of the T a *T
// decode returned points to. The census binds every codec.WithBranches set of
// T's generated package.
func CodecFor[T any](bound ...codec.Option) Codec {
	return Codec{
		decode: func(schema *xsd.Schema, backend value.Backend, data []byte, opts ...codec.Option) (any, error) {
			v, err := codec.Decode[T](schema, backend, data, append(slices.Clone(bound), opts...)...)
			if err != nil {
				return nil, err
			}
			return v, nil
		},
		encode: func(schema *xsd.Schema, backend value.Backend, name xsd.QName, v any) ([]byte, error) {
			return codec.Append(nil, schema, backend, name, *v.(*T), bound...)
		},
	}
}

// placeholder is the T a case decodes into when the census resolved no
// generated type for its document element ([Case]).
type placeholder struct{}

// bucket is the class of one Decode result. The zero value is invalid (STYLE
// T1).
type bucket uint8

const (
	// decoded: Decode returned a value.
	decoded bucket = iota + 1
	// declined: an error errors.As retrieves a *codec.DeclineError from,
	// Decode's doc items 5 and 7.
	declined
	// undecided: an error errors.As retrieves a *codec.UndecidedError from,
	// Decode's doc item 4.
	undecided
	// rejected: an error errors.Is reports as codec.ErrTypeMismatch, Decode's
	// doc item 6, on the type codegen emitted for the document element's
	// declaration.
	rejected
	// placeholderMismatch: an error errors.Is reports as
	// codec.ErrTypeMismatch on the placeholder, counted under failed other.
	placeholderMismatch
	// failedOther: every other error.
	failedOther
)

// outcome is one case's classification: its bucket, the construct of a
// declined one, and whether Decode returned a value where the census's own
// validation did not accept the instance.
type outcome struct {
	bucket       bucket
	construct    codec.Construct
	differential bool
}

// classify returns the bucket of err, a Codec's decode error, and for a
// decline its construct. own reports whether the Codec's T is the type codegen
// emitted for the document element's declaration: an ErrTypeMismatch on any
// other T is no codec defect.
func classify(err error, own bool) (bucket, codec.Construct) {
	if err == nil {
		return decoded, codec.Construct{}
	}
	var de *codec.DeclineError
	if errors.As(err, &de) {
		return declined, de.Construct()
	}
	if errors.Is(err, codec.ErrTypeMismatch) {
		if !own {
			return placeholderMismatch, codec.Construct{}
		}
		return rejected, codec.Construct{}
	}
	var ue *codec.UndecidedError
	if errors.As(err, &ue) {
		return undecided, codec.Construct{}
	}
	return failedOther, codec.Construct{}
}

// accepts reports whether validation accepts data against schema: it is
// accepted when validate.New and xmlsrc.Validate succeed and the Result has
// no source fault and stands validate.NotProvenInvalid.
func accepts(schema *xsd.Schema, backend value.Backend, data []byte, uri string) bool {
	v, err := validate.New(schema, backend)
	if err != nil {
		return false
	}
	res, err := xmlsrc.Validate(v, bytes.NewReader(data), xmlsrc.WithURI(uri))
	if err != nil || res.Err() != nil {
		return false
	}
	return res.Standing() == validate.NotProvenInvalid
}

// assess classifies c's decode of the instance data, named uri: its bucket
// by classify, and differential when decode returned no error where the
// census's own validation, never decode's error, does not accept data; it
// returns the decoded value beside the outcome, nil unless the bucket is
// decoded.
func assess(schema *xsd.Schema, backend value.Backend, data []byte, uri string, c Codec, own bool) (outcome, any) {
	v, err := c.decode(schema, backend, data, codec.WithURI(uri))
	b, construct := classify(err, own)
	return outcome{bucket: b, construct: construct, differential: b == decoded && !accepts(schema, backend, data, uri)}, v
}

// A Request is what [Drive] reads: the suite root every path is relative to,
// and the test groups whose cases it assesses.
type Request struct {
	// Root is the suite root, absolute.
	Root string
	// Groups are the test groups, each with the schema its cases share.
	Groups []Group
}

// A Group is one test group's schema documents, assembled by [Assemble], and
// the cases assessed against that schema.
type Group struct {
	Schema []string
	Cases  []Case
}

// A Case is one instance case: its ID, the conformance.CatalogEntry's, its
// instance document, a slash path relative to the suite root, and the index
// into the driver's codecs of the Codec of the type codegen emitted for its
// document element's declaration, or -1 for the placeholder, which an
// ErrTypeMismatch is no codec defect for. ID is exported, as every field
// here is, because the Request crosses the driver's process boundary as
// JSON: the census writes it and the driver copies it into its summary,
// since an instance document alone names no case (two cases can share one).
type Case struct {
	ID       string
	Instance string
	Codec    int
}

// A Summary is what [Drive] writes. Decoded, Undecided and FailedOther are
// counts; Rejected, Differential, PlaceholderMismatch and each Tally's Cases
// hold the IDs of their cases, in the order Drive met them, and the count of
// each is its list's length (STYLE D3), so that the census's -paths can name
// the cases behind it. Every field is exported for the JSON the census reads.
type Summary struct {
	Decoded     int
	Undecided   int
	FailedOther int
	// Rejected holds the cases whose Decode returned ErrTypeMismatch on the
	// type codegen emitted for the document element's declaration.
	Rejected []string
	// Differential holds the decoded cases the census's validation did not
	// accept.
	Differential []string
	// PlaceholderMismatch holds the cases whose Decode returned
	// ErrTypeMismatch on the placeholder; FailedOther counts them too.
	PlaceholderMismatch []string
	// Declines holds one Tally per declined construct, in the order the
	// cases first met each; the declined count is the sum of their Cases'
	// lengths.
	Declines []Tally
}

// A Tally is one declined construct, by its codec.Construct's String, and
// the cases that declined it first.
type Tally struct {
	Construct string
	Cases     []string
}

// Drive reads one JSON [Request] from r, assesses each of its cases with the
// Codec codecs holds at the case's index, or the placeholder's, and writes
// one JSON [Summary] to w. Within the one process it keys each decline on
// its codec.Construct, and writes the Construct's String, which codec makes
// distinct for each Construct, only into the Summary; [DriveEncode] keys its
// own declines the same way. A group that does not assemble, or an instance
// that does not read, is an error: the census generated each group's package
// from the same assembly, and read each instance's document element, before
// it wrote the Request.
func Drive(r io.Reader, w io.Writer, codecs []Codec) error {
	var sum Summary
	// index maps a construct to its position in sum.Declines, for lookup
	// only; nothing ranges over it (STYLE D2).
	index := map[codec.Construct]int{}
	err := walk(r, codecs, func(schema *xsd.Schema, backend value.Backend, data []byte, c Case, k Codec, own bool) {
		o, _ := assess(schema, backend, data, c.Instance, k, own)
		tallyOutcome(&sum, index, c.ID, o)
	})
	if err != nil {
		return err
	}
	return writeSummary(w, sum)
}

// walk reads one JSON Request from r and calls visit on each of its cases,
// in request order, with its group's schema, the backend that schema was
// assembled with, the instance's bytes, and the Codec codecs holds at the
// case's index, own true, or the placeholder's, own false. It returns the
// errors [Drive]'s doc states.
func walk(r io.Reader, codecs []Codec, visit func(schema *xsd.Schema, backend value.Backend, data []byte, c Case, k Codec, own bool)) error {
	var req Request
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return fmt.Errorf("assess: reading the request: %w", err)
	}
	backend := strict.New()
	for _, g := range req.Groups {
		schema, err := Assemble(req.Root, g.Schema, backend)
		if err != nil {
			return fmt.Errorf("assess: assembling %v: %w", g.Schema, err)
		}
		for _, c := range g.Cases {
			data, err := os.ReadFile(filepath.Join(req.Root, filepath.FromSlash(c.Instance)))
			if err != nil {
				return fmt.Errorf("assess: reading instance %s: %w", c.Instance, err)
			}
			k, own, err := codecAt(codecs, c.Codec)
			if err != nil {
				return fmt.Errorf("assess: case %s: %w", c.Instance, err)
			}
			visit(schema, backend, data, c, k, own)
		}
	}
	return nil
}

// writeSummary writes sum, a driver's summary, to w as one JSON value.
func writeSummary(w io.Writer, sum any) error {
	if err := json.NewEncoder(w).Encode(sum); err != nil {
		return fmt.Errorf("assess: writing the summary: %w", err)
	}
	return nil
}

// codecAt returns codecs[i] and true, or for i -1 the placeholder's Codec
// and false.
func codecAt(codecs []Codec, i int) (Codec, bool, error) {
	if i == -1 {
		return CodecFor[placeholder](), false, nil
	}
	if i < 0 || i >= len(codecs) {
		return Codec{}, false, fmt.Errorf("codec index %d is outside the %d codecs", i, len(codecs))
	}
	return codecs[i], true, nil
}

// tallyOutcome adds o, the outcome of the case id, to sum, index mapping a
// construct to its Tally's position in sum.Declines.
func tallyOutcome(sum *Summary, index map[codec.Construct]int, id string, o outcome) {
	if o.differential {
		sum.Differential = append(sum.Differential, id)
	}
	switch o.bucket {
	case decoded:
		sum.Decoded++
	case declined:
		sum.Declines = addDecline(sum.Declines, index, o.construct, id)
	case undecided:
		sum.Undecided++
	case rejected:
		sum.Rejected = append(sum.Rejected, id)
	case placeholderMismatch:
		sum.FailedOther++
		sum.PlaceholderMismatch = append(sum.PlaceholderMismatch, id)
	case failedOther:
		sum.FailedOther++
	}
}

// addDecline adds the case id, declining construct, to declines and returns
// them, index mapping a construct to its Tally's position in declines.
func addDecline(declines []Tally, index map[codec.Construct]int, construct codec.Construct, id string) []Tally {
	i, seen := index[construct]
	if !seen {
		i = len(declines)
		index[construct] = i
		declines = append(declines, Tally{Construct: construct.String()})
	}
	declines[i].Cases = append(declines[i].Cases, id)
	return declines
}
