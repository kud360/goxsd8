package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin the attribute group fold (attributegroupfold.go): the
// §3.6.2.1 {attribute uses} closure and the §3.6.2.2 {attribute wildcard}
// intersection over <attributeGroup ref>s, taken at Finalize for every
// container that can hold one, and ag-props-correct clause 2 charged over the
// result. They are package-internal because the owned routes — a redefining
// complex type's inline base, a <redefine> original's pairing — are reached
// only through unexported state. The component builders come from
// complexderivation_test.go, attributeusefold_test.go and
// ownedtypefold_test.go (STYLE T4).

// gUse is one ResolvedAttributeUse over a local declaration named local, typed
// by the str primitive oSchema supplies.
func gUse(t *testing.T, local string) AttributeUseOrGroupRef {
	t.Helper()
	return ResolvedAttributeUse{Use: dAttr(t, uq(local), uq("str"))}
}

// gRef is one <attributeGroup ref> naming the group local.
func gRef(local string) AttributeUseOrGroupRef {
	return AttributeGroupRef{Name: uq(local)}
}

// gGroupAt builds an attribute group definition named local at loc over the
// given content and ·local wildcard·.
func gGroupAt(t *testing.T, loc xsderr.Loc, local string, wildcard *Wildcard, content ...AttributeUseOrGroupRef) AttributeGroupDefinition {
	t.Helper()
	g, err := NewAttributeGroupDefinition(loc, uq(local), content, wildcard)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition(%s): %v", local, err)
	}
	return g
}

// gGroup is gGroupAt for a fixture that does not exercise positions.
func gGroup(t *testing.T, local string, wildcard *Wildcard, content ...AttributeUseOrGroupRef) AttributeGroupDefinition {
	t.Helper()
	return gGroupAt(t, xsderr.Loc{}, local, wildcard, content...)
}

// gTypeAt builds a named complex type at loc with no base over the given
// attribute content and ·local wildcard·.
func gTypeAt(t *testing.T, loc xsderr.Loc, local string, wildcard *Wildcard, content ...AttributeUseOrGroupRef) ComplexType {
	t.Helper()
	ct, err := NewComplexType(loc, uq(local), QName{}, nil, DerivationRestriction, false,
		content, nil, wildcard, EmptyContent{}, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%s): %v", local, err)
	}
	return ct
}

// gType is gTypeAt for a fixture that does not exercise positions.
func gType(t *testing.T, local string, wildcard *Wildcard, content ...AttributeUseOrGroupRef) ComplexType {
	t.Helper()
	return gTypeAt(t, xsderr.Loc{}, local, wildcard, content...)
}

// gFinalize finalizes a schema carrying xs:anyType and the str primitive, and
// returns Finalize's own result for a fixture that asserts a rejection.
func gFinalize(t *testing.T, build func(*SchemaBuilder)) (*Schema, error) {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(dPrimitive(t, uq("str")))
	build(b)
	return b.Finalize()
}

// gSchema is gFinalize for a fixture that must finalize.
func gSchema(t *testing.T, build func(*SchemaBuilder)) *Schema {
	t.Helper()
	s, err := gFinalize(t, build)
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// gGroupUses is the expanded local names of the finalized attribute group
// definition local's {attribute uses}, in the order the property holds them.
func gGroupUses(t *testing.T, s *Schema, local string) []string {
	t.Helper()
	g, ok := s.attributeGroupIndex[uq(local)]
	if !ok {
		t.Fatalf("attribute group %s is not in the finalized schema", local)
	}
	var names []string
	for _, u := range g.AttributeUses() {
		names = append(names, u.DeclarationName().Local)
	}
	return names
}

// TestAttributeGroupFoldTransitiveClosure pins §3.6.2.1's transitive closure
// over a three-level chain, into every definition on it and into a complex type
// referencing its head: each container holds its own uses and those of every
// group it reaches, in document order with each reached group's members at the
// position of the reference that reached it. It also pins the pre-Finalize
// reading: the unfolded type holds its direct use alone.
func TestAttributeGroupFoldTransitiveClosure(t *testing.T) {
	var pre ComplexType
	s := gSchema(t, func(b *SchemaBuilder) {
		b.AddAttributeGroup(gGroup(t, "A", nil, gUse(t, "a"), gRef("B")))
		b.AddAttributeGroup(gGroup(t, "B", nil, gRef("C"), gUse(t, "b")))
		b.AddAttributeGroup(gGroup(t, "C", nil, gUse(t, "c")))
		pre = gType(t, "T", nil, gUse(t, "t"), gRef("A"))
		b.AddType(pre)
	})
	if got, want := oUseNames(pre), []string{"t"}; !fEqual(got, want) {
		t.Errorf("unfolded T {attribute uses} = %v, want %v: only the ResolvedAttributeUse members before Finalize", got, want)
	}
	for _, tc := range []struct {
		group string
		want  []string
	}{
		{"A", []string{"a", "c", "b"}},
		{"B", []string{"c", "b"}},
		{"C", []string{"c"}},
	} {
		if got := gGroupUses(t, s, tc.group); !fEqual(got, tc.want) {
			t.Errorf("group %s {attribute uses} = %v, want %v", tc.group, got, tc.want)
		}
	}
	if got, want := fUses(t, s, uq("T")), []string{"t", "a", "c", "b"}; !fEqual(got, want) {
		t.Errorf("T {attribute uses} = %v, want %v", got, want)
	}
}

// TestAttributeGroupFoldCycleIsLegal pins that a reference cycle — which
// §3.6.2.1 permits ("Circular reference is not disallowed") and ag-props-correct
// has no clause against — finalizes, and that every group on it holds the
// attributes of the whole cycle, its own first.
func TestAttributeGroupFoldCycleIsLegal(t *testing.T) {
	s := gSchema(t, func(b *SchemaBuilder) {
		b.AddAttributeGroup(gGroup(t, "A", nil, gUse(t, "x"), gRef("B")))
		b.AddAttributeGroup(gGroup(t, "B", nil, gUse(t, "y"), gRef("A")))
		b.AddAttributeGroup(gGroup(t, "S", nil, gRef("S"), gUse(t, "z")))
		b.AddType(gType(t, "T", nil, gRef("A")))
	})
	if got, want := gGroupUses(t, s, "A"), []string{"x", "y"}; !fEqual(got, want) {
		t.Errorf("group A {attribute uses} = %v, want %v", got, want)
	}
	if got, want := gGroupUses(t, s, "B"), []string{"y", "x"}; !fEqual(got, want) {
		t.Errorf("group B {attribute uses} = %v, want %v", got, want)
	}
	if got, want := gGroupUses(t, s, "S"), []string{"z"}; !fEqual(got, want) {
		t.Errorf("self-referencing group S {attribute uses} = %v, want %v", got, want)
	}
	if got, want := fUses(t, s, uq("T")), []string{"x", "y"}; !fEqual(got, want) {
		t.Errorf("T {attribute uses} = %v, want %v", got, want)
	}
}

// TestAttributeGroupFoldReachesEachGroupOnce pins the closure as a SET union: a
// group reached twice — down a diamond, or named twice as an explicit ref and
// the <schema defaultAttributes> ref the producer synthesizes — contributes its
// uses once, so ct-props-correct clause 4 sees no duplicate the source never
// wrote.
func TestAttributeGroupFoldReachesEachGroupOnce(t *testing.T) {
	s := gSchema(t, func(b *SchemaBuilder) {
		b.AddAttributeGroup(gGroup(t, "B", nil, gRef("D")))
		b.AddAttributeGroup(gGroup(t, "C", nil, gRef("D")))
		b.AddAttributeGroup(gGroup(t, "D", nil, gUse(t, "d")))
		b.AddType(gType(t, "Diamond", nil, gRef("B"), gRef("C")))
		b.AddType(gType(t, "Twice", nil, gRef("D"), gRef("D")))
	})
	if got, want := fUses(t, s, uq("Diamond")), []string{"d"}; !fEqual(got, want) {
		t.Errorf("Diamond {attribute uses} = %v, want %v", got, want)
	}
	if got, want := fUses(t, s, uq("Twice")), []string{"d"}; !fEqual(got, want) {
		t.Errorf("Twice {attribute uses} = %v, want %v", got, want)
	}
}

// TestAttributeGroupFoldExpandsOwnedOriginal pins the ResolvedAttributeGroup
// arm: a redefining <attributeGroup>'s self-reference (src-redefine clause 7.1)
// holds the <redefine>d ORIGINAL by value, and the fold expands it in place —
// following the original's OWN references too — so the redefinition keeps
// every use the original contributes, and so does a type referencing it by
// name. The contrast case is what the arm exists to prevent: the same
// self-reference written as a by-name AttributeGroupRef names the redefinition
// itself, a legal self-cycle that contributes nothing.
func TestAttributeGroupFoldExpandsOwnedOriginal(t *testing.T) {
	s := gSchema(t, func(b *SchemaBuilder) {
		original := gGroup(t, "G", nil, gUse(t, "o1"), gRef("H"), gUse(t, "o2"))
		b.AddAttributeGroup(gGroup(t, "H", nil, gUse(t, "h")))
		b.AddAttributeGroup(gGroup(t, "G", nil, ResolvedAttributeGroup{Definition: original}, gUse(t, "r")))
		b.AddType(gType(t, "T", nil, gRef("G")))
	})
	if got, want := gGroupUses(t, s, "G"), []string{"o1", "h", "o2", "r"}; !fEqual(got, want) {
		t.Errorf("redefining group G {attribute uses} = %v, want %v: the original's uses, its own reference's, then the redefinition's", got, want)
	}
	if got, want := fUses(t, s, uq("T")), []string{"o1", "h", "o2", "r"}; !fEqual(got, want) {
		t.Errorf("T {attribute uses} = %v, want %v", got, want)
	}

	byName := gSchema(t, func(b *SchemaBuilder) {
		b.AddAttributeGroup(gGroup(t, "G", nil, gRef("G"), gUse(t, "r")))
	})
	if got, want := gGroupUses(t, byName, "G"), []string{"r"}; !fEqual(got, want) {
		t.Errorf("by-name self-reference G {attribute uses} = %v, want %v", got, want)
	}
}

// TestAttributeGroupFoldWildcard pins §3.6.2.2 over the closure: the
// {namespace constraint} is the intersection of every ·local wildcard· the
// closure reaches, and {process contents} is the container's own wildcard's
// when it has one and otherwise the first referenced group's in document order.
func TestAttributeGroupFoldWildcard(t *testing.T) {
	nsX, nsY, nsZ := NamespaceName("urn:x"), NamespaceName("urn:y"), NamespaceName("urn:z")
	xy := uWildcard(t, NamespaceConstraintEnumeration, []Namespace{nsX, nsY}, ProcessLax)
	yz := uWildcard(t, NamespaceConstraintEnumeration, []Namespace{nsY, nsZ}, ProcessStrict)
	skip := uWildcard(t, NamespaceConstraintAny, nil, ProcessSkip)
	s := gSchema(t, func(b *SchemaBuilder) {
		b.AddAttributeGroup(gGroup(t, "A", &xy, gRef("B")))
		b.AddAttributeGroup(gGroup(t, "B", &yz))
		b.AddType(gType(t, "RefsOnly", nil, gRef("B"), gRef("A")))
		b.AddType(gType(t, "OwnFirst", &skip, gRef("A")))
		b.AddType(gType(t, "OwnOnly", &skip))
	})
	wildcardOf := func(local string) Wildcard {
		t.Helper()
		def, _ := s.Type(uq(local))
		w, ok := def.(ComplexType).AttributeWildcard()
		if !ok {
			t.Fatalf("%s {attribute wildcard} is absent", local)
		}
		return w
	}
	onlyY := func(what string, w Wildcard) {
		t.Helper()
		nc := w.NamespaceConstraint()
		if !nc.AllowsNamespace(nsY) || nc.AllowsNamespace(nsX) || nc.AllowsNamespace(nsZ) {
			t.Errorf("%s {namespace constraint} = %v, want the intersection admitting urn:y alone", what, nc.Namespaces())
		}
	}
	g := s.attributeGroupIndex[uq("A")]
	gw, ok := g.AttributeWildcard()
	if !ok {
		t.Fatal("group A {attribute wildcard} is absent")
	}
	onlyY("group A", gw)
	if gw.ProcessContents() != ProcessLax {
		t.Errorf("group A {process contents} = %v, want lax from its own <anyAttribute>", gw.ProcessContents())
	}
	refsOnly := wildcardOf("RefsOnly")
	onlyY("RefsOnly", refsOnly)
	if refsOnly.ProcessContents() != ProcessStrict {
		t.Errorf("RefsOnly {process contents} = %v, want strict from B, the first group it references", refsOnly.ProcessContents())
	}
	ownFirst := wildcardOf("OwnFirst")
	onlyY("OwnFirst", ownFirst)
	if ownFirst.ProcessContents() != ProcessSkip {
		t.Errorf("OwnFirst {process contents} = %v, want skip from its own <anyAttribute>", ownFirst.ProcessContents())
	}
	if !wildcardOf("OwnOnly").NamespaceConstraint().AllowsNamespace(nsX) {
		t.Error("OwnOnly {attribute wildcard} lost its own ##any <anyAttribute>")
	}
}

// TestAttributeGroupDuplicateChargedAtFinalize pins ag-props-correct clause 2 at
// its new home: over the FOLDED {attribute uses}, at the definition's own
// position, naming the first duplicate by index. The interleaved fixture is the
// one ordering where the index distinguishes one ordered sum from a parallel
// slice of group names: folded in document order G holds [b y z b], so the
// repeat is at [3]; uses-then-groups would hold [b b y z] and report [1].
func TestAttributeGroupDuplicateChargedAtFinalize(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(*testing.T, *SchemaBuilder)
		want  string
	}{
		{"two direct uses", func(t *testing.T, b *SchemaBuilder) {
			b.AddAttributeGroup(gGroupAt(t, resolveLocAt(41), "G", nil, gUse(t, "a"), gUse(t, "a")))
		}, "{attribute uses}[1] repeats the expanded name " + uq("a").String()},
		{"a referenced group's use interleaved before a direct one", func(t *testing.T, b *SchemaBuilder) {
			b.AddAttributeGroup(gGroup(t, "H", nil, gUse(t, "b"), gUse(t, "y"), gUse(t, "z")))
			b.AddAttributeGroup(gGroupAt(t, resolveLocAt(41), "G", nil, gRef("H"), gUse(t, "b")))
		}, "{attribute uses}[3] repeats the expanded name " + uq("b").String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := gFinalize(t, func(b *SchemaBuilder) { tc.build(t, b) })
			expectRule(t, err, ruleAgPropsCorrect)
			if !strings.Contains(err.Error(), "attribute group definition "+uq("G").String()+" "+tc.want) {
				t.Fatalf("message %q does not name group G and %q", err, tc.want)
			}
			if got, _ := xsderr.LocOf(err); got != resolveLocAt(41) {
				t.Fatalf("charged at %s, want G's own position %s", got, resolveLocAt(41))
			}
		})
	}
}

// TestAttributeGroupRefDanglingChargedAtFinalize pins src-resolve clause 1.4 in
// Phase A: an <attributeGroup ref> naming no definition is rejected at the
// position of the container holding it, whichever kind of container that is.
func TestAttributeGroupRefDanglingChargedAtFinalize(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(*testing.T, *SchemaBuilder)
		owner string
	}{
		{"in a complex type", func(t *testing.T, b *SchemaBuilder) {
			b.AddType(gTypeAt(t, resolveLocAt(51), "T", nil, gRef("missing")))
		}, "complex type " + uq("T").String()},
		{"in an attribute group", func(t *testing.T, b *SchemaBuilder) {
			b.AddAttributeGroup(gGroupAt(t, resolveLocAt(51), "G", nil, gRef("missing")))
		}, "attribute group definition " + uq("G").String()},
		{"in a redefining group's owned original", func(t *testing.T, b *SchemaBuilder) {
			original := gGroupAt(t, resolveLocAt(51), "G", nil, gRef("missing"))
			b.AddAttributeGroup(gGroup(t, "G", nil, ResolvedAttributeGroup{Definition: original}))
		}, "attribute group definition " + uq("G").String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := gFinalize(t, func(b *SchemaBuilder) { tc.build(t, b) })
			expectRule(t, err, ruleSrcResolve)
			want := tc.owner + " references attribute group definition " + uq("missing").String() + " — through an <attributeGroup ref> child, or, on a complex type, the reference §3.4.2.4 synthesizes from <schema defaultAttributes> — but"
			if !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "src-resolve clause 1.4") {
				t.Fatalf("message %q does not contain %q and cite clause 1.4", err, want)
			}
			if got, _ := xsderr.LocOf(err); got != resolveLocAt(51) {
				t.Fatalf("charged at %s, want the referring container's position %s", got, resolveLocAt(51))
			}
		})
	}
}

// TestAttributeGroupFoldReachesOwnedTypes pins the fold's two owned routes for a
// complex type: the src-expredef clause 1.1 ORIGINAL a redefining complex type
// owns in its {base type definition}, which no index reaches, and an anonymous
// type an element declaration owns. Each holds an <attributeGroup ref> the fold
// must consume at the slot owning it.
func TestAttributeGroupFoldReachesOwnedTypes(t *testing.T) {
	s := gSchema(t, func(b *SchemaBuilder) {
		b.AddAttributeGroup(gGroup(t, "G", nil, gUse(t, "g")))
		id := NewComponentID()
		original, err := NewAnonymousComplexType(xsderr.Loc{}, ComplexTypeDefinitionContext{Component: id},
			anyTypeName, nil, DerivationRestriction, false, []AttributeUseOrGroupRef{gRef("G")}, nil, nil, EmptyContent{}, nil, nil)
		if err != nil {
			t.Fatalf("NewAnonymousComplexType(original): %v", err)
		}
		redefined, err := NewComplexTypeOwningBase(xsderr.Loc{}, id, uq("R"), original, nil, DerivationExtension, false,
			[]AttributeUseOrGroupRef{gUse(t, "r")}, nil, nil, EmptyContent{}, nil, nil)
		if err != nil {
			t.Fatalf("NewComplexTypeOwningBase(R): %v", err)
		}
		b.AddType(redefined)
		inline, err := NewAnonymousComplexType(xsderr.Loc{}, ElementDeclarationContext{Component: NewComponentID()},
			anyTypeName, nil, DerivationRestriction, false, []AttributeUseOrGroupRef{gRef("G")}, nil, nil, EmptyContent{}, nil, nil)
		if err != nil {
			t.Fatalf("NewAnonymousComplexType(inline): %v", err)
		}
		b.AddElement(dOwnInline(t, uq("e"), inline, NewGlobalScope()))
	})
	def, _ := s.Type(uq("R"))
	base, owns := ownedComplexType(def.(ComplexType).Base())
	if !owns {
		t.Fatal("R does not own its clause-1.1 original")
	}
	if got, want := oUseNames(base), []string{"g"}; !fEqual(got, want) {
		t.Errorf("R's owned original {attribute uses} = %v, want %v", got, want)
	}
	if got, want := fUses(t, s, uq("R")), []string{"r", "g"}; !fEqual(got, want) {
		t.Errorf("R {attribute uses} = %v, want %v: its own, then the original's inherited under clause 3.1", got, want)
	}
	if got, want := oUseNames(oOwnedType(t, oGlobal(t, s, uq("e")))), []string{"g"}; !fEqual(got, want) {
		t.Errorf("e's inline type {attribute uses} = %v, want %v", got, want)
	}
}

// TestAttributeGroupRedefinitionComparesFoldedOriginal pins
// checkAttributeGroupRedefinitions running AFTER the fold: the <redefine>d
// original reaches attribute a only through its own <attributeGroup ref>, so a
// redefinition declaring a restricts it (src-redefine clause 7.2.2) only once
// the original is folded — against the unfolded original, which holds no use
// and no wildcard, it is rejected. The control keeps the check live: an
// attribute the folded original does not reach is still charged.
func TestAttributeGroupRedefinitionComparesFoldedOriginal(t *testing.T) {
	pair := func(b *SchemaBuilder, redefining ...AttributeUseOrGroupRef) {
		b.AddAttributeGroup(gGroup(t, "H", nil, gUse(t, "a")))
		b.AddRedefiningAttributeGroup(gGroup(t, "G", nil, redefining...), gGroup(t, "G", nil, gRef("H")))
	}
	if _, err := gFinalize(t, func(b *SchemaBuilder) { pair(b, gUse(t, "a")) }); err != nil {
		t.Fatalf("a redefinition restricting the folded original was rejected: %v", err)
	}
	_, err := gFinalize(t, func(b *SchemaBuilder) { pair(b, gUse(t, "a"), gUse(t, "extra")) })
	expectRule(t, err, ruleSrcRedefine)
}

// TestFoldedAttributeGroupReferencedAgain pins attributeMembers' fallback: a
// definition taken from a finalized schema carries its closure in {attribute
// uses} and no content, and a type referencing it in a SECOND schema must fold
// that closure in rather than read the empty content as nothing.
func TestFoldedAttributeGroupReferencedAgain(t *testing.T) {
	first := gSchema(t, func(b *SchemaBuilder) {
		b.AddAttributeGroup(gGroup(t, "A", nil, gUse(t, "a"), gRef("B")))
		b.AddAttributeGroup(gGroup(t, "B", nil, gUse(t, "b")))
	})
	folded := first.attributeGroupIndex[uq("A")]
	second := gSchema(t, func(b *SchemaBuilder) {
		b.AddAttributeGroup(folded)
		b.AddType(gType(t, "T", nil, gRef("A")))
	})
	if got, want := fUses(t, second, uq("T")), []string{"a", "b"}; !fEqual(got, want) {
		t.Errorf("T {attribute uses} = %v, want %v from the already-folded A", got, want)
	}
}

// resolveLocAt is a distinct, non-zero position for fixtures that pin where a
// rejection is charged.
func resolveLocAt(line int) xsderr.Loc {
	return xsderr.Loc{URI: "fold.xsd", Line: line, Col: 3}
}
