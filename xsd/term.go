package xsd

// Term is the sealed sum of the three kinds of component that can appear as a
// particle's {term} (Structures §3.9 "Term" definition): an Element Declaration
// (§3.3.1), a Wildcard (§3.10.1), or a Model Group (§3.8.1). The spec's
// definitional sentence names exactly "the three kinds of components which can
// appear in particles", so the set is closed. The unexported term marker method
// seals it (STYLE T2/T7, the PRINCIPLES 7 sealed-sum exception), so consumers
// exhaustively switch these three variants and no fourth is representable; it
// mirrors attributeuse.go's AttributeDeclarationOrRef sealed sum.
//
// A Model Group Definition (§3.7.1) is deliberately NOT a Term: only its {model
// group} is (see modelgroupdefinition.go), and a <group ref> resolves to that
// shared Model Group, never to the definition itself.
type Term interface{ term() }

// TermOrRef is the pre-resolution {term} slot of a Particle (Structures §3.9.1):
// either an already-known Term or a deferred QName reference resolved to a live
// component at finalize (#173). It is a sealed sum (STYLE T2/T7, the PRINCIPLES
// 7 sealed-sum exception) mirroring attributeuse.go's AttributeDeclarationOrRef:
// ResolvedTerm, ElementDeclarationRef, and ModelGroupRef are its only
// implementations, sealed by the unexported termOrRef marker method, so
// consumers exhaustively switch exactly these branches.
//
// The split exists because two XML mappings — <element ref="..."> (§3.3.2.4,
// ref.elt.global) and <group ref="..."> (§3.7.2, declare-namedModelGroup) — may
// forward-reference a top-level declaration not yet parsed, so only the ref
// QName is available at shape-construction time. At finalize each ref variant is
// replaced by the live component it names — never persisting as a permanent
// branch: an ElementDeclarationRef by the referenced Element Declaration, a
// ModelGroupRef by the referenced Model Group Definition's {model group} (§3.7.2:
// the referencing particle's {term} IS that shared Model Group component
// directly, not a wrapper). There is no WildcardRef: a wildcard is never
// referenced by QName, only declared inline.
type TermOrRef interface{ termOrRef() }

// ResolvedTerm is the TermOrRef variant wrapping an already-known Term: an
// inline element/wildcard/group declaration whose component is built in the same
// producer call (no deferred reference). The field is read-only by convention;
// do not mutate it after construction.
type ResolvedTerm struct{ Term Term }

// ElementDeclarationRef is the TermOrRef variant for the <element ref="...">
// mapping (§3.3.2.4, ref.elt.global): a pre-resolution QName reference to a
// possibly-forward-referenced top-level Element Declaration, resolved to the
// live component at finalize (#173). The field is read-only by convention; do
// not mutate it after construction.
type ElementDeclarationRef struct{ Name QName }

// ModelGroupRef is the TermOrRef variant for the <group ref="..."> mapping
// (§3.7.2, declare-namedModelGroup): a pre-resolution QName reference to a
// possibly-forward-referenced top-level Model Group Definition. At finalize
// (#173) it is replaced by that definition's {model group} — the referencing
// particle's {term} becomes the shared Model Group component directly (§3.7.2),
// not a wrapper. The field is read-only by convention; do not mutate it after
// construction.
type ModelGroupRef struct{ Name QName }

func (ResolvedTerm) termOrRef()          {}
func (ElementDeclarationRef) termOrRef() {}
func (ModelGroupRef) termOrRef()         {}
