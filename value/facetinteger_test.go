package value

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// lengthStub is a test-only value.Lengthed of a fixed length.
type lengthStub int

func (l lengthStub) Len() int { return int(l) }

// digitsStub is a test-only value.DigitCounted with fixed digit counts.
type digitsStub struct{ total, fraction int }

func (d digitsStub) TotalDigits() int    { return d.total }
func (d digitsStub) FractionDigits() int { return d.fraction }

// huge is a length/digits/scale facet {value} past math.MaxInt, in the
// unbounded value space of each (Datatypes §3.4.20, §3.4.13).
const huge = "99999999999999999999"

// TestFacetValuePastMaxInt pins facetCount and facetInt on {value}s past the
// host int (#1779): each row's facet failed to construct while the readers used
// strconv.Atoi, which made the instance undecided. The MaxInt rows compare an
// instance measure of exactly math.MaxInt with a {value} one past it, which
// saturating the {value} at math.MaxInt would get wrong. msg, when set, is the
// rejection's whole opening up to the limit it names: it fails if the limit is
// printed as a parsed or saturated int rather than the literal.
func TestFacetValuePastMaxInt(t *testing.T) {
	if strconv.IntSize != 64 {
		t.Skip("the MaxInt rows spell a 64-bit math.MaxInt + 1")
	}
	const maxIntPlus1 = "9223372036854775808"
	stringPrim := primType(t, "string", "preserve")
	length := func(kind xsd.FacetKind, limit string) func() (ValueFacet, error) {
		return func() (ValueFacet, error) {
			return newLengthFacet(noSchema{}, stringPrim, xsd.NewFacet(kind, []string{limit}, false))
		}
	}
	digits := func(kind xsd.FacetKind, limit string) func() (ValueFacet, error) {
		return func() (ValueFacet, error) { return newDigitsFacet(xsd.NewFacet(kind, []string{limit}, false)) }
	}
	scale := func(kind xsd.FacetKind, limit string) func() (ValueFacet, error) {
		return func() (ValueFacet, error) { return newScaleFacet(xsd.NewFacet(kind, []string{limit}, false)) }
	}

	cases := []struct {
		name  string
		facet func() (ValueFacet, error)
		v     Value
		rule  xsderr.Rule // "" means accepted
		msg   string
	}{
		// --- facetCount via newLengthFacet ---
		{name: "minLength huge rejects abc", facet: length(xsd.FacetMinLength, huge), v: lengthStub(3),
			rule: ruleCvcMinLengthValid, msg: "value length 3 violates the minLength facet limit " + huge},
		{name: "length huge rejects abc", facet: length(xsd.FacetLength, huge), v: lengthStub(3),
			rule: ruleCvcLengthValid, msg: "value length 3 violates the length facet limit " + huge},
		{name: "maxLength huge accepts abc", facet: length(xsd.FacetMaxLength, huge), v: lengthStub(3)},
		{name: "length MaxInt+1 rejects MaxInt", facet: length(xsd.FacetLength, maxIntPlus1), v: lengthStub(math.MaxInt),
			rule: ruleCvcLengthValid, msg: "value length " + strconv.Itoa(math.MaxInt) + " violates the length facet limit " + maxIntPlus1},
		{name: "maxLength +007 names the literal", facet: length(xsd.FacetMaxLength, "+007"), v: lengthStub(8),
			rule: ruleCvcMaxLengthValid, msg: "value length 8 violates the maxLength facet limit +007"},
		{name: "maxLength +007 accepts 7", facet: length(xsd.FacetMaxLength, "+007"), v: lengthStub(7)},
		{name: "totalDigits +03 names the literal", facet: digits(xsd.FacetTotalDigits, "+03"), v: digitsStub{total: 4},
			rule: ruleCvcTotalDigitsValid, msg: "value has 4 totalDigits, exceeds facet limit +03"},

		// --- facetCount via newDigitsFacet ---
		{name: "totalDigits huge accepts", facet: digits(xsd.FacetTotalDigits, huge), v: digitsStub{total: 2, fraction: 1}},
		{name: "fractionDigits huge accepts", facet: digits(xsd.FacetFractionDigits, huge), v: digitsStub{total: 2, fraction: 1}},

		// --- facetInt via newScaleFacet ---
		{name: "minScale huge rejects scale 1", facet: scale(xsd.FacetMinScale, huge), v: scaledStub{scale: 1, present: true},
			rule: ruleCvcMinScaleValid, msg: "value scale 1 violates the minScale facet limit " + huge},
		{name: "maxScale huge accepts scale 1", facet: scale(xsd.FacetMaxScale, huge), v: scaledStub{scale: 1, present: true}},
		{name: "minScale -huge accepts scale -1", facet: scale(xsd.FacetMinScale, "-"+huge), v: scaledStub{scale: -1, present: true}},
		{name: "maxScale -huge rejects scale -1", facet: scale(xsd.FacetMaxScale, "-"+huge), v: scaledStub{scale: -1, present: true},
			rule: ruleCvcMaxScaleValid, msg: "value scale -1 violates the maxScale facet limit -" + huge},
		{name: "minScale MaxInt+1 rejects scale MaxInt", facet: scale(xsd.FacetMinScale, maxIntPlus1), v: scaledStub{scale: math.MaxInt, present: true},
			rule: ruleCvcMinScaleValid, msg: "value scale " + strconv.Itoa(math.MaxInt) + " violates the minScale facet limit " + maxIntPlus1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			facet, err := c.facet()
			if err != nil {
				t.Fatalf("building the facet: %v", err)
			}
			err = facet.CheckValue(c.v)
			if c.rule == "" {
				if err != nil {
					t.Fatalf("CheckValue = %v, want accept", err)
				}
				return
			}
			var xe *xsderr.Error
			if !errors.As(err, &xe) || xe.Rule != c.rule {
				t.Fatalf("CheckValue = %v, want a rejection charging %s", err, c.rule)
			}
			if !strings.HasPrefix(xe.Msg, c.msg) {
				t.Errorf("message = %q, want it to open %q", xe.Msg, c.msg)
			}
		})
	}
}

// TestFacetValueNotAnInteger pins that facetCount and facetInt still charge a
// {value} outside their lexical space under the per-facet rule, however long.
func TestFacetValueNotAnInteger(t *testing.T) {
	cases := []struct {
		kind  xsd.FacetKind
		value string
		rule  xsderr.Rule
	}{
		{xsd.FacetTotalDigits, "-" + huge, ruleCvcTotalDigitsValid},
		{xsd.FacetFractionDigits, huge + ".0", ruleCvcFractionDigitsValid},
		{xsd.FacetMaxScale, huge + "x", ruleCvcMaxScaleValid},
		{xsd.FacetMinScale, "", ruleCvcMinScaleValid},
	}
	for _, c := range cases {
		f := xsd.NewFacet(c.kind, []string{c.value}, false)
		var err error
		if c.kind == xsd.FacetTotalDigits || c.kind == xsd.FacetFractionDigits {
			_, err = newDigitsFacet(f)
		}
		if c.kind == xsd.FacetMaxScale || c.kind == xsd.FacetMinScale {
			_, err = newScaleFacet(f)
		}
		if r, _ := xsderr.RuleOf(err); r != c.rule {
			t.Errorf("%s %q: err = %v, want a rejection charging %s", c.kind, c.value, err, c.rule)
		}
	}
}
