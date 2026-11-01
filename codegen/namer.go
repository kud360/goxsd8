package codegen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/internal/goname"
)

// nameStep is one step of the path a type is named by: an XSD local name
// and whether it is a declared step or a generated word, as the package doc's
// "# Naming" tells them apart for shortening. Build one with declaredStep or
// generatedSteps.
type nameStep struct {
	word     string
	declared bool
}

// declaredStep returns the step for a local name a declaration supplies.
func declaredStep(word string) nameStep {
	return nameStep{word: word, declared: true}
}

// generatedSteps returns one generated step per word, in order.
func generatedSteps(words ...string) []nameStep {
	steps := make([]nameStep, len(words))
	for i, w := range words {
		steps[i] = nameStep{word: w}
	}
	return steps
}

// typeNamer claims a package's type names by the package doc's "# Naming".
// emit walks the Schema twice, through two implementations: surveyNamer
// records every claim, and replayNamer hands out the names resolve computes
// from that record. fixed claims a name rendered from words that never
// shortens; path claims a name by a path of steps, shortened to its shortest
// unique suffix; extend claims of, a name this typeNamer returned, followed
// by words.
type typeNamer interface {
	fixed(words ...string) string
	path(steps []nameStep) string
	extend(of string, words ...string) string
}

// typeClaim is one claim surveyNamer records, as the typeNamer method it was
// made through: a fixedClaim, a pathClaim or an extendClaim.
type typeClaim interface{ isTypeClaim() }

// fixedClaim is a fixed claim: its words.
type fixedClaim []string

// pathClaim is a path claim: its path.
type pathClaim []nameStep

// extendClaim is an extending claim: of, the index of the claim it extends,
// which is always an earlier one, and its words.
type extendClaim struct {
	of    int
	words []string
}

func (fixedClaim) isTypeClaim()  {}
func (pathClaim) isTypeClaim()   {}
func (extendClaim) isTypeClaim() {}

// surveyNamer is the typeNamer of emit's first walk: it records each claim in
// order and returns the provisional name a namer claims for it from its whole
// path, unshortened, which no output shows.
type surveyNamer struct {
	provisional *goname.Namer
	claims      []typeClaim
	// index maps each provisional name to its claim's index, so that an
	// extending claim finds the claim it extends; it is a lookup index only
	// and never iterated (STYLE D2).
	index map[string]int
	// err is the first claim that extended a name this namer never returned.
	err error
}

func newSurveyNamer() *surveyNamer {
	return &surveyNamer{provisional: goname.NewNamer(), index: map[string]int{}}
}

func (s *surveyNamer) record(c typeClaim, candidate string) string {
	name := s.provisional.ClaimCandidate(candidate)
	s.index[name] = len(s.claims)
	s.claims = append(s.claims, c)
	return name
}

func (s *surveyNamer) fixed(words ...string) string {
	return s.record(fixedClaim(slices.Clone(words)), goname.Render(words))
}

func (s *surveyNamer) path(steps []nameStep) string {
	return s.record(pathClaim(slices.Clone(steps)), renderSteps(steps))
}

func (s *surveyNamer) extend(of string, words ...string) string {
	i, ok := s.index[of]
	if !ok && s.err == nil {
		s.err = fmt.Errorf("the type name %s extended by %q was never claimed", of, words)
	}
	return s.record(extendClaim{of: i, words: slices.Clone(words)}, goname.Identifier(of)+goname.Render(words))
}

// replayNamer is the typeNamer of emit's second walk: it returns names[i] for
// the i-th claim, names being resolve's for claims, the first walk's record.
// The second walk walks the same Schema as the first, and its control flow
// reads a name only for equality: with "", for a once-per-package type not
// yet claimed, and with another name, in the attributes structs' lookups by
// the name of the struct they hold the fields of. Each walk's names, all
// distinct and none "", answer both alike, so it makes the same claims; a
// claim that differs from its record is an internal error, kept in err, and
// its name is "".
type replayNamer struct {
	claims []typeClaim
	names  []string
	next   int
	err    error
}

// take advances to the next claim and returns its index and record, or -1
// and nil when err already holds an error or the record holds no more
// claims, the latter kept in err.
func (r *replayNamer) take() (int, typeClaim) {
	i := r.next
	r.next++
	if r.err != nil {
		return -1, nil
	}
	if i >= len(r.claims) {
		r.err = fmt.Errorf("type claim %d: the first walk made %d", i, len(r.claims))
		return -1, nil
	}
	return i, r.claims[i]
}

// answer returns names[i] for claim i when same reports it equal to its
// record, and otherwise "", the difference kept in err; i is take's, and
// for -1 answer returns "" and keeps take's error.
func (r *replayNamer) answer(i int, same bool) string {
	if i < 0 {
		return ""
	}
	if !same {
		r.err = fmt.Errorf("type claim %d differs from the first walk's", i)
		return ""
	}
	return r.names[i]
}

func (r *replayNamer) fixed(words ...string) string {
	i, c := r.take()
	want, ok := c.(fixedClaim)
	return r.answer(i, ok && slices.Equal(words, []string(want)))
}

func (r *replayNamer) path(steps []nameStep) string {
	i, c := r.take()
	want, ok := c.(pathClaim)
	return r.answer(i, ok && slices.Equal(steps, []nameStep(want)))
}

func (r *replayNamer) extend(of string, words ...string) string {
	i, c := r.take()
	want, ok := c.(extendClaim)
	return r.answer(i, ok && slices.Equal(words, want.words) && of == r.names[want.of])
}

// renderSteps joins the identifier of each step's word.
func renderSteps(steps []nameStep) string {
	var b strings.Builder
	for _, s := range steps {
		b.WriteString(goname.Identifier(s.word))
	}
	return b.String()
}

// resolve returns the name of each of claims, in order, by the package doc's
// "# Naming": each claim's candidate — a fixed claim's rendering, an extending
// claim's extended name, as resolve names it, followed by its words'
// rendering, a path claim's shortest unique suffix (shortest) — then the
// numeric suffix a namer gives a candidate already taken, in claim order.
func resolve(claims []typeClaim) []string {
	full := make([][]string, len(claims))
	fixed := map[string]bool{} // fixed claims' renderings; a lookup index only, never iterated (STYLE D2)
	for i, c := range claims {
		switch c := c.(type) {
		case fixedClaim:
			full[i] = identifiers(c)
			fixed[strings.Join(full[i], "")] = true
		case pathClaim:
			for _, s := range c {
				full[i] = append(full[i], goname.Identifier(s.word))
			}
		case extendClaim:
			full[i] = slices.Concat(full[c.of], identifiers(c.words))
		}
	}
	names := make([]string, len(claims))
	n := goname.NewNamer()
	for i, c := range claims {
		var candidate string
		switch c := c.(type) {
		case fixedClaim:
			candidate = goname.Render(c)
		case pathClaim:
			candidate = shortest(i, c, full, fixed)
		case extendClaim:
			candidate = goname.Identifier(names[c.of]) + goname.Render(c.words)
		}
		names[i] = n.ClaimCandidate(candidate)
	}
	return names
}

// identifiers returns the identifier of each XSD local name in words.
func identifiers(words []string) []string {
	ids := make([]string, len(words))
	for i, w := range words {
		ids[i] = goname.Identifier(w)
	}
	return ids
}

// shortest returns the candidate of the path claim i, whose path is steps, by
// the package doc's "# Naming" rule for shortening a path: full holds every
// claim's whole path, each step rendered, and fixed every fixed claim's
// rendering.
func shortest(i int, steps pathClaim, full [][]string, fixed map[string]bool) string {
	for k := len(steps) - 1; k > 0; k-- {
		suffix := full[i][k:]
		if !steps[k].declared || fixed[strings.Join(suffix, "")] {
			continue
		}
		unique := true
		for j, other := range full {
			if j != i && len(other) >= len(suffix) && slices.Equal(other[len(other)-len(suffix):], suffix) {
				unique = false
				break
			}
		}
		if unique {
			return strings.Join(suffix, "")
		}
	}
	return strings.Join(full[i], "")
}
