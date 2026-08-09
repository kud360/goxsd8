package xsd

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal for the same reason valueconstraintvalid_test's
// are: checkElementDefaultValid runs inside Finalize and is unexported (STYLE T5),
// so every assertion is made on the error Finalize returns. They share that file's
// stubValueSpace, vcOnly, vcSchema and vcLoc rather than growing a second set
// (STYLE T4) — the value-space seam and the "charged at the component's own Loc"
// question are identical on both sides.

// eBad is the {lexical form} vcOnly("7") reports a DECIDED reject for. Every case
// below uses it, so a clause that never consults the value space is visible as an
// ACCEPT of a lexical the simple-typed cases reject.
const eBad = "not a value of str"

// eDecl builds a GLOBAL element declaration named local, typed by the top-level
// type typeName and carrying vc (nil for an absent {value constraint}), at vcLoc
// so a rejection's position can be pinned to the declaration's own.
func eDecl(t *testing.T, local string, typeName QName, vc *ValueConstraint) ElementDeclaration {
	t.Helper()
	return eScoped(t, local, typeName, vc, NewGlobalScope())
}

// eScoped is eDecl with the {scope} chosen by the caller: the reachability test
// needs the same offending declaration in both a global and a local position.
func eScoped(t *testing.T, local string, typeName QName, vc *ValueConstraint, scope Scope) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(vcLoc, uq(local), TypeDefinitionRef{Name: typeName}, nil, scope, vc,
		false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%s): %v", local, err)
	}
	return e
}

// eContent wraps a single child particle as a mixed or element-only {content
// type}. The child carries no {value constraint} of its own, so a rejection in
// these fixtures can only be the declaration under test's.
func eContent(t *testing.T, mixed bool, child Particle) ElementContent {
	t.Helper()
	return dElementContent(t, mixed, uGroup(t, CompositorSequence, child))
}

// eChild is the one local element declaration the content models below hold,
// typed by a top-level simple type so nothing about it is anonymous.
func eChild(t *testing.T) ElementDeclaration { return uLocal(t, uq("child"), uq("str")) }

// TestEPropsCorrectClause2Dispatch pins the case analysis of Element Default
// Valid (Immediate) (§3.3.6.2, cos-valid-default), one row per arm. The offending
// declaration is the same in every row — same {value constraint}, same Loc — so
// what varies is only its {type definition}, which is exactly what the two
// clauses' antecedents test.
//
// Rows 1 and 2 are clause 1, which defers to Simple Default Valid (§3.2.6.2)
// against T itself (T simple) or T.{content type}.{simple type definition} (T
// complex with simple content); both must CONSULT the value space, since that
// predicate is the installed backend's verdict. Rows 3 to 6 are clause 2, which
// is purely structural — mixed-ness and ·emptiable·-ness — and must NOT consult
// it at all: row 3 accepts a lexical rows 1 and 2 reject, which is only correct
// because clause 2 states no requirement on the value.
func TestEPropsCorrectClause2Dispatch(t *testing.T) {
	sc := dPrimitive(t, uq("sc"))
	bad := NewValueConstraint(ValueDefault, eBad)
	for _, tc := range []struct {
		name      string
		typeName  QName
		add       func(*SchemaBuilder)
		wantErr   bool
		wantCalls bool
		wantMsg   []string
	}{
		{
			name:      "clause 1: T is a simple type definition",
			typeName:  uq("str"),
			wantErr:   true,
			wantCalls: true,
			wantMsg:   []string{"e-props-correct clause 2", "cos-valid-simple-default"},
		},
		{
			name:     "clause 1: T is complex with {content type}.{variety} = simple",
			typeName: uq("t"),
			add: func(b *SchemaBuilder) {
				b.AddType(sc)
				b.AddType(dType(t, uq("t"), anyTypeName, SimpleContent{SimpleType: sc}, nil, nil))
			},
			wantErr:   true,
			wantCalls: true,
			wantMsg:   []string{"e-props-correct clause 2", "cos-valid-simple-default"},
		},
		{
			name:     "clause 2: mixed content over an emptiable particle is accepted",
			typeName: uq("t"),
			add: func(b *SchemaBuilder) {
				child := uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: eChild(t)})
				b.AddType(dType(t, uq("t"), anyTypeName, eContent(t, true, child), nil, nil))
			},
		},
		{
			name:     "clause 2.2: mixed content over a non-emptiable particle",
			typeName: uq("t"),
			add: func(b *SchemaBuilder) {
				child := uOne(t, ResolvedTerm{Term: eChild(t)})
				b.AddType(dType(t, uq("t"), anyTypeName, eContent(t, true, child), nil, nil))
			},
			wantErr: true,
			wantMsg: []string{"cos-valid-default clause 2.2", "emptiable"},
		},
		{
			name:     "clause 2.1: element-only content admits no default",
			typeName: uq("t"),
			add: func(b *SchemaBuilder) {
				child := uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: eChild(t)})
				b.AddType(dType(t, uq("t"), anyTypeName, eContent(t, false, child), nil, nil))
			},
			wantErr: true,
			wantMsg: []string{"cos-valid-default clause 2.1", "element-only"},
		},
		{
			name:     "clause 2.1: empty content admits no default",
			typeName: uq("t"),
			add: func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, nil, nil))
			},
			wantErr: true,
			wantMsg: []string{"cos-valid-default clause 2.1", "empty"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vs := vcOnly("7")
			_, err := vcSchema(t, vs, func(b *SchemaBuilder) {
				if tc.add != nil {
					tc.add(b)
				}
				b.AddElement(eDecl(t, "e", tc.typeName, &bad))
			})
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("unexpected rejection: %v", err)
				}
			}
			if tc.wantErr {
				expectRule(t, err, ruleEPropsCorrect)
				var xe *xsderr.Error
				if errors.As(err, &xe) && xe.Loc != vcLoc {
					t.Errorf("charged at %s, want the declaration's own %s", xe.Loc, vcLoc)
				}
				for _, want := range tc.wantMsg {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("message %q does not contain %q", err.Error(), want)
					}
				}
			}
			if got := vs.defaultCalls > 0; got != tc.wantCalls {
				t.Errorf("ValueSpace consulted %d time(s), want consulted=%t", vs.defaultCalls, tc.wantCalls)
			}
		})
	}
}

// TestEPropsCorrectClause2PresenceGate pins the two ways clause 2 must NOT fire,
// mirroring TestPhaseEAPropsCorrectClause2Accepts on the attribute side: a valid
// default passes cleanly, and a declaration with NO {value constraint} does not
// reach cos-valid-default at all. "If E has a ·non-absent· {value constraint}" is
// a presence gate, not a vacuously-satisfied test, so the value space must not
// even be consulted.
func TestEPropsCorrectClause2PresenceGate(t *testing.T) {
	valid := NewValueConstraint(ValueFixed, "7")
	for _, tc := range []struct {
		name      string
		vc        *ValueConstraint
		wantCalls bool
	}{
		{"a valid default passes", &valid, true},
		{"no {value constraint}: the clause is not reached", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vs := vcOnly("7")
			_, err := vcSchema(t, vs, func(b *SchemaBuilder) {
				b.AddElement(eDecl(t, "e", uq("str"), tc.vc))
			})
			if err != nil {
				t.Fatalf("unexpected rejection: %v", err)
			}
			if got := vs.defaultCalls > 0; got != tc.wantCalls {
				t.Errorf("ValueSpace consulted %d time(s), want consulted=%t", vs.defaultCalls, tc.wantCalls)
			}
		})
	}
}

// TestEPropsCorrectClause2ReachesEveryElementSite pins the walk shape, and is the
// test that would fail if clause 2 were charged by ranging s.elements alone. The
// offending declaration is identical in each case, so any site left unwalked shows
// up as a missing rejection.
//
// §3.3.6 charges the constraint against "all element declarations", and only the
// first row is a member of s.elements: a local <element name="…"> becomes a
// sibling Element Declaration inside its particle's {term} and enters no table
// (parser/produce_complex.go), while carrying a {value constraint} the same way a
// global one does. Measured against the W3C suite, 102 of the 242 <element> items
// with default=/fixed= are local.
func TestEPropsCorrectClause2ReachesEveryElementSite(t *testing.T) {
	bad := NewValueConstraint(ValueDefault, eBad)
	local := func(name string) Particle {
		return uOne(t, ResolvedTerm{Term: eScoped(t, name, uq("str"), &bad, uLocalScope(t))})
	}
	for _, tc := range []struct {
		name  string
		build func(*SchemaBuilder)
	}{
		{"a top-level element declaration", func(b *SchemaBuilder) {
			b.AddElement(eDecl(t, "e", uq("str"), &bad))
		}},
		{"a local declaration in a top-level complex type's content model", func(b *SchemaBuilder) {
			b.AddType(dType(t, uq("t"), anyTypeName, eContent(t, false, local("nested")), nil, nil))
		}},
		{"a local declaration inside an inline anonymous complex type", func(b *SchemaBuilder) {
			ct := dType(t, QName{}, anyTypeName, eContent(t, false, local("nested")), nil, nil)
			b.AddElement(dOwnInline(t, uq("outer"), ct, NewGlobalScope()))
		}},
		{"a local declaration in a top-level model group definition", func(b *SchemaBuilder) {
			mgd, err := NewModelGroupDefinition(xsderr.Loc{}, uq("mg"), uGroup(t, CompositorSequence, local("nested")), nil)
			if err != nil {
				t.Fatalf("NewModelGroupDefinition: %v", err)
			}
			b.AddModelGroup(mgd)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := vcSchema(t, vcOnly("7"), tc.build)
			expectRule(t, err, ruleEPropsCorrect)
			var xe *xsderr.Error
			if errors.As(err, &xe) && xe.Loc != vcLoc {
				t.Errorf("charged at %s, want the declaration's own %s", xe.Loc, vcLoc)
			}
		})
	}
}

// TestEPropsCorrectClause2FailsOpen pins the two accepting non-decisions, both of
// which withhold a rejection rather than inventing one: a {type definition} that
// is ABSENT gives cos-valid-default no T to predicate over, and an UNDECIDED
// ValueSpace verdict is the fail-open contract every consumer of checkSimpleDefault
// inherits (PRINCIPLES 20).
func TestEPropsCorrectClause2FailsOpen(t *testing.T) {
	bad := NewValueConstraint(ValueDefault, eBad)
	t.Run("an absent {type definition} is not decidable", func(t *testing.T) {
		vs := vcOnly("7")
		_, err := vcSchema(t, vs, func(b *SchemaBuilder) {
			e, err := NewElementDeclaration(vcLoc, uq("e"), nil, nil, NewGlobalScope(), &bad,
				false, nil, nil, nil, false, nil, nil)
			if err != nil {
				t.Fatalf("NewElementDeclaration: %v", err)
			}
			b.AddElement(e)
		})
		if err != nil {
			t.Fatalf("an absent {type definition} must be skipped, not charged: %v", err)
		}
		if vs.defaultCalls != 0 {
			t.Errorf("the ValueSpace was consulted %d time(s) for a declaration with no type", vs.defaultCalls)
		}
	})
	t.Run("an undecided verdict accepts", func(t *testing.T) {
		_, err := vcSchema(t, &stubValueSpace{undecidedDefault: true}, func(b *SchemaBuilder) {
			b.AddElement(eDecl(t, "e", uq("str"), &bad))
		})
		if err != nil {
			t.Fatalf("an undecided verdict must accept, never reject: %v", err)
		}
	})
}
