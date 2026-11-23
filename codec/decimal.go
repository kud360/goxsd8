package codec

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
)

// decimalValue is codec's value of xs:precisionDecimal (precisionDecimal §3.1
// sec-vs-pD), a closed sum with one arm per case of its ·numericalValue·
// (vp-pd-numVal): decimalNumerical, decimalPositiveInfinity,
// decimalNegativeInfinity and decimalNotANumber. It converts to no caller
// type: codegen's PrecisionDecimal holds its value in a field of the caller's
// own sealed interface, so pipeline.decimal builds the caller's struct from
// it, setting the caller's arm that armOf chooses.
type decimalValue interface {
	// armOf returns the arm type of arms whose role is the arm's.
	armOf(arms decimalArms) reflect.Type
	// lexical spells the value as a pDecimalRep that denotes it exactly
	// (precisionDecimal §3.2 nt-precDecRep), the spelling Append hands the
	// governing mapping's Parse: a numerical value as its Coefficient, then E
	// and its Scale negated, a leading '-' where Negative, zero included, and
	// the special values as INF, -INF and NaN.
	lexical() string
	// otherLexical reports whether a pDecimalRep other than the one the
	// governing mapping's Canonical renders denotes the value, which a pattern
	// facet may match where it rejects that one (Datatypes §2.1's note): a
	// numerical value's, spelled with a leading '+' or zero, and
	// positiveInfinity's, +INF, do; negativeInfinity's -INF and notANumber's
	// NaN are their only lexicals, as item 7 of [Append]'s doc states.
	otherLexical() bool
}

// decimalNumerical is the arm of a finite value, of the fields of codegen's
// PrecisionDecimalValueNumerical exactly, in name, order and Go type, so that
// the caller's arm converts to it and back: goshape.NumericalFields owns them
// and states each, and TestDecimalNumericalFields holds this struct to it, a
// named struct of codec's own being needed for its armOf method.
type decimalNumerical struct {
	Coefficient big.Int
	Scale       big.Int
	Negative    bool
}

// decimalPositiveInfinity, decimalNegativeInfinity and decimalNotANumber are
// the arms of the special values (precisionDecimal dt-specialvalue), which
// hold no field: their ·scale· is absent, and their ·sign· absent for
// notANumber and given by the infinities.
type (
	decimalPositiveInfinity struct{}
	decimalNegativeInfinity struct{}
	decimalNotANumber       struct{}
)

// armOf returns arms' numerical arm.
func (decimalNumerical) armOf(arms decimalArms) reflect.Type { return arms.numerical }

// armOf returns arms' positiveInfinity arm.
func (decimalPositiveInfinity) armOf(arms decimalArms) reflect.Type { return arms.positiveInfinity }

// armOf returns arms' negativeInfinity arm.
func (decimalNegativeInfinity) armOf(arms decimalArms) reflect.Type { return arms.negativeInfinity }

// armOf returns arms' notANumber arm.
func (decimalNotANumber) armOf(arms decimalArms) reflect.Type { return arms.notANumber }

// lexical returns the Coefficient, E and the Scale negated, after a '-' where
// Negative.
func (n decimalNumerical) lexical() string {
	sign := ""
	if n.Negative {
		sign = "-"
	}
	return sign + n.Coefficient.String() + "E" + new(big.Int).Neg(&n.Scale).String()
}

// lexical returns INF.
func (decimalPositiveInfinity) lexical() string { return "INF" }

// lexical returns -INF.
func (decimalNegativeInfinity) lexical() string { return "-INF" }

// lexical returns NaN.
func (decimalNotANumber) lexical() string { return "NaN" }

// otherLexical reports true: a leading '+' or a leading zero spells any
// numerical value otherwise (precisionDecimal §3.2).
func (decimalNumerical) otherLexical() bool { return true }

// otherLexical reports true: +INF spells positiveInfinity beside INF.
func (decimalPositiveInfinity) otherLexical() bool { return true }

// otherLexical reports false: -INF is negativeInfinity's only lexical.
func (decimalNegativeInfinity) otherLexical() bool { return false }

// otherLexical reports false: NaN is notANumber's only lexical.
func (decimalNotANumber) otherLexical() bool { return false }

// sameDecimal reports whether a and b are one precisionDecimal value as codegen's
// PrecisionDecimal tells values apart (codegen/doc.go "# Simple types"): two
// numerical values of equal Coefficient, Scale and Negative, so that 3 and
// 3.00 differ, and 0 and −0, or two special values of one arm, NaN equal to
// NaN, which the datatype's equality never makes it (precisionDecimal §3.1).
func sameDecimal(a, b decimalValue) bool {
	an, aNumerical := a.(decimalNumerical)
	bn, bNumerical := b.(decimalNumerical)
	if !aNumerical || !bNumerical {
		return reflect.TypeOf(a) == reflect.TypeOf(b)
	}
	return an.Coefficient.Cmp(&bn.Coefficient) == 0 && an.Scale.Cmp(&bn.Scale) == 0 && an.Negative == bn.Negative
}

// decimalSpecials are the special values' arms, which hold no field, in the
// order of decimalArms' roles.
var decimalSpecials = [...]decimalValue{decimalPositiveInfinity{}, decimalNegativeInfinity{}, decimalNotANumber{}}

// decimalValueType is decimalValue's reflect.Type, the bare Go type of
// atomicPrecisionDecimal.
var decimalValueType = reflect.TypeFor[decimalValue]()

// convertDecimal returns, as a value of decimalValueType, the precisionDecimal
// value lexical denotes by ·precisionDecimalLexicalMap· (precisionDecimal §6
// f-precDecLexmap), or the error for a lexical that is no pDecimalRep (§3.2
// nt-precDecRep): INF and +INF are positiveInfinity, -INF negativeInfinity
// and NaN notANumber, those spellings only; any other is a numeral of
// digits D with n of them after an optional '.', and an optional exponent E,
// whose ·scale· is n − E (·decimalPtPrecision·, ·scientificPrecision·),
// whose Coefficient is D read as a decimal integer, which give the value
// decimalNumerical's doc points to, and whose ·sign· is negative exactly when
// lexical opens with '-', zero included. Scale is unbounded, as ·scale· is.
func convertDecimal(lexical string) (reflect.Value, error) {
	v := reflect.New(decimalValueType).Elem()
	switch lexical {
	case "INF", "+INF":
		v.Set(reflect.ValueOf(decimalPositiveInfinity{}))
		return v, nil
	case "-INF":
		v.Set(reflect.ValueOf(decimalNegativeInfinity{}))
		return v, nil
	case "NaN":
		v.Set(reflect.ValueOf(decimalNotANumber{}))
		return v, nil
	}
	c := &scanner{s: lexical}
	negative := c.lit('-')
	if !negative {
		c.lit('+')
	}
	whole := c.digits()
	var frac string
	if c.lit('.') {
		frac = c.digits()
	}
	if whole == "" && frac == "" {
		return reflect.Value{}, errors.New("it has no digit before its exponent or end (precisionDecimal §3.2 pDecimalRep)")
	}
	var exponent big.Int
	if c.lit('e') || c.lit('E') {
		start := c.i
		if !c.lit('-') {
			c.lit('+')
		}
		if c.digits() == "" {
			return reflect.Value{}, fmt.Errorf("its exponent at byte %d has no digit (precisionDecimal §3.2 pDecimalRep)", start)
		}
		if _, ok := exponent.SetString(strings.TrimPrefix(c.s[start:c.i], "+"), 10); !ok {
			return reflect.Value{}, fmt.Errorf("big.Int does not read its exponent %q", c.s[start:c.i])
		}
	}
	if !c.done() {
		return reflect.Value{}, fmt.Errorf("%q follows the value at byte %d", c.s[c.i:], c.i)
	}
	n := decimalNumerical{Negative: negative}
	if _, ok := n.Coefficient.SetString(whole+frac, 10); !ok {
		return reflect.Value{}, fmt.Errorf("big.Int does not read its digits %q", whole+frac)
	}
	n.Scale.Sub(big.NewInt(int64(len(frac))), &exponent)
	v.Set(reflect.ValueOf(n))
	return v, nil
}

// decimalShape reports whether v is a struct of the shape of codegen's
// PrecisionDecimal, which codegen declares in the caller's package, so that
// it is matched by shape, its name never compared: exactly one field Value,
// not embedded, struct tags aside, whose type is an interface of one
// unexported method, as codegen's sealed interface PrecisionDecimalValue is.
func decimalShape(v reflect.Type) bool {
	return fieldsNamed(v, "Value") && sealedShape(v.Field(0).Type)
}

// checkDecimalArms returns nil when cx holds, for it, the Value interface of
// a PrecisionDecimal at the site site names, the arms codegen emits for it,
// and otherwise the ErrTypeMismatch error Decode's doc states, item 6: no
// WithPrecisionDecimalArms for it, a nil arm, a numerical arm that does not
// convert to decimalNumerical, so not exactly goshape.NumericalFields'
// fields, in that order, none embedded, struct tags aside, a special arm
// that is no struct of no field, or two arms of one type.
func checkDecimalArms(it reflect.Type, site string, cx *checkContext) error {
	arms, ok := cx.decimals.of(it)
	if !ok {
		return fmt.Errorf("%w: no WithPrecisionDecimalArms supplies the arms of %s, the Value interface of %s", ErrTypeMismatch, it, site)
	}
	roles := [...]struct {
		name string
		t    reflect.Type
	}{{"numerical", arms.numerical}, {"positiveInfinity", arms.positiveInfinity}, {"negativeInfinity", arms.negativeInfinity}, {"notANumber", arms.notANumber}}
	for _, r := range roles {
		if r.t == nil {
			return fmt.Errorf("%w: WithPrecisionDecimalArms supplies a nil %s arm of %s", ErrTypeMismatch, r.name, it)
		}
	}
	if arms.numerical.Kind() != reflect.Struct || !arms.numerical.ConvertibleTo(reflect.TypeFor[decimalNumerical]()) {
		return fmt.Errorf("%w: WithPrecisionDecimalArms supplies %s as the numerical arm of %s, where codegen emits a struct of Coefficient big.Int, Scale big.Int and Negative bool", ErrTypeMismatch, arms.numerical, it)
	}
	for i, r := range roles {
		if i > 0 && (r.t.Kind() != reflect.Struct || r.t.NumField() != 0) {
			return fmt.Errorf("%w: WithPrecisionDecimalArms supplies %s as the %s arm of %s, where codegen emits a struct of no field", ErrTypeMismatch, r.t, r.name, it)
		}
		for _, earlier := range roles[:i] {
			if earlier.t == r.t {
				return fmt.Errorf("%w: WithPrecisionDecimalArms supplies %s as both the %s and the %s arm of %s", ErrTypeMismatch, r.t, earlier.name, r.name, it)
			}
		}
	}
	return nil
}

// decimal returns the value of t, a struct of codegen's PrecisionDecimal
// shape (decimalShape) or a slice of one, that v holds: v a decimalValue
// value, or a slice of them for a list. Its Value field holds the arm of the
// set p.decimals supplies for its interface whose role v's arm has, the
// numerical arm converted from decimalNumerical. Item 6 held the arms first
// (checkDecimalArms), so an interface with none is an internal bug.
func (p *pipeline) decimal(v reflect.Value, t reflect.Type) reflect.Value {
	if t.Kind() == reflect.Slice {
		out := reflect.MakeSlice(t, v.Len(), v.Len())
		for i := range v.Len() {
			out.Index(i).Set(p.decimal(v.Index(i), t.Elem()))
		}
		return out
	}
	out := reflect.New(t).Elem()
	field := out.Field(0)
	arms, ok := p.decimals.of(field.Type())
	if !ok {
		panic(fmt.Sprintf("codec: no arms of %s reached the fill", field.Type()))
	}
	dv := v.Interface().(decimalValue)
	field.Set(reflect.ValueOf(dv).Convert(dv.armOf(arms)))
	return out
}
