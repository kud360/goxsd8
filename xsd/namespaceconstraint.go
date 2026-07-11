package xsd

import (
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// ruleWildcardCorrect is Wildcard Properties Correct (Structures §3.10.6.1,
// id="w-props-correct"): a wildcard's Namespace Constraint property record must
// match the §3.10.1 tableau. NewNamespaceConstraint rejects every state its
// clauses 1-4 forbid at construction time, so an ill-formed record is
// unrepresentable (STYLE T1). Clause 5 (attribute wildcards must not carry the
// sibling keyword) is not reachable here: this value type does not represent
// the defined/sibling keywords at all — see the GAP marker on
// NewNamespaceConstraint.
const ruleWildcardCorrect xsderr.Rule = "w-props-correct"

// NamespaceConstraintVariety is the {variety} property of a Namespace
// Constraint property record (Structures §3.10.1). Legal tokens: "any",
// "enumeration", "not". The zero value is invalid — an unset variety is a
// caught bug (STYLE T1/T7), never a valid record, mirroring the sealed enums
// in closedsets.go.
type NamespaceConstraintVariety uint8

// The NamespaceConstraintVariety values.
const (
	// NamespaceConstraintAny is the "any" token (§3.10.1): the wildcard
	// admits every namespace name, including ·absent·. Its {namespaces} set
	// is empty (w-props-correct clause 3).
	NamespaceConstraintAny NamespaceConstraintVariety = iota + 1
	// NamespaceConstraintEnumeration is the "enumeration" token (§3.10.1):
	// the wildcard admits exactly the members of {namespaces}.
	NamespaceConstraintEnumeration
	// NamespaceConstraintNot is the "not" token (§3.10.1): the wildcard
	// admits every namespace name that is NOT a member of {namespaces}. Its
	// {namespaces} set has at least one member (w-props-correct clause 2).
	NamespaceConstraintNot
)

// String returns the verbatim §3.10.1 token, or a diagnostic form for an
// invalid value (never panics).
func (v NamespaceConstraintVariety) String() string {
	switch v {
	case NamespaceConstraintAny:
		return "any"
	case NamespaceConstraintEnumeration:
		return "enumeration"
	case NamespaceConstraintNot:
		return "not"
	default:
		return "NamespaceConstraintVariety(" + strconv.Itoa(int(v)) + ")"
	}
}

// Namespace is a namespace name: "an xs:anyURI value or the distinguished
// value ·absent·" (Structures §3.10.1, the member type of a Namespace
// Constraint's {namespaces} set).
//
// The zero value IS ·absent· — there is no separate boolean flag. This collapse
// is sound because "" is never a legal *present* namespace name: Namespaces in
// XML forbids an empty namespace name, and xmlns="" denotes absence rather than
// a namespace literally named "", so mapping "" onto ·absent· loses no
// distinction (STYLE D3, one fact one encoding). It mirrors qname.go's
// zero-value discipline, where absence is the Go zero value, never a
// distinguished sentinel string.
//
// Note that this ·absent· namespace name is a different concept from a QName
// with Space=="" (a *present* no-namespace name): AllowsName bridges the latter
// to the former on the namespace-name axis, but they are not the same value —
// see AllowsName.
//
// Namespace is comparable: membership in {namespaces} is a plain == test
// (cvc-wildcard-namespace §3.10.4.3 speaks of identity), so it is usable with
// == and as a map key.
type Namespace struct {
	uri string
}

// absentNamespace is the ·absent· namespace name, kept named for readability at
// the internal bridge sites; it is exactly the zero Namespace and is not part
// of the exported surface.
var absentNamespace = Namespace{}

// NamespaceName returns the Namespace denoting the namespace name uri. It
// normalizes the empty string to ·absent· (the zero Namespace): "" is never a
// legal present namespace name (see the Namespace type doc), so NamespaceName("")
// == absentNamespace is a total-function collapse with no lost distinction.
func NamespaceName(uri string) Namespace {
	return Namespace{uri: uri}
}

// IsAbsent reports whether n is the ·absent· namespace name (the zero value).
func (n Namespace) IsAbsent() bool {
	return n == absentNamespace
}

// URI returns the xs:anyURI namespace name; the second result is false when n
// is ·absent·, in which case the first result is not meaningful.
func (n Namespace) URI() (string, bool) {
	if n == absentNamespace {
		return "", false
	}
	return n.uri, true
}

// NamespaceConstraint is the Namespace Constraint property record (Structures
// §3.10.1): the {namespace constraint} property of a Wildcard. It carries a
// {variety}, a {namespaces} set, and the literal-QName members of
// {disallowed names}. It is an immutable value; construct it only through
// NewNamespaceConstraint, which rejects every state Wildcard Properties Correct
// (§3.10.6.1) forbids, so an ill-formed record is unrepresentable (STYLE T1).
//
// The zero value is NOT a valid constraint (its {variety} is the invalid zero
// NamespaceConstraintVariety); AllowsNamespace/AllowsName on a zero value are
// safe and fail closed (return false), but a real constraint always comes from
// the constructor.
//
// AllowsNamespace (cvc-wildcard-namespace §3.10.4.3) and AllowsName
// (cvc-wildcard-name §3.10.4.2) are the sanctioned interrogation path. Variety
// and Namespaces are exposed for inspection and diagnostics, but callers MUST
// NOT reimplement the allowance algorithm by switching on Variety — the
// any/enumeration/not semantics (especially the not-plus-absent interaction)
// live in AllowsNamespace alone.
type NamespaceConstraint struct {
	variety NamespaceConstraintVariety
	// namespaces is the {namespaces} set in document order, deduplicated by
	// first occurrence. It is never produced by ranging a map (STYLE D2).
	namespaces []Namespace
	// disallowedNames holds ONLY the literal xs:QName members of
	// {disallowed names} (§3.10.1), in document order, deduplicated by first
	// occurrence. The defined/sibling keywords are deliberately not
	// represented — see the GAP marker on NewNamespaceConstraint.
	disallowedNames []QName
}

// NewNamespaceConstraint builds a Namespace Constraint record, rejecting the
// states Wildcard Properties Correct (§3.10.6.1, w-props-correct) forbids:
//
//   - clause 1: {variety} must be one of the three §3.10.1 tokens (a zero or
//     out-of-range NamespaceConstraintVariety is not a value the tableau
//     admits);
//   - clause 2: a not constraint has at least one {namespaces} member;
//   - clause 3: an any constraint has an empty {namespaces} set;
//   - clause 4: the namespace name of every {disallowed names} QName member is
//     itself allowed by the namespace-name test being constructed (a self-check
//     via AllowsNamespace, §3.10.4.3).
//
// Each violation returns an *xsderr.Error carrying rule w-props-correct at loc.
// The namespaces and disallowedNames slices are copied (the caller's backing
// arrays are never aliased) and deduplicated by first occurrence, preserving
// document order.
//
// GAP(xsd): ##defined/##definedSibling (§3.10.1 cl.5-6, cvc-wildcard §3.10.4.1
// cl.2-3) need the live declaration graph, which this pure leaf package does
// not have; deferred to the M4/M5 validator that owns it. There is therefore no
// parameter, field, or keyword for them, and w-props-correct clause 5
// (attribute wildcards must not carry sibling) is vacuously satisfied.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built wildcard — may
// legitimately pass the zero xsderr.Loc{}.
func NewNamespaceConstraint(loc xsderr.Loc, variety NamespaceConstraintVariety, namespaces []Namespace, disallowedNames []QName) (NamespaceConstraint, error) {
	switch variety {
	case NamespaceConstraintAny, NamespaceConstraintEnumeration, NamespaceConstraintNot:
	default:
		return NamespaceConstraint{}, xsderr.New(ruleWildcardCorrect, loc,
			"namespace constraint {variety} %s is not one of any/enumeration/not (w-props-correct clause 1)", variety)
	}
	if variety == NamespaceConstraintNot && len(namespaces) == 0 {
		return NamespaceConstraint{}, xsderr.New(ruleWildcardCorrect, loc,
			"namespace constraint {variety} not requires at least one {namespaces} member (w-props-correct clause 2)")
	}
	if variety == NamespaceConstraintAny && len(namespaces) != 0 {
		return NamespaceConstraint{}, xsderr.New(ruleWildcardCorrect, loc,
			"namespace constraint {variety} any requires an empty {namespaces} set, got %d member(s) (w-props-correct clause 3)", len(namespaces))
	}
	c := NamespaceConstraint{
		variety:         variety,
		namespaces:      dedupNamespaces(namespaces),
		disallowedNames: dedupQNames(disallowedNames),
	}
	for _, name := range c.disallowedNames {
		// Bridge the QName's namespace-name axis exactly as AllowsName does.
		if c.AllowsNamespace(NamespaceName(name.Space)) {
			continue
		}
		return NamespaceConstraint{}, xsderr.New(ruleWildcardCorrect, loc,
			"namespace name of {disallowed names} member %s is not allowed by the {namespace constraint} (w-props-correct clause 4)", name)
	}
	return c, nil
}

// Variety returns the {variety} property. It is exposed for inspection and
// diagnostics only; to decide whether a name is admitted, call AllowsNamespace
// or AllowsName rather than switching on this value.
func (c NamespaceConstraint) Variety() NamespaceConstraintVariety {
	return c.variety
}

// Namespaces returns the {namespaces} property in document order. It returns a
// copy: mutating the result does not affect c. An empty {namespaces} yields
// nil.
func (c NamespaceConstraint) Namespaces() []Namespace {
	if len(c.namespaces) == 0 {
		return nil
	}
	return append([]Namespace(nil), c.namespaces...)
}

// AllowsNamespace reports whether the namespace name v is ·valid· with respect
// to this constraint, per Wildcard allows Namespace Name (§3.10.4.3,
// cvc-wildcard-namespace):
//
//   - any: always true, for every v including ·absent·;
//   - enumeration: true iff v is a member of {namespaces} (a plain == identity
//     test, since Namespace is comparable);
//   - not: true iff v is NOT a member of {namespaces}.
//
// The target namespace is baked into {namespaces} at construction time; it is
// never a per-call parameter. A ##other wildcard in a schema whose
// targetNamespace is "http://example.com/t" maps (§3.10.2.2) to {variety} = not
// with {namespaces} = { ·absent·, NamespaceName("http://example.com/t") }.
// Given that constraint:
//
//   - AllowsNamespace(NamespaceName("http://example.com/t")) == false — the
//     target namespace is rejected, because it is a member of the not set;
//   - AllowsNamespace(absent) == false, where absent is the zero Namespace{} —
//     unqualified names are rejected, because ·absent· is a member too;
//   - AllowsNamespace(NamespaceName("http://other.example/u")) == true — a
//     third namespace is admitted, because it is not a member.
//
// Absent membership is spelled out on both sides: the zero Namespace{} (equal
// to NamespaceName("")) is ·absent· on the candidate side, and a ·absent·
// member on the constraint side is the identical value, so the == test matches
// them with no translation.
func (c NamespaceConstraint) AllowsNamespace(v Namespace) bool {
	switch c.variety {
	case NamespaceConstraintAny:
		return true
	case NamespaceConstraintEnumeration:
		return c.hasNamespace(v)
	case NamespaceConstraintNot:
		return !c.hasNamespace(v)
	default:
		return false
	}
}

// hasNamespace reports whether v is a member of {namespaces}, by == identity in
// document order (never via a map, STYLE D2/D3).
func (c NamespaceConstraint) hasNamespace(v Namespace) bool {
	for _, n := range c.namespaces {
		if n == v {
			return true
		}
	}
	return false
}

// AllowsName reports whether the expanded name is ·valid· with respect to this
// constraint, per Wildcard allows Expanded Name (§3.10.4.2, cvc-wildcard-name):
// its namespace name must be allowed by AllowsNamespace (clause 1) AND the name
// must not be a member of {disallowed names} (clause 2).
//
// The namespace-name bridge is the single subtle point. A QName carries only a
// present no-namespace / namespaced distinction: name.Space == "" spells a
// *present* no-namespace name, whereas the ·absent· namespace name is the zero
// Namespace{}. These are not the same concept. For the purpose of the
// namespace-name axis of cvc-wildcard-namespace, an unqualified instance item
// (name.Space == "") has namespace name ·absent·, so AllowsName bridges
// name.Space through NamespaceName — which maps "" to absentNamespace — before
// testing it. That bridge is local to this method (there is no exported
// QName→Namespace conversion).
//
// GAP(xsd): this checks only the literal xs:QName members of {disallowed names}.
// The ##defined/##definedSibling keywords require the live declaration graph and
// are deferred to the M4/M5 validator (see NewNamespaceConstraint); their
// exclusion is not applied here.
func (c NamespaceConstraint) AllowsName(name QName) bool {
	if !c.AllowsNamespace(NamespaceName(name.Space)) {
		return false
	}
	for _, d := range c.disallowedNames {
		if d == name {
			return false
		}
	}
	return true
}

// dedupNamespaces copies in, dropping members equal to an earlier one and
// preserving document order. The seen map is a lookup set only; output order is
// the input's, never a map iteration order (STYLE D2). An empty input yields
// nil.
func dedupNamespaces(in []Namespace) []Namespace {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[Namespace]struct{}, len(in))
	out := make([]Namespace, 0, len(in))
	for _, n := range in {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// dedupQNames copies in, dropping members equal to an earlier one and
// preserving document order. The seen map is a lookup set only; output order is
// the input's, never a map iteration order (STYLE D2). An empty input yields
// nil.
func dedupQNames(in []QName) []QName {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[QName]struct{}, len(in))
	out := make([]QName, 0, len(in))
	for _, q := range in {
		if _, ok := seen[q]; ok {
			continue
		}
		seen[q] = struct{}{}
		out = append(out, q)
	}
	return out
}
