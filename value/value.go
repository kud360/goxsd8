package value

// Value is the open representation of a typed XSD value. It is deliberately
// not a sealed interface (PRINCIPLES 2): backends bring their own value types.
// What a value can do is discovered through the small capability interfaces in
// this package (STYLE T2), never a type switch over concrete types.
type Value = any

// Ordering is the result of an [Ordered] comparison. XSD value spaces are
// PARTIALLY ordered (Datatypes §4.2.1, rf-ordered: the ordered fundamental
// facet is total, partial, or false) — float/double/duration and the
// date/time family are only partially ordered, so a comparison may yield no
// order at all. Cross-space comparisons and timezone-less vs timezone-aware
// date/times are Incomparable.
//
// A type whose ordered facet is false is not obliged to report Incomparable
// for everything: false means no order is defined for facet purposes, which
// does not preclude an [Eq]-only capability.
type Ordering int

const (
	// Less means the receiver sorts before the argument.
	Less Ordering = iota
	// Equal means the two values compare equal in their shared value space.
	Equal
	// Greater means the receiver sorts after the argument.
	Greater
	// Incomparable means no order relates the two values (different value
	// spaces, or a partial-order gap such as timezone-less vs
	// timezone-aware, or NaN in float/double).
	Incomparable
)

// Eq reports value-space equality (Datatypes §2.2.2, equality). Note that
// equality is not identity: in float/double NaN is not equal to itself, and
// +0 and -0 are equal but not identical — see [Identical].
type Eq interface {
	// Eq reports whether the receiver equals other in their value space.
	Eq(other Value) bool
}

// Ordered is a value that carries a partial order in addition to equality.
// Bounded types (those to which minInclusive/maxExclusive/… apply) implement
// it so bound facets can be checked. Cmp returns [Incomparable] rather than a
// spurious order when the two values do not relate (rf-ordered).
type Ordered interface {
	Eq
	// Cmp orders the receiver against other, yielding [Incomparable] when no
	// order relates them.
	Cmp(other Value) Ordering
}

// Identical reports the "identity" relation, which is distinct from order
// equality (PRINCIPLES 18, Datatypes §2.2.2). Enumeration matching
// (cvc-enumeration-valid, §4.3.5.4) accepts a value that is "equal or
// identical" to a member, so identity must distinguish cases [Eq] cannot:
// NaN is identical to NaN even though it is never equal to itself, and +0 is
// not identical to -0 even though they are equal. A value without this
// capability falls back to order equality for enumeration purposes.
type Identical interface {
	// Identical reports whether the receiver is the identical value to other.
	Identical(other Value) bool
}

// Lengthed is a value whose length the length/minLength/maxLength facets
// measure (Datatypes §4.3.1–§4.3.3).
type Lengthed interface {
	// Len returns the value's length in the units its primitive defines
	// (characters, list items, or octets).
	Len() int
}

// DigitCounted is a value the totalDigits and fractionDigits facets constrain
// (Datatypes §4.3.11–§4.3.12).
type DigitCounted interface {
	// TotalDigits returns the number of significant decimal digits.
	TotalDigits() int
	// FractionDigits returns the number of fractional decimal digits.
	FractionDigits() int
}

// Scaled is a value that retains a decimal scale as part of its identity.
// precisionDecimal keeps its lexical scale in the value: 3, 3.0 and 3.00 are
// distinct, numerically equal values (xsd-precisionDecimal). The special
// values (NaN, ±INF) have no scale and report ok=false.
type Scaled interface {
	// Scale returns the value's scale; ok is false for values (specials) that
	// carry no scale.
	Scale() (scale int, ok bool)
}

// TimezoneAware is a date/time-family value the explicitTimezone facet
// constrains (Datatypes §4.3.15).
type TimezoneAware interface {
	// HasTimezone reports whether the value carries an explicit timezone.
	HasTimezone() bool
}

// Canonical is a value that can render its own canonical lexical form.
//
// Canonical representations are an optional convenience the spec provides
// "for the use of other applications" (Datatypes, dt-canonical-representation);
// they are NOT required for schema processing itself. So this capability is
// exercised by callers that want a normalized string — internally for facet
// deduplication and enumeration comparison, and it MAY be surfaced to users —
// but it is not mandatory infrastructure. A backend whose type has no canonical
// mapping (e.g. QName, see [Mapping]) simply does not implement it, and a nil
// [Mapping.Canonical] is legitimate.
type Canonical interface {
	// Canonical returns the value's canonical lexical representation.
	Canonical() string
}
