package codec

import (
	"bytes"
	"math"
	"math/big"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/xsd"
)

// temporalAtomics are the twelve date, time and duration atomics form spells.
var temporalAtomics = []atomic{
	atomicDateTime, atomicDateTimeStamp, atomicTime, atomicDate, atomicGYearMonth, atomicGYear,
	atomicGMonthDay, atomicGDay, atomicGMonth, atomicDuration, atomicDayTimeDuration, atomicYearMonthDuration,
}

// randomTemporal returns a random value of a's bare type in a's value space:
// a year mostly within four digits, sometimes beyond 9999 or below -9999;
// each property within its §D.2.1 range, a day within its month's; a second
// of up to three fraction digits; a timezoneOffset absent a third of the time
// where a's may be; and a duration's months and seconds of one sign.
func randomTemporal(rng *rand.Rand, a atomic) reflect.Value {
	year := big.NewInt(rng.Int64N(20001) - 10000)
	if rng.IntN(10) == 0 {
		year.Mul(year, big.NewInt(rng.Int64N(1_000_000_000)+1))
	}
	month := uint8(rng.IntN(12) + 1)
	second := big.NewRat(rng.Int64N(60_000), []int64{1, 10, 100, 1000}[rng.IntN(4)])
	for second.Cmp(sixty) >= 0 {
		second.Sub(second, sixty)
	}
	off := int16(rng.IntN(1681) - 840)
	if rng.IntN(4) == 0 {
		off = 0
	}
	var offset *int16
	if rng.IntN(3) != 0 {
		offset = &off
	}
	sign := int64(1)
	if rng.IntN(2) == 0 {
		sign = -1
	}
	months := big.NewInt(sign * rng.Int64N(300))
	seconds := big.NewRat(sign*rng.Int64N(400_000_000), []int64{1, 2, 4, 10, 1000}[rng.IntN(5)])
	switch rng.IntN(5) {
	case 0:
		months.SetInt64(0)
	case 1:
		seconds.SetInt64(0)
	}
	fields := map[string]any{
		"Year": *year, "Month": month, "Hour": uint8(rng.IntN(24)), "Minute": uint8(rng.IntN(60)), "Second": *second,
		"TimezoneOffset": offset, "Months": *months, "Seconds": *seconds,
	}
	if a == atomicDateTimeStamp {
		fields["TimezoneOffset"] = off
	}
	var y *big.Int
	if a != atomicGMonthDay {
		y = year
	}
	m := month
	if a == atomicGDay {
		m = 0
	}
	fields["Day"] = uint8(rng.IntN(int(daysInMonth(y, m))) + 1)
	var props []prop
	for _, f := range goshape.Atomic(a).Fields() {
		props = append(props, prop{f.Name, fields[f.Name]})
	}
	return a.build(props...)
}

// TestTemporalFormDifferential is the warden pre-flight's R11 differential
// (#3902): for 10000 random values of each date, time and duration datatype
// struct, seeded, strict's Canonical of strict's Parse of form's spelling is
// that spelling, and convertTemporal reads it back as the value; the zero
// yearMonthDuration, which strict's Canonical refuses (§3.4.26.1's note), is
// checked by the read back alone.
func TestTemporalFormDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(3902, 1))
	backend := strict.New()
	for _, a := range temporalAtomics {
		datatype, _ := goshape.Atomic(a).Datatype()
		m, ok := backend.Mapping(datatype)
		if !ok {
			t.Fatalf("strict maps no %s", datatype.Display())
		}
		t.Run(datatype.Local, func(t *testing.T) {
			for range 10000 {
				v := randomTemporal(rng, a)
				form, err := a.form(v)
				if err != nil {
					t.Fatalf("form(%s) = %v", spellValue(v), err)
				}
				back, err := convertTemporal(a, form)
				if err != nil || spellValue(back) != spellValue(v) {
					t.Fatalf("form(%s) = %q, which convertTemporal reads as %v, %v", spellValue(v), form, spellValue(back), err)
				}
				if a == atomicYearMonthDuration && form == "P0M" {
					continue
				}
				parsed, err := m.Parse(form, nil)
				if err != nil {
					t.Fatalf("form(%s) = %q, which strict's Parse rejects: %v", spellValue(v), form, err)
				}
				c, err := m.Canonical(parsed)
				if err != nil || c != form {
					t.Fatalf("form(%s) = %q, which strict's Canonical renders %q, %v", spellValue(v), form, c, err)
				}
			}
		})
	}
}

// TestBinaryFormDifferential is the randomized differential R9 of #3935's
// warden pre-flight owes for the binary atomics: for 10000 random []byte
// values of up to 40 octets, seeded, the empty one among them, strict's
// Canonical of strict's Parse of form's spelling under xs:hexBinary and
// xs:base64Binary is that spelling, and convert reads it back as the value.
func TestBinaryFormDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(3935, 1))
	backend := strict.New()
	for _, row := range []struct {
		a     atomic
		local string
	}{{atomicHexBinary, "hexBinary"}, {atomicBase64Binary, "base64Binary"}} {
		a, datatype := row.a, xsd.QName{Space: xsd.XMLSchemaNS, Local: row.local}
		m, ok := backend.Mapping(datatype)
		if !ok {
			t.Fatalf("strict maps no %s", datatype.Display())
		}
		t.Run(datatype.Local, func(t *testing.T) {
			for range 10000 {
				b := make([]byte, rng.IntN(41))
				for i := range b {
					b[i] = byte(rng.UintN(256))
				}
				form, err := a.form(reflect.ValueOf(b))
				if err != nil {
					t.Fatalf("form(%x) = %v", b, err)
				}
				back, err := a.convert(form)
				if err != nil || !bytes.Equal(back.Bytes(), b) {
					t.Fatalf("form(%x) = %q, which convert reads as %x, %v", b, form, back.Bytes(), err)
				}
				parsed, err := m.Parse(form, nil)
				if err != nil {
					t.Fatalf("form(%x) = %q, which strict's Parse rejects: %v", b, form, err)
				}
				c, err := m.Canonical(parsed)
				if err != nil || c != form {
					t.Fatalf("form(%x) = %q, which strict's Canonical renders %q, %v", b, form, c, err)
				}
			}
		})
	}
}

// TestFloatDoubleDifferential is the randomized differential R9 of #3935's
// warden pre-flight owes for xs:float: for 10000 random float32 bit
// patterns, seeded, NaNs, infinities, zeros and subnormals among them,
// double at 32 bits under xs:float on strict's backend records no item 6 or
// 7 error and returns a text convert reads back as the same float32 by its
// bits, every NaN equal to every NaN.
func TestFloatDoubleDifferential(t *testing.T) {
	schema := schemaOf(t, ``)
	var st *xsd.SimpleType
	for _, td := range schema.Types() {
		if s, ok := td.(*xsd.SimpleType); ok && td.Name() == (xsd.QName{Space: xsd.XMLSchemaNS, Local: "float"}) {
			st = s
		}
	}
	if st == nil {
		t.Fatal("no xs:float among the schema's types")
	}
	rng := rand.New(rand.NewPCG(3935, 2))
	for range 10000 {
		f := math.Float32frombits(rng.Uint32())
		e := &encoder{schema: schema, backend: strict.New()}
		c, ok, err := e.double(st, float64(f), 32, "f", "xs:float")
		if err != nil || !ok || e.invalid != nil || e.declined != nil {
			t.Fatalf("double(%x) = %q, %v, %v; recorded %v, %v", math.Float32bits(f), c, ok, err, e.invalid, e.declined)
		}
		back, err := atomicFloat32.convert(c)
		if err != nil {
			t.Fatalf("double(%x) = %q, which convert rejects: %v", math.Float32bits(f), c, err)
		}
		g := float32(back.Float())
		if math.Float32bits(g) != math.Float32bits(f) && !(math.IsNaN(float64(f)) && math.IsNaN(float64(g))) {
			t.Fatalf("double(%x) = %q, which convert reads as %x", math.Float32bits(f), c, math.Float32bits(g))
		}
	}
}

// TestFormArms is the bar R7 of #3902's warden pre-flight sets in place of a
// lint over form's and otherLexicalOf's switches: every atomic builtin of a
// parsed schema has its primitive in encodedPrimitives, which a type of a
// primitive xsd.NewPrimitiveType mints alone lies outside, and the atomic
// goshape.AtomicType gives it answers otherLexicalOf without a panic, and
// form too, with no error, on an in-range value (zero but for Month and Day,
// 1), where formed holds for the atomic.
func TestFormArms(t *testing.T) {
	schema := schemaOf(t, ``)
	seen := 0
	for _, td := range schema.Types() {
		st, ok := td.(*xsd.SimpleType)
		if !ok || td.Name().Space != xsd.XMLSchemaNS || st.IsSpecial() {
			continue
		}
		variety, err := st.Variety(schema)
		if err != nil {
			t.Fatal(err)
		}
		if _, atomicVariety := variety.(xsd.Atomic); !atomicVariety {
			continue
		}
		primitive, err := st.Primitive(schema)
		if err != nil {
			t.Fatal(err)
		}
		if primitive == nil || !slices.Contains(encodedPrimitives, primitive.Name()) {
			t.Errorf("the primitive of the atomic builtin %s is no member of encodedPrimitives", td.Name().Display())
			continue
		}
		g, ok, err := goshape.AtomicType(schema, st)
		if err != nil || !ok {
			t.Fatalf("goshape.AtomicType(%s) = %v, %v, %v", td.Name().Display(), g, ok, err)
		}
		a := atomic(g)
		seen++
		t.Run(td.Name().Local, func(t *testing.T) {
			v := reflect.New(a.bare()).Elem()
			if a == atomicPrecisionDecimal {
				// The nil decimalValue has no method to call.
				v.Set(reflect.ValueOf(decimalNumerical{}))
			}
			for _, name := range []string{"Month", "Day"} {
				if v.Kind() == reflect.Struct {
					if f := v.FieldByName(name); f.IsValid() {
						f.SetUint(1)
					}
				}
			}
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("atomic %s panics: %v", a.spell(), r)
				}
			}()
			_ = a.otherLexicalOf(v)
			if !a.formed() {
				return
			}
			if _, err := a.form(v); err != nil {
				t.Errorf("form(%s) = %v", spellValue(v), err)
			}
		})
	}
	// The 45 atomic builtins of the twenty primitives: the primitives, nine
	// below xs:string, thirteen below xs:decimal, two below xs:duration and one
	// below xs:dateTime.
	if seen != 45 {
		t.Errorf("%d builtins checked, want 45", seen)
	}
}
