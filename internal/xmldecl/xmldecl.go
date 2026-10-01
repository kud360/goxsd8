package xmldecl

import "io"

// As10 returns a reader over r that presents a leading XML declaration's
// VersionNum (XML 1.0 [26] '1.' [0-9]+) as "1.0" whenever it is any other 1.x
// number, and passes every other byte through unchanged.
//
// r must be positioned at the document's first character: the XMLDecl ([23])
// is only ever the first thing in a document entity, so a byte-order mark is
// the caller's to have consumed. Only the declaration's head — '<?xml' S
// 'version' Eq and the quoted VersionNum ([23]-[26]) — is read ahead; what does
// not match that grammar is passed through untouched, for the XML decoder to
// judge as it would have.
//
// The rewrite keeps the stream's length: a one-digit minor version becomes '0'
// in place, and a longer one becomes "0" followed by the closing quote, with
// the digits it displaced turned into spaces after the quote. That second form
// is still an XMLDecl only when the source's quote was followed by S or the
// '?' of '?>', so it is applied only then; any other byte there is a malformed
// declaration, left for the decoder to reject.
func As10(r io.Reader) io.Reader {
	return &reader{src: r}
}

// reader is As10's stream: head holds the declaration bytes read ahead and not
// yet handed out, already rewritten.
type reader struct {
	src     io.Reader
	head    []byte
	scanned bool
	// err is the read failure, io.EOF included, that ended the read-ahead; it
	// is reported once head is drained, without reading src again.
	err error
}

// Read serves the read-ahead first, then the source.
func (x *reader) Read(p []byte) (int, error) {
	if !x.scanned {
		x.scanned = true
		x.scan()
	}
	if len(x.head) > 0 {
		n := copy(p, x.head)
		x.head = x.head[n:]
		return n, nil
	}
	if x.err != nil {
		return 0, x.err
	}
	return x.src.Read(p)
}

// scan reads the declaration head into x.head and rewrites its VersionNum.
func (x *reader) scan() {
	if !x.literal("<?xml") {
		return
	}
	b, spaced, ok := x.afterSpace()
	if !ok || !spaced || b != 'v' || !x.literal("ersion") {
		return
	}
	// Eq ([25]): S? '=' S?.
	if b, _, ok = x.afterSpace(); !ok || b != '=' {
		return
	}
	quote, _, ok := x.afterSpace()
	if !ok || (quote != '"' && quote != '\'') || !x.literal("1.") {
		return
	}
	start := len(x.head)
	for {
		if b, ok = x.next(); !ok {
			return
		}
		if b == quote {
			break
		}
		if b < '0' || b > '9' {
			return
		}
	}
	x.rewrite(start, len(x.head)-1-start, quote)
}

// rewrite turns the n minor-version digits at x.head[start:], which the closing
// quote follows, into "0".
func (x *reader) rewrite(start, n int, quote byte) {
	switch {
	case n == 0, n == 1 && x.head[start] == '0':
		// "1." is no VersionNum, and "1.0" needs nothing.
		return
	case n == 1:
		x.head[start] = '0'
		return
	}
	b, ok := x.next()
	if !ok || (!isSpace(b) && b != '?') {
		return
	}
	x.head[start] = '0'
	x.head[start+1] = quote
	for i := start + 2; i < len(x.head)-1; i++ {
		x.head[i] = ' '
	}
}

// literal reads len(s) bytes and reports whether they spell s, stopping at the
// first that does not.
func (x *reader) literal(s string) bool {
	for i := 0; i < len(s); i++ {
		b, ok := x.next()
		if !ok || b != s[i] {
			return false
		}
	}
	return true
}

// afterSpace reads past any S ([3]) and returns the first byte that is not
// one, reporting whether any S preceded it.
func (x *reader) afterSpace() (byte, bool, bool) {
	spaced := false
	for {
		b, ok := x.next()
		if !ok || !isSpace(b) {
			return b, spaced, ok
		}
		spaced = true
	}
}

// next reads one byte from the source into the read-ahead, latching the read
// failure that ends it.
func (x *reader) next() (byte, bool) {
	var b [1]byte
	if _, err := io.ReadFull(x.src, b[:]); err != nil {
		x.err = err
		return 0, false
	}
	x.head = append(x.head, b[0])
	return b[0], true
}

// isSpace reports whether b is one of XML 1.0's S characters ([3]).
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}
