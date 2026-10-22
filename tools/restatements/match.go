package main

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// minShared is the overlap measure's threshold, which the package doc states.
const minShared = 2

// clauseBreak is where the overlap measure cuts an added sentence into
// clauses.
var clauseBreak = regexp.MustCompile(`[,;:()]|\s[-–—]\s`)

// wordPattern finds a sentence's words once it is lowercased: '\n' as written
// in Go source, or a run opening on a letter or digit that may carry the
// punctuation of an identifier, a rule ID, a path or a possessive.
var wordPattern = regexp.MustCompile(`'\\n'|[a-z0-9][a-z0-9.\-:#_/'’]*`)

// stopWords are the words the overlap measure drops.
var stopWords = func() map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(`a all also an and any are as at be been being but by can
		could do does each every for from had has have in into is it it's its may might must nor
		of on onto or over same shall should so such than that the their them then there these
		they this those to under was were what when where which who whom whose will with would
		package doc section see`) {
		set[w] = true
	}
	return set
}()

// contentWords reduces text to the overlap measure's content words, in order.
// A word's trailing punctuation goes with its possessive.
func contentWords(text string) []string {
	var out []string
	for _, w := range wordPattern.FindAllString(strings.ToLower(text), -1) {
		if w != `'\n'` {
			w = strings.TrimRight(w, ".:-/'’")
			w = strings.TrimSuffix(strings.TrimSuffix(w, "'s"), "’s")
		}
		if w == "" || stopWords[w] {
			continue
		}
		if len(w) > 4 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
			w = w[:len(w)-1]
		}
		if stopWords[w] {
			continue
		}
		out = append(out, w)
	}
	return out
}

// pair is two adjacent content words.
type pair [2]string

// pairs is the set of text's distinct adjacent content-word pairs, a lookup
// index (STYLE D2's permitted use of a map).
func pairs(text string) map[pair]bool {
	words := contentWords(text)
	set := make(map[pair]bool, len(words))
	for i := 1; i < len(words); i++ {
		set[pair{words[i-1], words[i]}] = true
	}
	return set
}

// maxHolders is the overlap measure's bound on how many owner sentences may
// hold a pair it counts, which the package doc states.
const maxHolders = 20

// common is the set of pairs more than maxHolders of owners hold, a lookup
// index (STYLE D2's permitted use of a map).
func common(owners []map[pair]bool) map[pair]bool {
	holders := make(map[pair]int)
	for _, o := range owners {
		for p := range o {
			holders[p]++
		}
	}
	set := make(map[pair]bool)
	for p, n := range holders {
		if n > maxHolders {
			set[p] = true
		}
	}
	return set
}

// shared is the most distinct pairs any one of clauses shares with owner,
// pairs in skip aside.
func shared(clauses []map[pair]bool, owner, skip map[pair]bool) int {
	best := 0
	for _, c := range clauses {
		n := 0
		for p := range c {
			if owner[p] && !skip[p] {
				n++
			}
		}
		best = max(best, n)
	}
	return best
}

// clauses returns the pair set of each of text's clauses.
func clauses(text string) []map[pair]bool {
	var out []map[pair]bool
	for _, c := range clauseBreak.Split(text, -1) {
		out = append(out, pairs(c))
	}
	return out
}

// match is one owner sentence an added sentence overlaps, and by how many
// pairs.
type match struct {
	owner  sentence
	shared int
}

// finding is one added sentence and every owner sentence it overlaps, the
// most-shared first.
type finding struct {
	added   sentence
	matches []match
}

// report is a survey's result: how many sentences the working tree added,
// and the ones that overlap an owner, in file and line order.
type report struct {
	added    int
	findings []finding
}

// survey reads the trees rooted at base and head and pairs every sentence
// head added with the owner sentences of head it overlaps.
func survey(base, head string) (report, error) {
	before, err := readTree(base)
	if err != nil {
		return report{}, fmt.Errorf("reading the base tree: %w", err)
	}
	after, err := readTree(head)
	if err != nil {
		return report{}, fmt.Errorf("reading the working tree: %w", err)
	}

	type key struct{ file, text string }
	existed := make(map[key]bool, len(before))
	for _, s := range before {
		existed[key{s.file, s.text}] = true
	}
	var added, owners []sentence
	for _, s := range after {
		if !existed[key{s.file, s.text}] {
			added = append(added, s)
		}
		if s.owner {
			owners = append(owners, s)
		}
	}
	sort.SliceStable(added, func(i, j int) bool {
		if added[i].file != added[j].file {
			return added[i].file < added[j].file
		}
		return added[i].line < added[j].line
	})

	ownerPairs := make([]map[pair]bool, len(owners))
	for i, o := range owners {
		ownerPairs[i] = pairs(o.text)
	}
	skip := common(ownerPairs)
	rep := report{added: len(added)}
	for _, a := range added {
		cs := clauses(a.text)
		var ms []match
		for i, o := range owners {
			if o == a {
				continue
			}
			if n := shared(cs, ownerPairs[i], skip); n >= minShared {
				ms = append(ms, match{owner: o, shared: n})
			}
		}
		if len(ms) == 0 {
			continue
		}
		sort.SliceStable(ms, func(i, j int) bool { return ms[i].shared > ms[j].shared })
		rep.findings = append(rep.findings, finding{added: a, matches: ms})
	}
	return rep, nil
}

// write prints the report: each finding's added sentence, then each owner
// sentence it overlaps indented beneath it, then the summary line every run
// ends with.
func (r report) write(w io.Writer) error {
	var b strings.Builder
	for _, f := range r.findings {
		fmt.Fprintf(&b, "%s:%d: %s\n", f.added.file, f.added.line, f.added.text)
		for _, m := range f.matches {
			fmt.Fprintf(&b, "\t%s:%d: (%d shared) %s\n", m.owner.file, m.owner.line, m.shared, m.owner.text)
		}
	}
	fmt.Fprintf(&b, "restatements: %d of %d added sentence(s) overlap a sentence of a package doc, an exported identifier's doc or cmd/goxsd8's usage; the matching is heuristic (%d adjacent content-word pairs shared with one clause, each held by at most %d owner sentences), so read each pair before acting on it\n",
		len(r.findings), r.added, minShared, maxHolders)
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("writing the report: %w", err)
	}
	return nil
}
