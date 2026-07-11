package xsd

import "strconv"

// This file holds the sealed scalar enum types for the closed value sets of
// the XSD 1.1 component model (Structures §2.2). Each is a uint8 whose zero
// value is invalid — an unset field is a caught bug (STYLE T1/T7), never a
// valid instance. Constants start at iota+1 and carry the verbatim spec token
// they denote; String() returns that exact token via a switch (no package-level
// map — a map beside the switch would be a duplicate table, STYLE D3) and never
// panics, because String() is reached from logging and error-formatting paths.
// Names are type-prefixed to avoid collisions as the xsd namespace grows.

// AttributeUse is the value of the lexical use XML attribute of an
// <attribute> in a complex type (§3.2.2). Legal tokens: "optional",
// "prohibited", "required". The zero value is invalid (see builtin.Ordered).
//
// This models the XML-representation attribute only. It is NOT the Attribute
// Use component's {required} boolean property (§3.5.1), which is a distinct
// fact derived from this attribute at parse time and lands in a later issue;
// do not conflate the two.
type AttributeUse uint8

// The AttributeUse values.
const (
	// AttributeUseOptional is the "optional" token (§3.2.2).
	AttributeUseOptional AttributeUse = iota + 1
	// AttributeUseProhibited is the "prohibited" token (§3.2.2).
	AttributeUseProhibited
	// AttributeUseRequired is the "required" token (§3.2.2).
	AttributeUseRequired
)

// String returns the verbatim §3.2.2 token, or a diagnostic form for an
// invalid value (never panics).
func (u AttributeUse) String() string {
	switch u {
	case AttributeUseOptional:
		return "optional"
	case AttributeUseProhibited:
		return "prohibited"
	case AttributeUseRequired:
		return "required"
	default:
		return "AttributeUse(" + strconv.Itoa(int(u)) + ")"
	}
}

// ContentTypeVariety is the {variety} property of a Content Type (§3.4.1).
// Legal tokens: "empty", "simple", "element-only", "mixed". The zero value is
// invalid (see builtin.Ordered).
type ContentTypeVariety uint8

// The ContentTypeVariety values.
const (
	// ContentEmpty is the "empty" token (§3.4.1).
	ContentEmpty ContentTypeVariety = iota + 1
	// ContentSimple is the "simple" token (§3.4.1).
	ContentSimple
	// ContentElementOnly is the "element-only" token (§3.4.1), hyphenated
	// exactly as the spec spells it.
	ContentElementOnly
	// ContentMixed is the "mixed" token (§3.4.1).
	ContentMixed
)

// String returns the verbatim §3.4.1 token, or a diagnostic form for an
// invalid value (never panics).
func (v ContentTypeVariety) String() string {
	switch v {
	case ContentEmpty:
		return "empty"
	case ContentSimple:
		return "simple"
	case ContentElementOnly:
		return "element-only"
	case ContentMixed:
		return "mixed"
	default:
		return "ContentTypeVariety(" + strconv.Itoa(int(v)) + ")"
	}
}

// OpenContentMode is the {mode} property of an Open Content (§3.4.1). Legal
// tokens: "interleave", "suffix". The zero value is invalid (see
// builtin.Ordered).
//
// There is deliberately no "none" member. In the XML representation
// (§3.4.2.3) mode="none" means the Open Content component is ABSENT entirely —
// the enclosing complex type carries a nil Open Content, not a third mode — so
// a "present but mode=none" state stays unrepresentable.
type OpenContentMode uint8

// The OpenContentMode values.
const (
	// OpenContentInterleave is the "interleave" token (§3.4.1).
	OpenContentInterleave OpenContentMode = iota + 1
	// OpenContentSuffix is the "suffix" token (§3.4.1).
	OpenContentSuffix
)

// String returns the verbatim §3.4.1 token, or a diagnostic form for an
// invalid value (never panics).
func (m OpenContentMode) String() string {
	switch m {
	case OpenContentInterleave:
		return "interleave"
	case OpenContentSuffix:
		return "suffix"
	default:
		return "OpenContentMode(" + strconv.Itoa(int(m)) + ")"
	}
}

// Compositor is the {compositor} property of a Model Group (§3.8.1). Legal
// tokens: "all", "choice", "sequence". The zero value is invalid (see
// builtin.Ordered).
type Compositor uint8

// The Compositor values.
const (
	// CompositorAll is the "all" token (§3.8.1).
	CompositorAll Compositor = iota + 1
	// CompositorChoice is the "choice" token (§3.8.1).
	CompositorChoice
	// CompositorSequence is the "sequence" token (§3.8.1).
	CompositorSequence
)

// String returns the verbatim §3.8.1 token, or a diagnostic form for an
// invalid value (never panics).
func (c Compositor) String() string {
	switch c {
	case CompositorAll:
		return "all"
	case CompositorChoice:
		return "choice"
	case CompositorSequence:
		return "sequence"
	default:
		return "Compositor(" + strconv.Itoa(int(c)) + ")"
	}
}

// DerivationMethod is the shared 5-token derivation vocabulary used across
// several component properties: complex-type {derivation method} and {final}/
// {prohibited substitutions} (§3.4.1), element {disallowed substitutions} and
// {substitution group exclusions} (§3.3.1), and simple-type {final}
// (xmlschema11-2.md §4.1.1). Legal tokens: "extension", "restriction",
// "substitution", "list", "union". The zero value is invalid (see
// builtin.Ordered).
//
// This is only the shared vocabulary. It does NOT assert that all five values
// are legal in every consuming context: each consuming property admits its own
// subset (e.g. complex-type {derivation method} is 2-valued, element
// {disallowed substitutions} is a 3-valued subset, simple-type {final} is a
// 4-valued subset), and each validates that subset at construction time. That
// per-property validation is not built here — no consumer exists yet.
type DerivationMethod uint8

// The DerivationMethod values.
const (
	// DerivationExtension is the "extension" token (§3.4.1).
	DerivationExtension DerivationMethod = iota + 1
	// DerivationRestriction is the "restriction" token (§3.4.1,
	// xmlschema11-2.md §4.1.1).
	DerivationRestriction
	// DerivationSubstitution is the "substitution" token (§3.3.1).
	DerivationSubstitution
	// DerivationList is the "list" token (xmlschema11-2.md §4.1.1).
	DerivationList
	// DerivationUnion is the "union" token (xmlschema11-2.md §4.1.1).
	DerivationUnion
)

// String returns the verbatim derivation token, or a diagnostic form for an
// invalid value (never panics).
func (d DerivationMethod) String() string {
	switch d {
	case DerivationExtension:
		return "extension"
	case DerivationRestriction:
		return "restriction"
	case DerivationSubstitution:
		return "substitution"
	case DerivationList:
		return "list"
	case DerivationUnion:
		return "union"
	default:
		return "DerivationMethod(" + strconv.Itoa(int(d)) + ")"
	}
}

// ProcessContents is the {process contents} property of a Wildcard (§3.10.1).
// Legal tokens: "skip", "strict", "lax". The zero value is invalid (see
// builtin.Ordered).
type ProcessContents uint8

// The ProcessContents values.
const (
	// ProcessSkip is the "skip" token (§3.10.1).
	ProcessSkip ProcessContents = iota + 1
	// ProcessStrict is the "strict" token (§3.10.1).
	ProcessStrict
	// ProcessLax is the "lax" token (§3.10.1).
	ProcessLax
)

// String returns the verbatim §3.10.1 token, or a diagnostic form for an
// invalid value (never panics).
func (p ProcessContents) String() string {
	switch p {
	case ProcessSkip:
		return "skip"
	case ProcessStrict:
		return "strict"
	case ProcessLax:
		return "lax"
	default:
		return "ProcessContents(" + strconv.Itoa(int(p)) + ")"
	}
}

// ValueConstraintKind is the {variety} property of a Value Constraint,
// identical across the three components that carry one: attribute declaration
// (§3.2.1), element declaration (§3.3.1), and attribute use (§3.5.1). Legal
// tokens: "default", "fixed". The zero value is invalid (see builtin.Ordered).
//
// It is named "Kind" rather than "Variety" on purpose: "variety" is a spec
// term of art reserved for the simple-type variety (atomic/list/union,
// §2.4 / xmlschema11-2.md §4.1.1), an unrelated concept, and reusing that word
// here would invite confusion.
type ValueConstraintKind uint8

// The ValueConstraintKind values.
const (
	// ValueDefault is the "default" token (§3.2.1; identical in §3.3.1, §3.5.1).
	ValueDefault ValueConstraintKind = iota + 1
	// ValueFixed is the "fixed" token (§3.2.1; identical in §3.3.1, §3.5.1).
	ValueFixed
)

// String returns the verbatim §3.2.1 token, or a diagnostic form for an
// invalid value (never panics).
func (k ValueConstraintKind) String() string {
	switch k {
	case ValueDefault:
		return "default"
	case ValueFixed:
		return "fixed"
	default:
		return "ValueConstraintKind(" + strconv.Itoa(int(k)) + ")"
	}
}

// IdentityConstraintCategory is the {identity-constraint category} property
// of an Identity-Constraint Definition (§3.11.1). Legal tokens: "key",
// "keyref", "unique". The zero value is invalid (see builtin.Ordered).
type IdentityConstraintCategory uint8

// The IdentityConstraintCategory values.
const (
	// IdentityConstraintKey is the "key" token (§3.11.1).
	IdentityConstraintKey IdentityConstraintCategory = iota + 1
	// IdentityConstraintKeyref is the "keyref" token (§3.11.1).
	IdentityConstraintKeyref
	// IdentityConstraintUnique is the "unique" token (§3.11.1).
	IdentityConstraintUnique
)

// String returns the verbatim §3.11.1 token, or a diagnostic form for an
// invalid value (never panics).
func (c IdentityConstraintCategory) String() string {
	switch c {
	case IdentityConstraintKey:
		return "key"
	case IdentityConstraintKeyref:
		return "keyref"
	case IdentityConstraintUnique:
		return "unique"
	default:
		return "IdentityConstraintCategory(" + strconv.Itoa(int(c)) + ")"
	}
}
