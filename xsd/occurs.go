package xsd

import (
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// ruleParticleCorrect is Particle Correct (Structures §3.9.6.1). Its clause 1
// requires a particle's property values to match the §3.9.1 tableau — {min
// occurs} a nonNegativeInteger, {max occurs} a nonNegativeInteger or unbounded
// — and its clause 2.1 requires {min occurs} not greater than a numeric {max
// occurs}. Occurs' constructors reject every state those clauses forbid.
const ruleParticleCorrect xsderr.Rule = "p-props-correct"

// unboundedMax is the internal sentinel that encodes an unbounded {max occurs}.
// It never escapes: Max reports unboundedness through the ok result, so no
// caller ever handles a magic maxOccurs literal. A single field carrying either
// a real bound or this sentinel keeps one fact in one encoding (STYLE D3) and
// makes "unbounded yet also bounded to N" unrepresentable (STYLE T1).
const unboundedMax = -1

// Occurs is a particle's occurrence range: its {min occurs} and {max occurs}
// properties (Structures §3.9.1), where {max occurs} is either a numeric bound
// or unbounded. It is the scalar value used after the minOccurs/maxOccurs XML
// attributes (each defaulting to 1) have been resolved; it does no attribute
// parsing itself.
//
// Construct it only through NewOccurs or NewUnboundedOccurs, which reject the
// states Particle Correct (§3.9.6.1) forbids — a negative bound, or a bounded
// {max occurs} below {min occurs} — so an inverted or negative range is
// unrepresentable (STYLE T1).
//
// The zero value is the vacuous range 0..0 (a legal particle: §3.9.2 admits
// maxOccurs 0), which Permits only n == 0. It is a valid Occurs, not an
// absent/invalid marker.
type Occurs struct {
	min int
	max int
}

// NewOccurs builds a bounded occurrence range. It rejects a negative min or max
// (both are nonNegativeInteger value spaces, §3.9.1 via §3.9.2's xs:allNNI) and
// a max below min (Particle Correct §3.9.6.1, clause 2.1), returning an
// *xsderr.Error carrying rule p-props-correct. A bounded max of 0 (a vacuous
// particle) is legal and forces min to 0.
func NewOccurs(min, max int) (Occurs, error) {
	if min < 0 {
		return Occurs{}, xsderr.New(ruleParticleCorrect, xsderr.Loc{},
			"particle {min occurs} must be a nonNegativeInteger, got %d", min)
	}
	if max < 0 {
		return Occurs{}, xsderr.New(ruleParticleCorrect, xsderr.Loc{},
			"particle {max occurs} must be a nonNegativeInteger or unbounded, got %d", max)
	}
	if max < min {
		return Occurs{}, xsderr.New(ruleParticleCorrect, xsderr.Loc{},
			"particle {min occurs} %d is greater than {max occurs} %d", min, max)
	}
	return Occurs{min: min, max: max}, nil
}

// NewUnboundedOccurs builds an occurrence range whose {max occurs} is
// unbounded. It rejects a negative min (nonNegativeInteger, §3.9.1), returning
// an *xsderr.Error carrying rule p-props-correct. Particle Correct clause 2
// imposes no upper-bound check when {max occurs} is unbounded, so any
// nonNegativeInteger min is legal.
func NewUnboundedOccurs(min int) (Occurs, error) {
	if min < 0 {
		return Occurs{}, xsderr.New(ruleParticleCorrect, xsderr.Loc{},
			"particle {min occurs} must be a nonNegativeInteger, got %d", min)
	}
	return Occurs{min: min, max: unboundedMax}, nil
}

// Min returns the {min occurs} property.
func (o Occurs) Min() int {
	return o.min
}

// IsUnbounded reports whether {max occurs} is unbounded.
func (o Occurs) IsUnbounded() bool {
	return o.max == unboundedMax
}

// Max returns the numeric {max occurs} bound; the second result is false when
// {max occurs} is unbounded, in which case the first result is not meaningful.
func (o Occurs) Max() (int, bool) {
	if o.max == unboundedMax {
		return 0, false
	}
	return o.max, true
}

// Permits reports whether n occurrences fall within the range: min <= n <= max,
// with an unbounded max dropping the upper bound. Permits(Min()) is always
// true, and Permits(max) is true for a bounded max (both ends inclusive). A
// negative n is always false, since min is a nonNegativeInteger.
func (o Occurs) Permits(n int) bool {
	if n < o.min {
		return false
	}
	if o.max == unboundedMax {
		return true
	}
	return n <= o.max
}

// String renders the range: a lone bound when min == max (e.g. "1"), "min..max"
// for a wider bounded range (e.g. "2..5"), and "min..unbounded" when {max
// occurs} is unbounded (e.g. "1..unbounded"), spelling the token exactly as the
// schema does.
func (o Occurs) String() string {
	if o.max == unboundedMax {
		return strconv.Itoa(o.min) + "..unbounded"
	}
	if o.min == o.max {
		return strconv.Itoa(o.min)
	}
	return strconv.Itoa(o.min) + ".." + strconv.Itoa(o.max)
}
