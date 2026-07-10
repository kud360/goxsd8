package xsd

import "testing"

// Each table pins every constant to its exact verbatim spec token, and checks
// that the invalid zero value yields a non-panicking diagnostic string.

func TestAttributeUseString(t *testing.T) {
	cases := []struct {
		u    AttributeUse
		want string
	}{
		{AttributeUseOptional, "optional"},
		{AttributeUseProhibited, "prohibited"},
		{AttributeUseRequired, "required"},
		{0, "AttributeUse(0)"},
		{99, "AttributeUse(99)"},
	}
	for _, c := range cases {
		if got := c.u.String(); got != c.want {
			t.Errorf("AttributeUse(%d).String() = %q, want %q", uint8(c.u), got, c.want)
		}
	}
}

func TestContentTypeVarietyString(t *testing.T) {
	cases := []struct {
		v    ContentTypeVariety
		want string
	}{
		{ContentEmpty, "empty"},
		{ContentSimple, "simple"},
		{ContentElementOnly, "element-only"},
		{ContentMixed, "mixed"},
		{0, "ContentTypeVariety(0)"},
		{99, "ContentTypeVariety(99)"},
	}
	for _, c := range cases {
		if got := c.v.String(); got != c.want {
			t.Errorf("ContentTypeVariety(%d).String() = %q, want %q", uint8(c.v), got, c.want)
		}
	}
}

func TestOpenContentModeString(t *testing.T) {
	cases := []struct {
		m    OpenContentMode
		want string
	}{
		{OpenContentInterleave, "interleave"},
		{OpenContentSuffix, "suffix"},
		{0, "OpenContentMode(0)"},
		{99, "OpenContentMode(99)"},
	}
	for _, c := range cases {
		if got := c.m.String(); got != c.want {
			t.Errorf("OpenContentMode(%d).String() = %q, want %q", uint8(c.m), got, c.want)
		}
	}
}

func TestCompositorString(t *testing.T) {
	cases := []struct {
		c    Compositor
		want string
	}{
		{CompositorAll, "all"},
		{CompositorChoice, "choice"},
		{CompositorSequence, "sequence"},
		{0, "Compositor(0)"},
		{99, "Compositor(99)"},
	}
	for _, c := range cases {
		if got := c.c.String(); got != c.want {
			t.Errorf("Compositor(%d).String() = %q, want %q", uint8(c.c), got, c.want)
		}
	}
}

func TestDerivationMethodString(t *testing.T) {
	cases := []struct {
		d    DerivationMethod
		want string
	}{
		{DerivationExtension, "extension"},
		{DerivationRestriction, "restriction"},
		{DerivationSubstitution, "substitution"},
		{DerivationList, "list"},
		{DerivationUnion, "union"},
		{0, "DerivationMethod(0)"},
		{99, "DerivationMethod(99)"},
	}
	for _, c := range cases {
		if got := c.d.String(); got != c.want {
			t.Errorf("DerivationMethod(%d).String() = %q, want %q", uint8(c.d), got, c.want)
		}
	}
}

func TestProcessContentsString(t *testing.T) {
	cases := []struct {
		p    ProcessContents
		want string
	}{
		{ProcessSkip, "skip"},
		{ProcessStrict, "strict"},
		{ProcessLax, "lax"},
		{0, "ProcessContents(0)"},
		{99, "ProcessContents(99)"},
	}
	for _, c := range cases {
		if got := c.p.String(); got != c.want {
			t.Errorf("ProcessContents(%d).String() = %q, want %q", uint8(c.p), got, c.want)
		}
	}
}

func TestValueConstraintKindString(t *testing.T) {
	cases := []struct {
		k    ValueConstraintKind
		want string
	}{
		{ValueDefault, "default"},
		{ValueFixed, "fixed"},
		{0, "ValueConstraintKind(0)"},
		{99, "ValueConstraintKind(99)"},
	}
	for _, c := range cases {
		if got := c.k.String(); got != c.want {
			t.Errorf("ValueConstraintKind(%d).String() = %q, want %q", uint8(c.k), got, c.want)
		}
	}
}
