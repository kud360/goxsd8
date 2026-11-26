package codec

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/internal/goname"
	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// rnode is one particle of a random content model: an element a, b or c of
// type xs:string, a ##other wildcard (wild), or a sequence or choice of kids,
// occurring min to max times, max -1 for unbounded.
type rnode struct {
	kind     string // "elem", "wild", "sequence" or "choice"
	name     string
	min, max int
	kids     []*rnode
}

// rOccurs are the occurrence ranges a random particle draws from, the
// warden's set {1, 0..1, 1..2, unbounded} with the fixed 2..2 added, under
// which an item opens where a particle has taken its {max occurs}.
var rOccurs = [][2]int{{1, 1}, {0, 1}, {1, 2}, {1, -1}, {2, 2}}

// genModel returns a random particle at depth, wilds counting the wildcards
// the model holds so far, at most one. A nested model group occurs exactly
// once half the time, the occurrence Decode binds most nested groups in.
func genModel(rng *rand.Rand, depth int, wilds *int) *rnode {
	occ := rOccurs[rng.IntN(len(rOccurs))]
	if depth == 0 || depth < 3 && rng.IntN(10) < 3 {
		if depth > 0 && rng.IntN(2) == 0 {
			occ = rOccurs[0]
		}
		n := &rnode{kind: [2]string{"sequence", "choice"}[rng.IntN(2)], min: occ[0], max: occ[1]}
		for range 1 + rng.IntN(3) {
			n.kids = append(n.kids, genModel(rng, depth+1, wilds))
		}
		return n
	}
	if *wilds == 0 && rng.IntN(6) == 0 {
		*wilds++
		return &rnode{kind: "wild", min: occ[0], max: occ[1]}
	}
	return &rnode{kind: "elem", name: string(rune('a' + rng.IntN(3))), min: occ[0], max: occ[1]}
}

// xsdOf renders n as a schema document's particle.
func (n *rnode) xsdOf() string {
	occurs := fmt.Sprintf(` minOccurs="%d" maxOccurs="%s"`, n.min, maxString(n.max))
	switch n.kind {
	case "elem":
		return `<xs:element name="` + n.name + `" type="xs:string"` + occurs + `/>`
	case "wild":
		return `<xs:any namespace="##other" processContents="skip"` + occurs + `/>`
	}
	var b strings.Builder
	for _, k := range n.kids {
		b.WriteString(k.xsdOf())
	}
	return "<xs:" + n.kind + occurs + ">" + b.String() + "</xs:" + n.kind + ">"
}

// namesOf returns the child names of leaves, a sequence genInstance draws:
// each element particle's name, "w" for the wildcard's child.
func namesOf(leaves []*rnode) []string {
	names := make([]string, len(leaves))
	for i, l := range leaves {
		names[i] = l.name
		if l.kind == "wild" {
			names[i] = "w"
		}
	}
	return names
}

// maxString renders max as a maxOccurs value.
func maxString(m int) string {
	if m < 0 {
		return "unbounded"
	}
	return strconv.Itoa(m)
}

// genInstance appends to out, one per child, the element and wildcard
// particles of a random sequence n's particle matches, its repetitions capped
// at min+2.
func (n *rnode) genInstance(rng *rand.Rand, out []*rnode) []*rnode {
	upper := n.min + 2
	if n.max >= 0 && n.max < upper {
		upper = n.max
	}
	for range n.min + rng.IntN(upper-n.min+1) {
		out = n.genOnce(rng, out)
	}
	return out
}

// genOnce appends the particles of one random occurrence of n's term.
func (n *rnode) genOnce(rng *rand.Rand, out []*rnode) []*rnode {
	switch n.kind {
	case "elem", "wild":
		return append(out, n)
	case "choice":
		return n.kids[rng.IntN(len(n.kids))].genInstance(rng, out)
	}
	for _, k := range n.kids {
		out = k.genInstance(rng, out)
	}
	return out
}

// onceEnds returns every j such that xs[i:j] matches one occurrence of n's
// term, ascending: the brute-force oracle, independent of xsd's Matcher.
func (n *rnode) onceEnds(xs []string, i int) []int {
	switch n.kind {
	case "elem", "wild":
		want := n.name
		if n.kind == "wild" {
			want = "w"
		}
		if i < len(xs) && xs[i] == want {
			return []int{i + 1}
		}
		return nil
	case "choice":
		var ends []int
		for _, k := range n.kids {
			ends = union(ends, k.particleEnds(xs, i))
		}
		return ends
	}
	ends := []int{i}
	for _, k := range n.kids {
		var next []int
		for _, e := range ends {
			next = union(next, k.particleEnds(xs, e))
		}
		ends = next
	}
	return ends
}

// particleEnds returns every j such that xs[i:j] matches n's particle, from
// {min occurs} to {max occurs} occurrences of its term.
func (n *rnode) particleEnds(xs []string, i int) []int {
	// Past {min occurs}, an occurrence beyond the remaining children's count
	// adds no end, so that count bounds an unbounded particle's.
	limit := n.max
	if limit < 0 {
		limit = n.min + len(xs) - i + 1
	}
	var ends []int
	cur := []int{i}
	for k := 0; k <= limit; k++ {
		if k >= n.min {
			ends = union(ends, cur)
		}
		var next []int
		for _, e := range cur {
			next = union(next, n.onceEnds(xs, e))
		}
		if len(next) == 0 {
			break
		}
		cur = next
	}
	return ends
}

// union returns the sorted union of a and b.
func union(a, b []int) []int {
	out := slices.Concat(a, b)
	slices.Sort(out)
	return slices.Compact(out)
}

// partitions returns every partition of xs into non-empty sub-sequences each
// matching one occurrence of n's term (cvc-accept clause 3.3), whose count,
// with as many empty occurrences as n's term admits added, can meet clauses
// 3.1 and 3.2: at most {max occurs} non-empty ones, and at least {min occurs}
// in all, empty ones counted only where one occurrence can match nothing.
// Each partition is the start index of each sub-sequence.
func (n *rnode) partitions(xs []string) [][]int {
	emptiable := slices.Contains(n.onceEnds(xs[:0], 0), 0)
	var out [][]int
	var walk func(i int, starts []int)
	walk = func(i int, starts []int) {
		if i == len(xs) {
			k := len(starts)
			if (n.max < 0 || k <= n.max) && (k >= n.min || emptiable) {
				out = append(out, slices.Clone(starts))
			}
			return
		}
		for _, e := range n.onceEnds(xs, i) {
			if e > i {
				walk(e, append(starts, i))
			}
		}
	}
	walk(0, nil)
	return out
}

// randomSchema parses decls as schemaOf does, returning false where the
// parser rejects them.
func randomSchema(decls string) (*xsd.Schema, bool) {
	doc := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="urn:s" targetNamespace="urn:s" elementFormDefault="qualified">` + decls + `</xs:schema>`
	resolve := loader.ResolverFunc(func(_, location string) (io.ReadCloser, string, error) {
		return io.NopCloser(strings.NewReader(doc)), location, nil
	})
	schema, err := parser.Parse("s.xsd", parser.WithResolver(resolve))
	return schema, err == nil
}

// TestOccurrencePartitionDifferential holds partition, the rule Decode's
// occurrence field opens items by, to a brute-force enumeration of every
// ·partition· cvc-accept clause 3 admits: over random content models,
// seeded, of depth at most 3, sequences and choices of elements a, b, c and
// at most one ##other wildcard, each particle occurring as rOccurs draws,
// kept where Finalize accepts the model (cos-nonambig among them), codegen
// neither splices nor flattens its content particle, and Decode binds it,
// each with random valid instances. For each instance the enumeration must
// find exactly one partition, and partition's item starts must be it. Where
// the occurrence's item is a struct of string fields, reflect builds codegen's
// type and Decode's items are held to the partition too, by the ids each
// child carries.
func TestOccurrencePartitionDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(3593, 1))
	models, instances, decoded, tries, choices, nested, shared := 0, 0, 0, 0, 0, 0, 0
	for ; models < 2000 && tries < 20000; tries++ {
		wilds := 0
		top := genModel(rng, 0, &wilds)
		if top.min == 1 && top.max == 1 {
			continue
		}
		schema, ok := randomSchema(`<xs:element name="r"><xs:complexType>` + top.xsdOf() + `</xs:complexType></xs:element>`)
		if !ok {
			continue
		}
		decl, _ := schema.Element(rootName)
		td, _ := schema.ResolvedType(decl.TypeDefinition())
		ct := td.(xsd.ComplexType)
		content := ct.ContentType().(xsd.ElementContent)
		group, _, err := goshape.ModelGroupTerm(schema, content.Particle.Term())
		if err != nil {
			t.Fatal(err)
		}
		hold, err := goshape.ParticleHold(schema, group, content.Particle.Occurs())
		if err != nil {
			t.Fatal(err)
		}
		if hold == goshape.Spliced || hold == goshape.Flattened {
			continue
		}
		var de *DeclineError
		if err := CheckDecode[struct{}](schema, rootName); errors.As(err, &de) {
			continue
		}
		models++
		if top.kind == "choice" {
			choices++
		}
		if slices.ContainsFunc(top.kids, func(k *rnode) bool { return len(k.kids) > 0 }) {
			nested++
		}
		rt, structured := occurrenceGoType(top)
		for range 4 {
			names := namesOf(top.genInstance(rng, nil))
			if len(names) > 10 {
				continue
			}
			instances++
			found := top.partitions(names)
			if len(found) != 1 {
				t.Fatalf("model %s, instance %v: the enumeration finds %d partitions %v, want exactly 1", top.xsdOf(), names, len(found), found)
			}
			m, ok := schema.ContentMatcher(ct)
			if !ok {
				t.Fatalf("model %s: ContentMatcher declines a model Decode binds", top.xsdOf())
			}
			part := partition{schema: schema, group: group}
			var starts []int
			for i, name := range names {
				qn := xsd.QName{Space: "urn:s", Local: name}
				if name == "w" {
					qn = xsd.QName{Space: "urn:o", Local: "w"}
				}
				_, pos, ok := m.NextParticle(qn)
				if !ok {
					t.Fatalf("model %s, instance %v: the Matcher rejects child %d", top.xsdOf(), names, i)
				}
				opens, err := part.next(pos)
				if err != nil {
					t.Fatal(err)
				}
				if opens {
					starts = append(starts, i)
				}
			}
			if !slices.Equal(starts, found[0]) {
				t.Fatalf("model %s, instance %v: partition opens items at %v, the enumeration's partition is %v", top.xsdOf(), names, starts, found[0])
			}
			if !structured {
				continue
			}
			decoded++
			if sharesName(top) {
				shared++
			}
			holdDecoded(t, schema, top, rt, names, found[0])
		}
	}
	if models < 2000 {
		t.Fatalf("%d models kept of %d drawn, want 2000", models, tries)
	}
	if shared == 0 {
		t.Fatal("no decoded instance is of an item model two of whose particles take one expanded name, which Decode admits")
	}
	t.Logf("%d models of %d drawn, %d of them choices and %d holding a model group particle; %d instances, %d decoded through reflect-built types, %d of them of an item model sharing an expanded name",
		models, tries, choices, nested, instances, decoded, shared)
}

// occurrenceGoType returns codegen's type for s:r of top's model, reflect
// built, and true, where top is a sequence whose particles are elements and
// a wildcard only, so that the item is a struct of no sealed interface. Its
// fields are named from one Namer in particle order, as codegen names one
// occurrence's struct (codegen/doc.go "# Naming"), so that two particles of
// one name get two fields.
func occurrenceGoType(top *rnode) (reflect.Type, bool) {
	if top.kind != "sequence" {
		return nil, false
	}
	names := goname.NewNamer()
	var fields []reflect.StructField
	for _, k := range top.kids {
		if k.kind != "elem" && k.kind != "wild" {
			return nil, false
		}
		v, local := reflect.TypeFor[string](), k.name
		if k.kind == "wild" {
			v, local = reflect.TypeFor[anyElement](), "any"
		}
		fields = append(fields, reflect.StructField{Name: names.Claim(local), Type: shapedType(k.min, k.max, v)})
	}
	item := reflect.StructOf(fields)
	return reflect.StructOf([]reflect.StructField{{Name: "Sequence", Type: shapedType(top.min, top.max, item)}}), true
}

// sharesName reports whether two of top's element particles take one
// expanded name, the share at an occurrence field's item that Decode
// attributes by position (TestDecodeSharedNameInItem).
func sharesName(top *rnode) bool {
	seen := map[string]bool{}
	for _, k := range top.kids {
		if k.kind != "elem" {
			continue
		}
		if seen[k.name] {
			return true
		}
		seen[k.name] = true
	}
	return false
}

// shapedType is v in the shape goshape.ParticleShape gives a particle
// occurring min to max times over a struct or string.
func shapedType(lo, hi int, v reflect.Type) reflect.Type {
	if hi < 0 || hi > 1 {
		return reflect.SliceOf(v)
	}
	if lo == 0 {
		return reflect.PointerTo(v)
	}
	return v
}

// holdDecoded decodes the instance of names, each child carrying its index
// as its text, into rt, and holds each item's children, read back by those
// indices, to the sub-sequence of starts it stands for.
func holdDecoded(t *testing.T, schema *xsd.Schema, top *rnode, rt reflect.Type, names []string, starts []int) {
	t.Helper()
	out := reflect.New(rt).Elem()
	if err := decodeValue(schema, strict.New(), idInstance(names), out, nil); err != nil {
		t.Fatalf("model %s, instance %v: Decode = %v", top.xsdOf(), names, err)
	}
	var items []reflect.Value
	field := out.Field(0)
	switch field.Kind() {
	case reflect.Slice:
		for i := range field.Len() {
			items = append(items, field.Index(i))
		}
	case reflect.Pointer:
		if !field.IsNil() {
			items = append(items, field.Elem())
		}
	default:
		items = append(items, field)
	}
	var got [][]int
	for _, it := range items {
		got = append(got, idsOf(it))
	}
	var want [][]int
	for k, s := range starts {
		end := len(names)
		if k+1 < len(starts) {
			end = starts[k+1]
		}
		var ids []int
		for i := s; i < end; i++ {
			ids = append(ids, i)
		}
		want = append(want, ids)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("model %s, instance %v: Decode's items hold children %v, the partition's are %v", top.xsdOf(), names, got, want)
	}
}

// idInstance returns the instance of s:r whose children are names, in order,
// "w" an urn:o child, each carrying its index as its text.
func idInstance(names []string) []byte {
	var body strings.Builder
	for i, n := range names {
		if n == "w" {
			fmt.Fprintf(&body, `<o:w xmlns:o="urn:o">%d</o:w>`, i)
			continue
		}
		fmt.Fprintf(&body, "<s:%s>%d</s:%s>", n, i, n)
	}
	return instanceOf("", body.String())
}

// TestAppendOccurrenceRoundTripDifferential holds Append's occurrence field
// to Decode's partition with a nested model group in the item: over random
// content models drawn as TestOccurrencePartitionDifferential draws them,
// seeded, kept where Finalize accepts the model, codegen holds the content
// particle in an occurrence field, a sequence whose item holds at least one
// nested model group particle and only nested sequences codegen splices, at
// any depth, so that reflect builds the item's struct (memberFields), and
// Append encodes the type, each with random valid instances decoded into it.
// Each decoded value is appended, and the written bytes are checked
// independently by validate's verdict, which must accept them, and by
// Decode, whose value must equal the one appended. Its seed keeps 2000
// models of 112824 drawn, over 6519 instances, which its log states.
func TestAppendOccurrenceRoundTripDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(4076, 1))
	models, instances, tries := 0, 0, 0
	for ; models < 2000 && tries < 1000000; tries++ {
		wilds := 0
		top := genModel(rng, 0, &wilds)
		if top.kind != "sequence" || top.min == 1 && top.max == 1 || !slices.ContainsFunc(top.kids, func(k *rnode) bool { return len(k.kids) > 0 }) {
			continue
		}
		schema, ok := randomSchema(`<xs:element name="r"><xs:complexType>` + top.xsdOf() + `</xs:complexType></xs:element>`)
		if !ok {
			continue
		}
		decl, _ := schema.Element(rootName)
		td, _ := schema.ResolvedType(decl.TypeDefinition())
		content := td.(xsd.ComplexType).ContentType().(xsd.ElementContent)
		group, _, err := goshape.ModelGroupTerm(schema, content.Particle.Term())
		if err != nil {
			t.Fatal(err)
		}
		if hold, err := goshape.ParticleHold(schema, group, content.Particle.Occurs()); err != nil || hold != goshape.Occurrence {
			continue
		}
		fields, _, why := memberFields(t, schema, group, top, goname.NewNamer(), nil, nil)
		if why != "" || flattenedFields(fields) > 0 {
			continue
		}
		rt := reflect.StructOf([]reflect.StructField{{Name: "Sequence", Type: shapedType(top.min, top.max, reflect.StructOf(fields))}})
		var de *DeclineError
		if _, err := encodeValue(nil, schema, strict.New(), rootName, reflect.New(rt).Elem(), nil); errors.As(err, &de) {
			continue
		}
		models++
		for range 4 {
			names := namesOf(top.genInstance(rng, nil))
			if len(names) > 10 {
				continue
			}
			instances++
			v := reflect.New(rt).Elem()
			if err := decodeValue(schema, strict.New(), idInstance(names), v, nil); err != nil {
				t.Fatalf("model %s, instance %v: Decode = %v", top.xsdOf(), names, err)
			}
			data, err := encodeValue(nil, schema, strict.New(), rootName, v, nil)
			if err != nil {
				t.Fatalf("model %s, instance %v: Append = %v", top.xsdOf(), names, err)
			}
			accepts(t, schema, data)
			back := reflect.New(rt).Elem()
			if err := decodeValue(schema, strict.New(), data, back, nil); err != nil {
				t.Fatalf("model %s, instance %v: Decode(%s) = %v", top.xsdOf(), names, data, err)
			}
			if !reflect.DeepEqual(unscoped(back.Interface()), unscoped(v.Interface())) {
				t.Fatalf("model %s, instance %v: Decode(%s) = %#v, Append wrote %#v", top.xsdOf(), names, data, back.Interface(), v.Interface())
			}
		}
	}
	if models < 2000 {
		t.Fatalf("%d models kept of %d drawn, want 2000", models, tries)
	}
	t.Logf("%d models of %d drawn, %d instances appended and decoded back", models, tries, instances)
}

// memberFields appends to fields the fields of the struct codegen emits for
// one occurrence of group, the model group of n, a random sequence's, and to
// ends, for each field, the end in the reach of n's root (rnode.reachOf) of
// the leaves its member holds, each field's first leaf the previous field's
// end, and returns both and the empty reason, where each of n's particles is
// an element, a wildcard, a nested sequence goshape.ParticleHold answers
// Spliced for, whose own particles are so in turn, at any depth, or a nested
// particle it answers Flattened for: each element's and wildcard's field in
// particle order, holding its one leaf, a spliced sequence's in its place,
// and a flattened particle's []rfItem, holding its whole reach, named from
// names, one Namer, as codegen claims the fields of every particle its
// spliced returns (codegen/doc.go "# Naming"). Otherwise the reason names the
// first nested particle's hold that is none of these.
func memberFields(t *testing.T, schema *xsd.Schema, group xsd.ModelGroup, n *rnode, names *goname.Namer, fields []reflect.StructField, ends []int) ([]reflect.StructField, []int, string) {
	t.Helper()
	for i, k := range n.kids {
		end := 0
		if len(ends) > 0 {
			end = ends[len(ends)-1]
		}
		p := group.Particles()[i]
		switch k.kind {
		case "elem", "wild":
			v, local := reflect.TypeFor[string](), k.name
			if k.kind == "wild" {
				v, local = reflect.TypeFor[anyElement](), "any"
			}
			fields = append(fields, reflect.StructField{Name: names.Claim(local), Type: shapedType(k.min, k.max, v)})
			ends = append(ends, end+1)
			continue
		}
		inner, _, err := goshape.ModelGroupTerm(schema, p.Term())
		if err != nil {
			t.Fatal(err)
		}
		hold, err := goshape.ParticleHold(schema, inner, p.Occurs())
		if err != nil {
			t.Fatal(err)
		}
		switch hold {
		case goshape.Spliced:
			var why string
			if fields, ends, why = memberFields(t, schema, inner, k, names, fields, ends); why != "" {
				return nil, nil, why
			}
		case goshape.Flattened:
			fields = append(fields, reflect.StructField{Name: names.Claim(k.kind), Type: reflect.TypeFor[[]rfItem]()})
			ends = append(ends, end+len(k.reachOf(nil, nil)))
		case goshape.OnceChoice:
			return nil, nil, "a nested choice held once"
		default:
			return nil, nil, "a nested " + k.kind + " held as an occurrence field"
		}
	}
	return fields, ends, ""
}

// isFlattenedField reports whether f, a field of memberFields' struct, is a
// flattened particle's, the []rfItem field its Flattened arm builds.
func isFlattenedField(f reflect.StructField) bool {
	return f.Type == reflect.TypeFor[[]rfItem]()
}

// flattenedFields returns how many of fields, memberFields', are a flattened
// particle's (isFlattenedField).
func flattenedFields(fields []reflect.StructField) int {
	n := 0
	for _, f := range fields {
		if isFlattenedField(f) {
			n++
		}
	}
	return n
}

// idsOf returns the child indices an item holds in field order: each string
// field's, each anyElement's Content.Text, through pointers and slices, -1
// for a string that is no index, which no partition holds.
func idsOf(v reflect.Value) []int {
	var ids []int
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			ids = idsOf(v.Elem())
		}
	case reflect.Slice:
		for i := range v.Len() {
			ids = append(ids, idsOf(v.Index(i))...)
		}
	case reflect.String:
		n, err := strconv.Atoi(v.String())
		if err != nil {
			n = -1
		}
		ids = append(ids, n)
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[anyElement]() {
			return idsOf(v.FieldByName("Content").FieldByName("Text"))
		}
		for i := range v.NumField() {
			ids = append(ids, idsOf(v.Field(i))...)
		}
	default:
		// No other kind is a field of the types occurrenceGoType builds.
		ids = append(ids, -1)
	}
	return ids
}

// rfItem is the sealed interface of the flattened particle the random
// differentials write: the content {particle} TestAppendFlattenedDifferential
// writes, held by rfSequence for a sequence and by rfChoice for a choice, and
// the nested particle TestAppendNestedFlattenedDifferential writes, held by
// memberFields' []rfItem field, as codegen holds each (codegen/doc.go
// "# Particles"). Its branch family: RfItemA to RfItemC4, the branches of up
// to four element particles of each name a, b and c, named as codegen names a
// tie's (codegen/doc.go "# Naming") and placed by shape where no tie holds
// them; and rfAny, the wildcard particle's.
type (
	rfSequence struct{ Sequence []rfItem }
	rfChoice   struct{ Choice []rfItem }
	rfItem     interface{ isRfItem() }
	RfItemA    struct{ A string }
	RfItemA2   struct{ A string }
	RfItemA3   struct{ A string }
	RfItemA4   struct{ A string }
	RfItemB    struct{ B string }
	RfItemB2   struct{ B string }
	RfItemB3   struct{ B string }
	RfItemB4   struct{ B string }
	RfItemC    struct{ C string }
	RfItemC2   struct{ C string }
	RfItemC3   struct{ C string }
	RfItemC4   struct{ C string }
	rfAny      struct{ Any anyElement }
)

func (RfItemA) isRfItem()  {}
func (RfItemA2) isRfItem() {}
func (RfItemA3) isRfItem() {}
func (RfItemA4) isRfItem() {}
func (RfItemB) isRfItem()  {}
func (RfItemB2) isRfItem() {}
func (RfItemB3) isRfItem() {}
func (RfItemB4) isRfItem() {}
func (RfItemC) isRfItem()  {}
func (RfItemC2) isRfItem() {}
func (RfItemC3) isRfItem() {}
func (RfItemC4) isRfItem() {}
func (rfAny) isRfItem()    {}

// rfTies holds, by element name, the branches of the first to fourth element
// particle of that name in a reach, in reach order.
var rfTies = map[string][]rfItem{
	"a": {RfItemA{}, RfItemA2{}, RfItemA3{}, RfItemA4{}},
	"b": {RfItemB{}, RfItemB2{}, RfItemB3{}, RfItemB4{}},
	"c": {RfItemC{}, RfItemC2{}, RfItemC3{}, RfItemC4{}},
}

// placeBranches places at each of leaves[first:end], a flattened particle's
// reach, its rfItem branch, rfAny for the wildcard and rfTies' for an
// element, and returns them in reach order and true, or false where the
// reach holds more element particles of one name than rfTies has branches.
func placeBranches(leaves []flatLeaf, first, end int) ([]rfItem, bool) {
	var branches []rfItem
	ranks := map[string]int{}
	for i := first; i < end; i++ {
		l := leaves[i]
		if l.node.kind == "wild" {
			leaves[i].branch = rfAny{}
			branches = append(branches, leaves[i].branch)
			continue
		}
		if ranks[l.node.name] == len(rfTies[l.node.name]) {
			return nil, false
		}
		leaves[i].branch = rfTies[l.node.name][ranks[l.node.name]]
		branches = append(branches, leaves[i].branch)
		ranks[l.node.name]++
	}
	return branches, true
}

// flatLeaf is one basic particle of a random model's reach: node, its
// position as xsd.Matcher.NextParticle reports one, and branch, the rfItem
// branch the test's WithBranches set places at it (placeBranches), nil
// outside the flattened particle's reach.
type flatLeaf struct {
	node   *rnode
	pos    []int
	branch rfItem
}

// reachOf appends to out the leaves of n's reach, n a model group at
// position at: depth-first through its kids in order, each element or
// wildcard one leaf, each model group replaced by its own kids' leaves.
func (n *rnode) reachOf(at []int, out []flatLeaf) []flatLeaf {
	for i, k := range n.kids {
		pos := append(slices.Clone(at), i)
		if len(k.kids) == 0 {
			out = append(out, flatLeaf{node: k, pos: pos})
			continue
		}
		out = k.reachOf(pos, out)
	}
	return out
}

// flatItem is one item of a flattened slice a draw holds: the index of its
// branch's leaf in the reach, and the expanded name of the child it writes,
// the leaf's own for an element, the anyElement's for the wildcard.
type flatItem struct {
	leaf int
	name xsd.QName
}

// The verdicts of the content model on a draw, as matcherVerdict reads them
// from xsd.Matcher: accepted; an item whose name it admits at no particle; an
// item it attributes to a particle other than its branch's leaf; and content
// after whose last item it is not accepting. Each reject is §3.9.4.3
// cvc-accept clause 3.3 through §3.8.4.3 cvc-model-group.
const (
	verdictAccept = iota
	verdictNowhere
	verdictElsewhere
	verdictShort
)

// verdictNames names the verdicts in the test's log, by verdict.
var verdictNames = [...]string{"accepted", "name admitted nowhere", "name attributed elsewhere", "ends short"}

// matcherVerdict returns m's verdict on items, m a fresh Matcher of the
// content whose reach is leaves: the reject at the first item m admits at no
// particle or attributes to a position other than its leaf's, otherwise the
// reject at the end where m is not accepting, otherwise accepted.
func matcherVerdict(m *xsd.Matcher, leaves []flatLeaf, items []flatItem) int {
	for _, it := range items {
		_, pos, ok := m.NextParticle(it.name)
		if !ok {
			return verdictNowhere
		}
		if !slices.Equal(pos, leaves[it.leaf].pos) {
			return verdictElsewhere
		}
	}
	if !m.Accepting() {
		return verdictShort
	}
	return verdictAccept
}

// rfWildNames are the names a wildcard item's child draws from: urn:o's w,
// which the ##other wildcard admits, two times in six; urn:p's w, which it
// admits too; one in no namespace and one in urn:s, which it does not
// (§3.10.4.1 cvc-wildcard); and urn:s's a, which an element particle a
// admits first where one is live.
var rfWildNames = []xsd.QName{{Space: "urn:o", Local: "w"}, {Space: "urn:o", Local: "w"}, {Space: "urn:p", Local: "w"}, {Local: "w"}, {Space: "urn:s", Local: "w"}, {Space: "urn:s", Local: "a"}}

// drawItems returns a random draw over leaves, the reach of top: a sequence
// genInstance draws, each child the item of its leaf, then, by one of six
// ops drawn evenly, unchanged by two of them, or with one random item
// inserted, or, where the sequence holds an item, with one random item
// replaced or removed or the tail from a random index cut.
func drawItems(rng *rand.Rand, top *rnode, leaves []flatLeaf) []flatItem {
	var items []flatItem
	for _, n := range top.genInstance(rng, nil) {
		items = append(items, randomItem(rng, leaves, slices.IndexFunc(leaves, func(l flatLeaf) bool { return l.node == n })))
	}
	switch op := rng.IntN(6); {
	case op == 2 && len(items) > 0:
		items[rng.IntN(len(items))] = randomItem(rng, leaves, rng.IntN(len(leaves)))
	case op == 3:
		items = slices.Insert(items, rng.IntN(len(items)+1), randomItem(rng, leaves, rng.IntN(len(leaves))))
	case op == 4 && len(items) > 0:
		i := rng.IntN(len(items))
		items = slices.Delete(items, i, i+1)
	case op == 5 && len(items) > 0:
		items = items[:rng.IntN(len(items))]
	}
	return items
}

// randomItem returns an item of leaves[leaf], its name drawn from
// rfWildNames for the wildcard.
func randomItem(rng *rand.Rand, leaves []flatLeaf, leaf int) flatItem {
	if leaves[leaf].node.kind == "wild" {
		return flatItem{leaf: leaf, name: rfWildNames[rng.IntN(len(rfWildNames))]}
	}
	return firstItem(leaves, leaf)
}

// firstItem returns the item of leaves[leaf] of its first name: the
// element's own, rfWildNames' first for the wildcard.
func firstItem(leaves []flatLeaf, leaf int) flatItem {
	n := leaves[leaf].node
	if n.kind == "wild" {
		return flatItem{leaf: leaf, name: rfWildNames[0]}
	}
	return flatItem{leaf: leaf, name: xsd.QName{Space: "urn:s", Local: n.name}}
}

// flatValue returns the value of rt, rfSequence or rfChoice, whose slice
// holds items, each in its leaf's branch (branchValue), the text of each
// child its index, and is nil for no item, as Decode returns it.
func flatValue(rt reflect.Type, leaves []flatLeaf, items []flatItem) reflect.Value {
	v := reflect.New(rt).Elem()
	if len(items) == 0 {
		return v
	}
	slice := reflect.MakeSlice(reflect.TypeFor[[]rfItem](), len(items), len(items))
	for i, it := range items {
		slice.Index(i).Set(branchValue(leaves, it, strconv.Itoa(i)))
	}
	v.Field(0).Set(slice)
	return v
}

// branchValue returns the value of it's leaf's branch holding the child
// leafValue returns.
func branchValue(leaves []flatLeaf, it flatItem, text string) reflect.Value {
	b := reflect.New(reflect.TypeOf(leaves[it.leaf].branch)).Elem()
	b.Field(0).Set(leafValue(leaves, it, text))
	return b
}

// leafValue returns the child it writes, of text text: the string of an
// element's, the anyElement of the wildcard's, of it's name.
func leafValue(leaves []flatLeaf, it flatItem, text string) reflect.Value {
	if leaves[it.leaf].node.kind == "wild" {
		return reflect.ValueOf(anyElement{Space: it.name.Space, Local: it.name.Local, Content: anyType{Text: text}})
	}
	return reflect.ValueOf(text)
}

// checkDraw returns the Matcher's verdict (matcherVerdict) on items, the
// children Append writes from v, under ct, the content whose reach is leaves,
// and fails t where errors.Is(err, ErrInvalidValue) on Append's answer under
// opts is not that verdict's reject, where Append answers any other error, or
// where an accepted draw's bytes do not decode, under opts, to v.
func checkDraw(t *testing.T, schema *xsd.Schema, ct xsd.ComplexType, top *rnode, leaves []flatLeaf, items []flatItem, v reflect.Value, opts []Option) int {
	t.Helper()
	m, ok := schema.ContentMatcher(ct)
	if !ok {
		t.Fatalf("model %s: ContentMatcher declines a model CheckAppend admits", top.xsdOf())
	}
	verdict := matcherVerdict(m, leaves, items)
	data, err := encodeValue(nil, schema, strict.New(), rootName, v, opts)
	if err != nil && !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("model %s, items %v: Append = %v, want nil or ErrInvalidValue", top.xsdOf(), items, err)
	}
	if errors.Is(err, ErrInvalidValue) != (verdict != verdictAccept) {
		t.Fatalf("model %s, items %v: Append = %v, the Matcher's verdict is %s", top.xsdOf(), items, err, verdictNames[verdict])
	}
	if err != nil {
		return verdict
	}
	back := reflect.New(v.Type()).Elem()
	if err := decodeValue(schema, strict.New(), data, back, opts); err != nil {
		t.Fatalf("model %s, items %v: Decode(%s) = %v", top.xsdOf(), items, data, err)
	}
	if !reflect.DeepEqual(unscoped(back.Interface()), unscoped(v.Interface())) {
		t.Fatalf("model %s, items %v: Decode(%s) = %#v, Append wrote %#v", top.xsdOf(), items, data, back.Interface(), v.Interface())
	}
	return verdict
}

// TestAppendFlattenedDifferential holds Append's flattened content
// {particle}, boundFlattened.write, to xsd.Schema.ContentMatcher on both
// sides (§3.9.4.3 cvc-accept clause 3.3, §3.8.4.3 cvc-model-group; item 6 of
// Append's doc): over random content models drawn as
// TestOccurrencePartitionDifferential draws them, seeded, kept where Finalize
// accepts the model, goshape.ParticleHold answers Flattened for its content
// {particle}, the rfItem family covers its reach, and CheckAppend admits
// rfSequence or rfChoice under the WithBranches set placing it, each with
// random draws of items (drawItems). On each draw, errors.Is(err,
// ErrInvalidValue) on Append's answer must equal the Matcher's reject
// (matcherVerdict), any other error failing, and each accepted draw's bytes
// must decode, under the same set, to the value appended. The log counts
// the models kept, the models of a Flattened content {particle} the family
// does not cover, by reason, and the draws of each verdict; the test fails
// where the kept models or a verdict count 0.
func TestAppendFlattenedDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(4105, 1))
	models, tries, wideTie, declined := 0, 0, 0, 0
	var verdicts [len(verdictNames)]int
	for ; tries < 20000; tries++ {
		wilds := 0
		top := genModel(rng, 0, &wilds)
		schema, ok := randomSchema(`<xs:element name="r"><xs:complexType>` + top.xsdOf() + `</xs:complexType></xs:element>`)
		if !ok {
			continue
		}
		decl, _ := schema.Element(rootName)
		td, _ := schema.ResolvedType(decl.TypeDefinition())
		ct := td.(xsd.ComplexType)
		content := ct.ContentType().(xsd.ElementContent)
		group, _, err := goshape.ModelGroupTerm(schema, content.Particle.Term())
		if err != nil {
			t.Fatal(err)
		}
		if hold, err := goshape.ParticleHold(schema, group, content.Particle.Occurs()); err != nil || hold != goshape.Flattened {
			continue
		}
		leaves := top.reachOf(nil, nil)
		branches, covered := placeBranches(leaves, 0, len(leaves))
		if !covered {
			wideTie++
			continue
		}
		opts := []Option{WithBranches(branches...)}
		rt, check := reflect.TypeFor[rfSequence](), CheckAppend[rfSequence]
		if top.kind == "choice" {
			rt, check = reflect.TypeFor[rfChoice](), CheckAppend[rfChoice]
		}
		if err := check(schema, rootName, opts...); err != nil {
			declined++
			t.Logf("model %s: CheckAppend = %v", top.xsdOf(), err)
			continue
		}
		models++
		for range 12 {
			items := drawItems(rng, top, leaves)
			verdicts[checkDraw(t, schema, ct, top, leaves, items, flatValue(rt, leaves, items), opts)]++
		}
	}
	t.Logf("%d models kept of %d drawn; not covered: %d with more than 4 element particles of one name, %d CheckAppend declines", models, tries, wideTie, declined)
	for i, n := range verdicts {
		t.Logf("%d draws %s", n, verdictNames[i])
	}
	if models == 0 || slices.Contains(verdicts[:], 0) {
		t.Fatalf("%d models kept, verdict counts %v: want each above 0", models, verdicts)
	}
}

// holdsFlattened reports whether a nested particle of group, the model group
// of n, is one goshape.ParticleHold answers Flattened for, at any depth
// outside such a particle's reach.
func holdsFlattened(t *testing.T, schema *xsd.Schema, group xsd.ModelGroup, n *rnode) bool {
	t.Helper()
	for i, k := range n.kids {
		if len(k.kids) == 0 {
			continue
		}
		p := group.Particles()[i]
		inner, _, err := goshape.ModelGroupTerm(schema, p.Term())
		if err != nil {
			t.Fatal(err)
		}
		hold, err := goshape.ParticleHold(schema, inner, p.Occurs())
		if err != nil {
			t.Fatal(err)
		}
		if hold == goshape.Flattened || holdsFlattened(t, schema, inner, k) {
			return true
		}
	}
	return false
}

// fieldLeaves returns the bounds in the reach of the leaves field i of
// memberFields' struct holds, ends its ends.
func fieldLeaves(ends []int, i int) (first, end int) {
	if i > 0 {
		first = ends[i-1]
	}
	return first, ends[i]
}

// nestedValue returns the value of rt, the struct memberFields builds, whose
// field i holds the leaves fieldLeaves bounds, holding items,
// and the children Append writes from it, in field order (writeMembers):
// each field's items in draw order, a flattened particle's each in its
// leaf's branch (branchValue), an element's or wildcard's (leafValue) cut to
// the one item a pointer or a bare field holds, a bare field's missing one
// its leaf's firstItem; the text of each child its index among them, and a
// field of no item nil, as Decode returns it.
func nestedValue(rt reflect.Type, ends []int, leaves []flatLeaf, items []flatItem) (reflect.Value, []flatItem) {
	v := reflect.New(rt).Elem()
	var written []flatItem
	for i := range ends {
		first, end := fieldLeaves(ends, i)
		var held []flatItem
		for _, it := range items {
			if it.leaf >= first && it.leaf < end {
				held = append(held, it)
			}
		}
		f := v.Field(i)
		flat := isFlattenedField(rt.Field(i))
		switch {
		case f.Kind() == reflect.Pointer:
			held = held[:min(len(held), 1)]
		case f.Kind() != reflect.Slice && len(held) == 0:
			held = []flatItem{firstItem(leaves, first)}
		case f.Kind() != reflect.Slice:
			held = held[:1]
		}
		if len(held) == 0 {
			continue
		}
		vals := make([]reflect.Value, len(held))
		for k, it := range held {
			text := strconv.Itoa(len(written))
			vals[k] = leafValue(leaves, it, text)
			if flat {
				vals[k] = branchValue(leaves, it, text)
			}
			written = append(written, it)
		}
		switch f.Kind() {
		case reflect.Slice:
			f.Set(reflect.MakeSlice(f.Type(), len(vals), len(vals)))
			for k, x := range vals {
				f.Index(k).Set(x)
			}
		case reflect.Pointer:
			f.Set(reflect.New(f.Type().Elem()))
			f.Elem().Set(vals[0])
		default:
			f.Set(vals[0])
		}
	}
	return v, written
}

// TestAppendNestedFlattenedDifferential holds Append's nested flattened
// particle, boundFlatSlice.writeMember through writeMembers, to
// xsd.Schema.ContentMatcher on both sides, as TestAppendFlattenedDifferential
// holds the flattened content {particle} (§3.9.4.3 cvc-accept clause 3.3,
// §3.8.4.3 cvc-model-group, §3.10.4.1 cvc-wildcard): over random content
// models drawn as TestOccurrencePartitionDifferential draws them, seeded,
// kept where Finalize accepts the model, goshape.ParticleHold answers Spliced
// for its content {particle}, a sequence one of whose nested particles it
// answers Flattened for, reflect builds codegen's struct (memberFields) with
// exactly one such particle, the rfItem family covers that particle's reach,
// and CheckAppend admits the struct under the WithBranches set placing it
// (encodeStatic, CheckAppend's answer), each with random draws of items
// (drawItems) held in the struct's fields (nestedValue). On each draw,
// errors.Is(err, ErrInvalidValue) on Append's answer must equal the Matcher's
// reject (matcherVerdict) on the children nestedValue states Append writes,
// any other error failing, and each accepted draw's bytes must decode, under
// the same set, to the value appended. A model whose content {particle}
// ParticleHold answers Occurrence for, a sequence, is built as its occurrence
// field and must be declined by item 4 of Append's doc. The log counts the
// models kept, the declines by construct and the content {particle}'s hold,
// the models holding a nested flattened particle not built, by reason, and
// the draws of each verdict; the test fails where the kept models or a
// verdict count 0.
func TestAppendNestedFlattenedDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(4153, 1))
	models, tries := 0, 0
	unbuilt, declined := map[string]int{}, map[string]int{}
	var verdicts [len(verdictNames)]int
	for ; tries < 20000; tries++ {
		wilds := 0
		top := genModel(rng, 0, &wilds)
		schema, ok := randomSchema(`<xs:element name="r"><xs:complexType>` + top.xsdOf() + `</xs:complexType></xs:element>`)
		if !ok {
			continue
		}
		decl, _ := schema.Element(rootName)
		td, _ := schema.ResolvedType(decl.TypeDefinition())
		ct := td.(xsd.ComplexType)
		content := ct.ContentType().(xsd.ElementContent)
		group, _, err := goshape.ModelGroupTerm(schema, content.Particle.Term())
		if err != nil {
			t.Fatal(err)
		}
		hold, err := goshape.ParticleHold(schema, group, content.Particle.Occurs())
		if err != nil {
			t.Fatal(err)
		}
		if hold == goshape.Flattened || !holdsFlattened(t, schema, group, top) {
			continue
		}
		if top.kind == "choice" && hold == goshape.OnceChoice {
			unbuilt["a content choice held once"]++
			continue
		}
		if top.kind == "choice" {
			unbuilt["a content choice held as an occurrence field"]++
			continue
		}
		fields, ends, why := memberFields(t, schema, group, top, goname.NewNamer(), nil, nil)
		if why != "" {
			unbuilt[why]++
			continue
		}
		if flattenedFields(fields) > 1 {
			unbuilt["two or more flattened particles"]++
			continue
		}
		leaves := top.reachOf(nil, nil)
		first, end := fieldLeaves(ends, slices.IndexFunc(fields, isFlattenedField))
		branches, covered := placeBranches(leaves, first, end)
		if !covered {
			unbuilt["more than 4 element particles of one name in the reach"]++
			continue
		}
		rt := reflect.StructOf(fields)
		if hold == goshape.Occurrence {
			rt = reflect.StructOf([]reflect.StructField{{Name: "Sequence", Type: shapedType(top.min, top.max, rt)}})
		}
		opts := []Option{WithBranches(branches...)}
		_, _, _, err = encodeStatic(schema, rootName, rt, newConfig(opts))
		var de *DeclineError
		if errors.As(err, &de) {
			declined[fmt.Sprintf("%s, the content {particle} held %s", de.Construct(), [...]string{goshape.Spliced: "spliced", goshape.Occurrence: "as an occurrence field"}[hold])]++
			continue
		}
		if err != nil {
			t.Fatalf("model %s: CheckAppend = %v, want nil or a *DeclineError", top.xsdOf(), err)
		}
		if hold == goshape.Occurrence {
			t.Fatalf("model %s: CheckAppend admits the occurrence field of a nested flattened particle, which the test draws no value of", top.xsdOf())
		}
		models++
		for range 12 {
			v, written := nestedValue(rt, ends, leaves, drawItems(rng, top, leaves))
			verdicts[checkDraw(t, schema, ct, top, leaves, written, v, opts)]++
		}
	}
	t.Logf("%d models kept of %d drawn", models, tries)
	for _, k := range slices.Sorted(maps.Keys(declined)) {
		t.Logf("%d CheckAppend declines: %s", declined[k], k)
	}
	for _, k := range slices.Sorted(maps.Keys(unbuilt)) {
		t.Logf("%d not built: %s", unbuilt[k], k)
	}
	for i, n := range verdicts {
		t.Logf("%d draws %s", n, verdictNames[i])
	}
	if models == 0 || slices.Contains(verdicts[:], 0) {
		t.Fatalf("%d models kept, verdict counts %v: want each above 0", models, verdicts)
	}
}
