package main

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// refPattern is one issue reference in each form GitHub binds a closing
// keyword to — `#<N>`, `<owner>/<repo>#<N>` and an issue URL — with the
// number as its only group. The \b after the digits refuses a longer
// number's prefix while leaving a following `'s` outside the reference, so
// `#345's` names #345.
const refPattern = `(?:https://github\.com/[\w.-]+/[\w.-]+/issues/|(?:[\w.-]+/[\w.-]+)?#)(\d+)\b`

var (
	// keywordRef is a closing keyword from docs/WORKFLOW.md's Landing list
	// bound to the single reference that follows it. Groups: 1 the whole
	// reference, 2 its number. \s spans line breaks, since a line break does
	// not break the binding (#1034).
	keywordRef = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:es|ed)?|resolve[sd]?)(?::\s*|\s+)(` + refPattern + `)`)

	// anyRef is any reference, bound or not.
	anyRef = regexp.MustCompile(refPattern)

	// commaNext is the comma form's tail, anchored at the end of a bound
	// reference: a `,` and a further reference, which the keyword does not
	// close.
	commaNext = regexp.MustCompile(`^\s*,\s*` + refPattern)

	// leaveOpenPhrase is each phrase that declares the references in its
	// clause unclosed: docs/WORKFLOW.md's prescribed "names and leaves open",
	// and "not closing keywords", `2355fe0`'s form.
	leaveOpenPhrase = regexp.MustCompile(`(?i)\bnames\s+and\s+leaves\s+open\b|\bnot\s+closing\s+keywords\b`)

	// clauseBreak ends a clause: `.`, `!` or `?` before whitespace or the end
	// of the text, a blank line, or the start of a list item. A single line
	// break does not.
	clauseBreak = regexp.MustCompile(`[.!?](?:\s|$)|\n[ \t]*\n|\n[ \t]*(?:[-*+]|\d+\.)[ \t]`)
)

// landingText is one text GitHub reads closing keywords from at the merge:
// the squash commit text or the PR description.
type landingText struct {
	name string // how the report names it
	body string
}

// binding is one closing keyword bound to its reference, as byte offsets
// into the text it was found in.
type binding struct {
	start    int // the keyword's first byte
	refStart int // the bound reference's first byte
	refEnd   int // one past the bound reference's last byte
	issue    int
}

// written is the binding as its text spells it, whitespace collapsed so a
// binding across a line break reports on one line.
func (b binding) written(text string) string {
	return strings.Join(strings.Fields(text[b.start:b.refEnd]), " ")
}

// checkClosingKeywords rejects the closing-keyword forms docs/WORKFLOW.md's
// Landing section forbids, in every text GitHub closes from, before the
// merge rather than after it (#1178). Per binding, in document order:
//
//   - the comma form: the bound reference is followed by `,` and a further
//     reference, a list heading whose later items close nothing;
//   - the contradiction: the bound issue is also named, outside a binding
//     of its own, in a clause holding "names and leaves open" or "not
//     closing keywords" — per reference, so `Closes #625. Names and leaves
//     open #830.` passes;
//   - under closesNone, for a PR that closes no issue, the binding itself.
//
// It returns 1 when any binding is rejected, and reports what it saw for
// every text on both paths.
func checkClosingKeywords(texts []landingText, closesNone bool, stdout io.Writer) (int, error) {
	code := 0
	for _, t := range texts {
		bs, err := findBindings(t.body)
		if err != nil {
			return 0, fmt.Errorf("reading closing keywords in the %s: %w", t.name, err)
		}
		open, err := leftOpen(t.body, bs)
		if err != nil {
			return 0, fmt.Errorf("reading closing keywords in the %s: %w", t.name, err)
		}
		var findings []string
		for _, b := range bs {
			w := b.written(t.body)
			if commaNext.MatchString(t.body[b.refEnd:]) {
				findings = append(findings, fmt.Sprintf("%q is the comma form: a further reference follows #%d, and the keyword closes only #%d", w, b.issue, b.issue))
			}
			if slices.Contains(open, b.issue) {
				findings = append(findings, fmt.Sprintf("%q binds #%d, which a \"names and leaves open\" or \"not closing keywords\" clause also names", w, b.issue))
			}
			if closesNone {
				findings = append(findings, fmt.Sprintf("%q binds #%d in a PR that closes no issue", w, b.issue))
			}
		}
		if len(findings) > 0 {
			code = 1
		}
		for _, f := range findings {
			if _, err := fmt.Fprintf(stdout, "landcheck: %s: %s\n", t.name, f); err != nil {
				return 0, err
			}
		}
		if len(findings) == 0 {
			if _, err := fmt.Fprintf(stdout, "landcheck: %s: closing keywords %s\n", t.name, describeBound(bs)); err != nil {
				return 0, err
			}
		}
	}
	return code, nil
}

// describeBound names the issues bs binds, for the clean report.
func describeBound(bs []binding) string {
	if len(bs) == 0 {
		return "bind nothing"
	}
	refs := make([]string, len(bs))
	for i, b := range bs {
		refs[i] = "#" + strconv.Itoa(b.issue)
	}
	return "bind " + strings.Join(refs, " ")
}

// findBindings returns every closing keyword in text bound to a reference,
// in document order.
func findBindings(text string) ([]binding, error) {
	var bs []binding
	for _, m := range keywordRef.FindAllStringSubmatchIndex(text, -1) {
		n, err := strconv.Atoi(text[m[4]:m[5]])
		if err != nil {
			return nil, fmt.Errorf("issue number %q at byte %d: %w", text[m[4]:m[5]], m[4], err)
		}
		bs = append(bs, binding{start: m[0], refStart: m[2], refEnd: m[3], issue: n})
	}
	return bs, nil
}

// leftOpen returns the issues text names inside a leave-open clause — the
// clause holding a leaveOpenPhrase, bounded by clauseBreak — in document
// order. A reference that is itself one of bs's bound references is not
// counted: it is the binding, not a mention of it.
func leftOpen(text string, bs []binding) ([]int, error) {
	breaks := clauseBreak.FindAllStringIndex(text, -1)
	var issues []int
	for _, p := range leaveOpenPhrase.FindAllStringIndex(text, -1) {
		start, end := 0, len(text)
		for _, br := range breaks {
			if br[1] <= p[0] {
				start = br[1]
			}
			if br[0] >= p[1] {
				end = br[0]
				break
			}
		}
		for _, m := range anyRef.FindAllStringSubmatchIndex(text[start:end], -1) {
			if isBoundRef(bs, start+m[0]) {
				continue
			}
			n, err := strconv.Atoi(text[start+m[2] : start+m[3]])
			if err != nil {
				return nil, fmt.Errorf("issue number %q at byte %d: %w", text[start+m[2]:start+m[3]], start+m[2], err)
			}
			issues = append(issues, n)
		}
	}
	return issues, nil
}

// isBoundRef reports whether a reference starting at offset is one of bs's
// bound references.
func isBoundRef(bs []binding, offset int) bool {
	for _, b := range bs {
		if b.refStart == offset {
			return true
		}
	}
	return false
}
