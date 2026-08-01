package xmltree

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// bomEncoding is the character encoding a document's leading byte-order mark
// signals (XML 1.0 §4.3.3, byte sequences enumerated in Appendix F.1). It is
// the reader's evidence of the entity's actual encoding, against which an
// encoding declaration — if the document carries one — must agree.
type bomEncoding int

const (
	// bomNone is a document with no byte-order mark, read as UTF-8.
	bomNone bomEncoding = iota
	bomUTF8
	bomUTF16BE
	bomUTF16LE
)

// String describes the evidence the mark gave, for error messages naming the
// encoding an entity was actually presented in.
func (e bomEncoding) String() string {
	switch e {
	case bomUTF8:
		return "a UTF-8 byte-order mark"
	case bomUTF16BE:
		return "a UTF-16 big-endian byte-order mark"
	case bomUTF16LE:
		return "a UTF-16 little-endian byte-order mark"
	case bomNone:
		return "no byte-order mark, so UTF-8"
	}
	return "an unrecognized byte-order mark"
}

// agreesWith reports whether the encoding named by an XML declaration's
// encoding pseudo-attribute is the one the byte-order mark signalled. XML 1.0
// §4.3.3 makes a disagreement a fatal error, so this is the whole of that
// rule's name table: EncName values are matched without regard to case, and a
// bare "UTF-16" agrees with a mark of either byte order.
func (e bomEncoding) agreesWith(name string) bool {
	switch {
	case strings.EqualFold(name, "UTF-8"):
		return e == bomNone || e == bomUTF8
	case strings.EqualFold(name, "UTF-16"):
		return e == bomUTF16BE || e == bomUTF16LE
	case strings.EqualFold(name, "UTF-16BE"):
		return e == bomUTF16BE
	case strings.EqualFold(name, "UTF-16LE"):
		return e == bomUTF16LE
	}
	// GAP(xml): every other EncName — legacy single-byte encodings, and the
	// BOM-less UTF-16 form that XML 1.0 §4.3.3 permits an entity to declare
	// without a mark — is undecodable here rather than silently mis-decoded.
	// Only a mark is taken as evidence of an entity's encoding; a declaration
	// standing alone is deliberately not. Tracked by #344.
	return false
}

// charsetReader satisfies xml.Decoder.CharsetReader for a stream this package
// has already decoded to UTF-8. The byte-order mark settled the encoding
// before the declaration was read (XML 1.0 §4.3.3), so a declaration that
// agrees with the mark names bytes the decoder is already being handed and
// needs no further conversion; one that disagrees is that section's fatal
// error, reported through the decoder as a read failure and mapped to
// RuleXMLWellFormed by handleReadErr.
func (e bomEncoding) charsetReader(name string, input io.Reader) (io.Reader, error) {
	if e.agreesWith(name) {
		return input, nil
	}
	return nil, fmt.Errorf("encoding declaration %q disagrees with the entity's actual encoding: %s", name, e)
}

// bomLen is the longest byte-order mark this package recognizes (UTF-8's
// EF BB BF); the UTF-16 marks are two bytes.
const bomLen = 3

// decodeBOM reads r's leading byte-order mark and returns the document body as
// UTF-8 together with the encoding the mark named. A UTF-16 mark (FE FF or
// FF FE) selects a transcoding reader; a UTF-8 mark is dropped, being "an
// encoding signature, not part of either the markup or the character data of
// the XML document" (XML 1.0 §4.3.3); an unmarked stream is passed through.
//
// Peek answers two independent questions at once, and each is handled on its
// own terms. "Is a mark present" is settled by the prefix Peek did manage to
// read, however short and whether or not the read also failed — markOf tests
// what it was given. "Did the source fail" is settled by Peek's error, and
// that error cannot simply be observed and forgotten: bufio.Reader.Peek takes
// it from readErr, which does `err := b.err; b.err = nil`, so bufio hands the
// error out exactly once and does not re-deliver it on the next read. Dropping
// it therefore destroys the only report the failure will ever make, and the
// retry that follows may answer with something unrelated — a source that fails
// once and then reports no more data would surface as a bare io.EOF, which
// Token documents as the end of a well-formed document. So a genuine
// (non-EOF) error is latched and re-surfaced by latchedErrReader once the
// bytes Peek buffered are drained. An io.EOF needs no latching: it says the
// stream is shorter than a mark, and the decoder reading that same short
// stream to its end reports the same io.EOF for itself.
func decodeBOM(r io.Reader) (io.Reader, bomEncoding) {
	br := bufio.NewReader(r)
	prefix, err := br.Peek(bomLen)
	enc, markLen := markOf(prefix)
	// Peek buffered the mark, so Discard consumes exactly markLen bytes (and
	// nothing at all for markLen 0); it has no failure this caller could act on.
	_, _ = br.Discard(markLen)

	body := io.Reader(br)
	if err != nil && !errors.Is(err, io.EOF) {
		body = &latchedErrReader{br: br, err: err}
	}
	if enc == bomUTF16BE || enc == bomUTF16LE {
		return newUTF16Reader(body, enc == bomUTF16BE), enc
	}
	return body, enc
}

// latchedErrReader hands back the bytes a bufio.Reader had already buffered
// and then reports a read error that bufio itself no longer holds. It exists
// because bufio.Reader.Peek clears its sticky error as it returns it
// (readErr: `err := b.err; b.err = nil`), so decodeBOM's peek is the last
// place that error is visible; without this wrapper the failure is dropped and
// the source is re-read instead, which is how a broken disk becomes a bare
// io.EOF and then "document has no root element". Collapsing this back into a
// plain *bufio.Reader reintroduces exactly that bug.
type latchedErrReader struct {
	br  *bufio.Reader
	err error
}

// Read serves whatever the peek left buffered, then reports the latched error
// without touching the underlying source again.
func (l *latchedErrReader) Read(p []byte) (int, error) {
	if l.br.Buffered() == 0 {
		return 0, l.err
	}
	// A non-empty buffer is served from memory: bufio.Reader.Read only calls
	// the source when its buffer is empty.
	return l.br.Read(p)
}

// markOf identifies the byte-order mark a document's leading bytes carry and
// how many bytes it occupies (XML 1.0 §4.3.3; byte sequences enumerated in
// Appendix F.1). Only the two-byte UTF-16 marks and the three-byte UTF-8 mark
// are recognized: the four-byte UCS-4 marks of that same table share their
// leading bytes, so extending this must test the longer sequences first.
func markOf(prefix []byte) (bomEncoding, int) {
	switch {
	case len(prefix) >= 3 && prefix[0] == 0xEF && prefix[1] == 0xBB && prefix[2] == 0xBF:
		return bomUTF8, 3
	case len(prefix) >= 2 && prefix[0] == 0xFE && prefix[1] == 0xFF:
		return bomUTF16BE, 2
	case len(prefix) >= 2 && prefix[0] == 0xFF && prefix[1] == 0xFE:
		return bomUTF16LE, 2
	}
	return bomNone, 0
}

// srcChunk is approximately how many UTF-16 source bytes one fill draws; the
// exact read length varies with how many bytes the previous fill carried over
// undecoded. Nothing depends on that length, or on its parity: a code unit
// split across two fills is preserved by decode's carry of the leftover bytes,
// not by any property of the chunk size.
const srcChunk = 4096

// maxCarry is the most source bytes that can survive a fill undecoded: a lead
// surrogate's two bytes plus the first byte of the unit that should pair with
// it.
const maxCarry = 3

// maxEmptyReads bounds a source that keeps returning (0, nil): without it a
// transcoding layer would spin where the bufio.Reader it displaces would have
// reported io.ErrNoProgress. bufio uses the same bound for the same reason.
const maxEmptyReads = 100

// utf16Reader transcodes a UTF-16 byte stream to UTF-8, streaming with bounded
// buffers (STYLE P4). Input that is not well-formed UTF-16 — an unpaired
// surrogate or a trailing partial code unit — is a terminal error rather than
// a U+FFFD substitution, so a document mangled at the encoding layer still
// fails well-formedness instead of parsing as replacement characters.
type utf16Reader struct {
	src    io.Reader
	bigEnd bool

	// buf holds source bytes not yet decoded; at most maxCarry of them survive
	// a fill.
	buf []byte
	// out holds decoded UTF-8; out[outOff:] is what the caller has yet to take.
	out    []byte
	outOff int
	// empty counts consecutive source reads that returned nothing at all.
	empty int
	// err latches the first terminal condition — io.EOF, a source failure, or
	// ill-formed UTF-16 — so it is reported once the decoded prefix is drained.
	err error
}

func newUTF16Reader(src io.Reader, bigEnd bool) *utf16Reader {
	return &utf16Reader{src: src, bigEnd: bigEnd, buf: make([]byte, 0, srcChunk+maxCarry)}
}

// Read returns decoded UTF-8, filling from the source until there is output to
// hand back or a terminal condition to report.
func (u *utf16Reader) Read(p []byte) (int, error) {
	for u.outOff == len(u.out) {
		if u.err != nil {
			return 0, u.err
		}
		u.fill()
	}
	n := copy(p, u.out[u.outOff:])
	u.outOff += n
	return n, nil
}

// fill draws one chunk from the source and decodes as much of it as forms
// whole code units.
func (u *utf16Reader) fill() {
	u.out, u.outOff = u.out[:0], 0
	base := len(u.buf)
	n, err := u.src.Read(u.buf[base:cap(u.buf)])
	if n == 0 && err == nil {
		u.empty++
		if u.empty == maxEmptyReads {
			u.err = io.ErrNoProgress
		}
		return
	}
	u.empty = 0
	u.buf = u.buf[:base+n]
	u.decode(errors.Is(err, io.EOF))
	if u.err != nil {
		return
	}
	u.err = err
}

// decode consumes whole UTF-16 code units from buf into out. atEOF reports
// that no further source bytes will arrive, making a leftover partial code
// unit or unpaired surrogate final rather than merely incomplete.
func (u *utf16Reader) decode(atEOF bool) {
	i := 0
	for len(u.buf)-i >= 2 {
		c := rune(u.unit(u.buf[i:]))
		if !utf16.IsSurrogate(c) {
			u.out = utf8.AppendRune(u.out, c)
			i += 2
			continue
		}
		if len(u.buf)-i < 4 {
			break
		}
		r := utf16.DecodeRune(c, rune(u.unit(u.buf[i+2:])))
		if r == unicode.ReplacementChar {
			u.err = fmt.Errorf("ill-formed UTF-16 input: unpaired surrogate %#04x", c)
			return
		}
		u.out = utf8.AppendRune(u.out, r)
		i += 4
	}
	u.buf = append(u.buf[:0], u.buf[i:]...)
	if atEOF && len(u.buf) > 0 {
		u.err = fmt.Errorf("ill-formed UTF-16 input: %d trailing byte(s) do not form a code unit", len(u.buf))
	}
}

// unit reads one UTF-16 code unit in the stream's byte order.
func (u *utf16Reader) unit(b []byte) uint16 {
	if u.bigEnd {
		return uint16(b[0])<<8 | uint16(b[1])
	}
	return uint16(b[1])<<8 | uint16(b[0])
}

// declaredEncoding returns the value of an XML declaration's encoding
// pseudo-attribute, or "" when the declaration omits it. inst is the
// declaration's content — everything between "<?xml" and "?>".
func declaredEncoding(inst string) string {
	rest := inst
	for {
		at := strings.Index(rest, "encoding")
		if at < 0 {
			return ""
		}
		rest = rest[at+len("encoding"):]
		value, ok := pseudoAttrValue(rest)
		if ok {
			return value
		}
	}
}

// pseudoAttrValue reads the quoted value of a pseudo-attribute whose name has
// just been consumed, reporting false when what follows is not "= 'value'".
func pseudoAttrValue(rest string) (string, bool) {
	rest = strings.TrimLeft(rest, " \t\r\n")
	if !strings.HasPrefix(rest, "=") {
		return "", false
	}
	rest = strings.TrimLeft(rest[1:], " \t\r\n")
	if rest == "" {
		return "", false
	}
	quote := rest[0]
	if quote != '\'' && quote != '"' {
		return "", false
	}
	end := strings.IndexByte(rest[1:], quote)
	if end < 0 {
		return "", false
	}
	return rest[1 : 1+end], true
}
