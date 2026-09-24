package xsd

import "github.com/kud360/goxsd8/xsderr"

// AttributeUseOrGroupRef is one member of the ATTRIBUTE CONTENT a container
// hands its constructor — an Attribute Group Definition's or a Complex Type
// Definition's own <attribute> and <attributeGroup ref> children, in document
// order. It is the input §3.6.2.1 and §3.4.2.4 clause 2 build {attribute uses}
// from, never the property itself: a <attributeGroup ref> "does not correspond
// to any component as such" (§3.6.2.1), so a finalized component holds no
// member of this sum, and AttributeUses keeps answering []AttributeUse.
//
// It is a sealed sum (STYLE T2) mirroring term.go's TermOrRef:
// ResolvedAttributeUse, AttributeGroupRef and ResolvedAttributeGroup are its
// only implementations, sealed by the unexported attributeUseOrGroupRef marker
// method, so consumers exhaustively switch exactly these three.
//
// ONE ordered slice carries all three arms rather than a parallel slice of group
// names beside the uses, because the interleaving is observable:
// ag-props-correct clause 2 reports the first duplicate expanded name BY INDEX
// into the folded {attribute uses}, and that index depends on where each
// referenced group's uses land among the container's own.
//
// Finalize folds the content into {attribute uses} and {attribute wildcard}
// (attributegroupfold.go): it resolves each AttributeGroupRef by name
// (src-resolve clause 1.4), expands each ResolvedAttributeGroup in place, and
// takes §3.6.2.1's transitive closure.
type AttributeUseOrGroupRef interface{ attributeUseOrGroupRef() }

// ResolvedAttributeUse is the AttributeUseOrGroupRef arm for one <attribute>
// child (§3.2.2.2 dcl.att.local, §3.2.2.3 ref.att.local): an Attribute Use the
// container contributes directly. The field is read-only by convention; do not
// mutate it after construction.
//
// Use is a PRESENT attribute use, built through NewAttributeUse; the zero
// AttributeUse carries no {attribute declaration}, which that constructor
// refuses, and the container constructors reject it here too.
type ResolvedAttributeUse struct{ Use AttributeUse }

// AttributeGroupRef is the AttributeUseOrGroupRef arm for one <attributeGroup
// ref="..."> child (§3.6.2.1): a pre-resolution QName reference to a
// possibly-forward-referenced top-level Attribute Group Definition. Finalize
// resolves it against {attribute group definitions} (src-resolve clause 1.4)
// and folds that definition's own attribute content into the container's. The
// field is read-only by convention; do not mutate it after construction.
//
// Name is a PRESENT reference, never the absent (zero) QName: an
// <attributeGroup> inside a container is always the reference form, and
// attributeGroup/@ref is typed xs:QName. The container constructors reject a
// zero Name on the footing NewParticle rejects a zero ModelGroupRef.
//
// It is a separate type from ModelGroupRef, AttributeDeclarationRef and
// TypeDefinitionRef because each targets a different symbol space (§3.17.6.2
// clauses 1.4, 1.5, 1.2 and 1.1); one shared ref type would make the target kind
// a runtime property (STYLE T1).
type AttributeGroupRef struct{ Name QName }

// ResolvedAttributeGroup is the AttributeUseOrGroupRef arm for the one
// <attributeGroup ref> a name lookup would answer WRONGLY: a redefining
// <attributeGroup>'s self-reference (§4.2.4 src-redefine clause 7.1). There
// src-expredef clause 2 makes the reference mean "the top-level definition item
// of that name … in S2" — the ORIGINAL — which clause 4.1.2 keeps out of
// {attribute group definitions}. Resolving it by name at finalize would land on
// the redefinition itself, a legal §3.6.2.1 self-cycle that silently drops every
// use the original contributes. So the producer builds the original under its
// own document's producer and hands it over by value, as a redefining <group>'s
// self-reference becomes a ResolvedTerm (parser/produce_complex.go's
// produceGroupRefParticle). The field is read-only by convention; do not mutate
// it after construction.
//
// Definition is OWNED by this slot: nothing else holds it, so finalize expands
// it in place wherever this arm stands and needs no cycle guard for it (an owned
// component pre-exists the slot holding it). Its {name} is present, as every
// Attribute Group Definition's is; the container constructors reject a zero
// Definition, the one value without one.
type ResolvedAttributeGroup struct{ Definition AttributeGroupDefinition }

func (ResolvedAttributeUse) attributeUseOrGroupRef()   {}
func (AttributeGroupRef) attributeUseOrGroupRef()      {}
func (ResolvedAttributeGroup) attributeUseOrGroupRef() {}

// checkAttributeContent rejects the four member states no container may hold,
// all charged to xsderr.RuleComponentInvariant as NewParticle charges its
// ref-arm rejections: no numbered ag-props-correct or ct-props-correct clause
// forbids them, because each is a fault of THIS package's representation of the
// source, not of a component the spec describes.
//
//   - a nil member, which is none of the three arms;
//   - a ResolvedAttributeUse wrapping the zero AttributeUse, whose {attribute
//     declaration} is absent although au-props-correct clause 1 makes it
//     Required (NewAttributeUse refuses to build one);
//   - an AttributeGroupRef whose Name has an empty local part — attributeGroup/@ref
//     is typed xs:QName, whose local part is an NCName and never empty (Datatypes
//     §3.3.18). The local part is tested, not Name == QName{}, which would admit
//     QName{Space: "urn:x"} as a reference;
//   - a ResolvedAttributeGroup whose Definition has an absent {name}, which only
//     the zero AttributeGroupDefinition has (NewAttributeGroupDefinition rejects
//     every other route, ag-props-correct clause 1).
//
// It is called from NewAttributeGroupDefinition and from newComplexType, the
// core every complex-type constructor routes through, so each container kind
// validates its members in one place (STYLE T4). owner names the container for
// the message; loc positions the rejection there, since no member retains a
// position of its own.
func checkAttributeContent(loc xsderr.Loc, owner string, content []AttributeUseOrGroupRef) error {
	for i, m := range content {
		switch m := m.(type) {
		case nil:
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s attribute content[%d] is nil, but every member must be a ResolvedAttributeUse, an AttributeGroupRef or a ResolvedAttributeGroup", owner, i)
		case ResolvedAttributeUse:
			if m.Use.attributeDeclaration == nil {
				return xsderr.New(xsderr.RuleComponentInvariant, loc,
					"%s attribute content[%d] is a ResolvedAttributeUse wrapping the zero AttributeUse, whose {attribute declaration} is absent although au-props-correct clause 1 makes it Required; build the use through NewAttributeUse", owner, i)
			}
		case AttributeGroupRef:
			if m.Name.Local == "" {
				return xsderr.New(xsderr.RuleComponentInvariant, loc,
					"%s attribute content[%d] is an <attributeGroup ref> carrying an empty reference: attributeGroup/@ref is typed xs:QName (§3.6.2.1), whose local part is an NCName and so is never empty", owner, i)
			}
		case ResolvedAttributeGroup:
			if m.Definition.name.Local == "" {
				return xsderr.New(xsderr.RuleComponentInvariant, loc,
					"%s attribute content[%d] is a ResolvedAttributeGroup holding an attribute group definition with an absent {name}, which only the zero AttributeGroupDefinition has; build the original through NewAttributeGroupDefinition", owner, i)
			}
		default:
			panic("xsd: checkAttributeContent: non-exhaustive AttributeUseOrGroupRef switch")
		}
	}
	return nil
}

// directAttributeUses returns the ResolvedAttributeUse arms of content, in order:
// the attribute uses a container contributes DIRECTLY. It is what a container
// constructor seeds {attribute uses} with before finalize folds the referenced
// groups in — the pre-finalize under-approximation AttributeUses documents — and
// nil when content has no such arm.
func directAttributeUses(content []AttributeUseOrGroupRef) []AttributeUse {
	var uses []AttributeUse
	for _, m := range content {
		if u, ok := m.(ResolvedAttributeUse); ok {
			uses = append(uses, u.Use)
		}
	}
	return uses
}

// attributeUseMembers lifts an already-folded {attribute uses} into the
// content sum, every member a ResolvedAttributeUse. It is the inverse view
// attributeMembers takes of a container whose content finalize has folded, and
// the shape newCollapsedExtension hands the shared core for a component that is
// born folded.
func attributeUseMembers(uses []AttributeUse) []AttributeUseOrGroupRef {
	if len(uses) == 0 {
		return nil
	}
	members := make([]AttributeUseOrGroupRef, len(uses))
	for i, u := range uses {
		members[i] = ResolvedAttributeUse{Use: u}
	}
	return members
}

// attributeMembers is the attribute content a container holds NOW, in the one
// form every reader of it walks: content while it is unfolded, and otherwise the
// {attribute uses} the fold stored, lifted into the sum. The fold is what sets
// content to nil (attributegroupfold.go), so a nil content means uses is the
// complete property — whether the fold has run, or the container never had
// content at all, in which case uses is nil too.
//
// Reading through this rather than content alone is what lets a FOLDED
// definition be referenced again: a group re-added to a builder from a finalized
// schema carries its closure in uses and nothing in content, and reading content
// alone would fold it in as empty.
func attributeMembers(content []AttributeUseOrGroupRef, uses []AttributeUse) []AttributeUseOrGroupRef {
	if content != nil {
		return content
	}
	return attributeUseMembers(uses)
}
