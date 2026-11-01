package codegen

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// namer is the one component that turns XSD names into Go identifiers
// (package doc, "# Naming"). One namer scopes one set of identifiers that
// must not collide: a package's type names, or one struct's field names.
// Claims are made in document order, and that order settles every collision
// among the candidates it is handed; a package's type candidates are resolve's.
type namer struct {
	// claimed indexes the identifiers handed out so far; it is a lookup
	// index only and never iterated (STYLE D2).
	claimed map[string]bool
}

func newNamer() *namer {
	return &namer{claimed: map[string]bool{}}
}

// claim returns a fresh identifier in this scope for the path of XSD local
// names, by the package doc's "# Naming" rules for paths and collisions.
func (n *namer) claim(path ...string) string {
	return n.claimCandidate(render(path))
}

// claimCandidate returns candidate, an identifier already rendered, or,
// when this scope holds it, candidate followed by the smallest free numeric
// suffix from 2.
func (n *namer) claimCandidate(candidate string) string {
	name := candidate
	for i := 2; n.claimed[name]; i++ {
		name = candidate + strconv.Itoa(i)
	}
	n.claimed[name] = true
	return name
}

// render joins the identifier of each XSD local name in path.
func render(path []string) string {
	var b strings.Builder
	for _, local := range path {
		b.WriteString(identifier(local))
	}
	return b.String()
}

// identifier maps one XSD local name (an NCName) to an exported Go
// identifier by the package doc's "# Naming" rules for one name.
func identifier(local string) string {
	var b strings.Builder
	breakWord := true
	for _, r := range local {
		if r == '_' || !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			breakWord = true
			continue
		}
		if breakWord {
			r = unicode.ToUpper(r)
			breakWord = false
		}
		b.WriteRune(r)
	}
	s := b.String()
	first, _ := utf8.DecodeRuneInString(s)
	if !unicode.IsUpper(first) {
		return "X" + s
	}
	return s
}

// nameStep is one step of the path a type is named by: an XSD local name
// and whether a declaration supplies it — a scope's root, an element, an
// attribute or its owner, or a named list — rather than the generator, as it
// does a compositor word, a {type table} slot step, value or union. Only a
// declared step can start the suffix a path is shortened to ("# Naming").
// Build one with declaredStep or generatedSteps.
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

// claimKind is the typeNamer method one typeClaim was made through. The zero
// value is invalid (STYLE T1).
type claimKind uint8

// The claimKind values.
const (
	fixedClaim claimKind = iota + 1
	pathClaim
	extendClaim
)

// typeClaim is one claim surveyNamer records: its kind and its steps — a
// path claim's path, or a fixed or extending claim's words, as generated
// steps, since neither shortens — and, for an extending claim, of, the index
// of the claim it extends, which is always an earlier one.
type typeClaim struct {
	kind  claimKind
	steps []nameStep
	of    int
}

// surveyNamer is the typeNamer of emit's first walk: it records each claim in
// order and returns the provisional name a namer claims for it, today's
// rendering of its whole path, which the first walk's output never shows.
type surveyNamer struct {
	provisional *namer
	claims      []typeClaim
	// index maps each provisional name to its claim's index, so that an
	// extending claim finds the claim it extends; it is a lookup index only
	// and never iterated (STYLE D2).
	index map[string]int
	// err is the first claim that extended a name this namer never returned.
	err error
}

func newSurveyNamer() *surveyNamer {
	return &surveyNamer{provisional: newNamer(), index: map[string]int{}}
}

func (s *surveyNamer) record(c typeClaim, candidate string) string {
	name := s.provisional.claimCandidate(candidate)
	s.index[name] = len(s.claims)
	s.claims = append(s.claims, c)
	return name
}

func (s *surveyNamer) fixed(words ...string) string {
	return s.record(typeClaim{kind: fixedClaim, steps: generatedSteps(words...)}, render(words))
}

func (s *surveyNamer) path(steps []nameStep) string {
	return s.record(typeClaim{kind: pathClaim, steps: slices.Clone(steps)}, renderSteps(steps))
}

func (s *surveyNamer) extend(of string, words ...string) string {
	i, ok := s.index[of]
	if !ok && s.err == nil {
		s.err = fmt.Errorf("the type name %s extended by %q was never claimed", of, words)
	}
	return s.record(typeClaim{kind: extendClaim, steps: generatedSteps(words...), of: i}, identifier(of)+render(words))
}

// replayNamer is the typeNamer of emit's second walk: it returns names[i] for
// the i-th claim, names being resolve's for claims, the first walk's record.
// The second walk walks the same Schema as the first and no control flow
// depends on a name's text, so it makes the same claims; a claim that differs
// from its record is an internal error, kept in err, and its name is "".
type replayNamer struct {
	claims []typeClaim
	names  []string
	next   int
	err    error
}

func (r *replayNamer) replay(c typeClaim, extended string) string {
	i := r.next
	r.next++
	if r.err != nil {
		return ""
	}
	if i >= len(r.claims) {
		r.err = fmt.Errorf("type claim %d: the first walk made %d", i, len(r.claims))
		return ""
	}
	want := r.claims[i]
	if c.kind != want.kind || !slices.Equal(c.steps, want.steps) || c.kind == extendClaim && extended != r.names[want.of] {
		r.err = fmt.Errorf("type claim %d differs from the first walk's", i)
		return ""
	}
	return r.names[i]
}

func (r *replayNamer) fixed(words ...string) string {
	return r.replay(typeClaim{kind: fixedClaim, steps: generatedSteps(words...)}, "")
}

func (r *replayNamer) path(steps []nameStep) string {
	return r.replay(typeClaim{kind: pathClaim, steps: steps}, "")
}

func (r *replayNamer) extend(of string, words ...string) string {
	return r.replay(typeClaim{kind: extendClaim, steps: generatedSteps(words...)}, of)
}

// renderSteps joins the identifier of each step's word.
func renderSteps(steps []nameStep) string {
	var b strings.Builder
	for _, s := range steps {
		b.WriteString(identifier(s.word))
	}
	return b.String()
}

// resolve returns the name of each of claims, in order, by the package doc's
// "# Naming": each claim's candidate — a fixed claim's rendering, an extending
// claim's extended name followed by its words' rendering, a path claim's
// rendering — then the numeric suffix a namer gives a candidate already
// taken, in claim order.
func resolve(claims []typeClaim) []string {
	names := make([]string, len(claims))
	n := newNamer()
	for i, c := range claims {
		candidate := renderSteps(c.steps)
		if c.kind == extendClaim {
			candidate = identifier(names[c.of]) + candidate
		}
		names[i] = n.claimCandidate(candidate)
	}
	return names
}
