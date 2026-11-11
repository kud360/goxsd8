package codec

import (
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
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

// maxString renders max as a maxOccurs value.
func maxString(m int) string {
	if m < 0 {
		return "unbounded"
	}
	return strconv.Itoa(m)
}

// genInstance appends to out the names of a random sequence n's particle
// matches, "w" for a wildcard's child, its repetitions capped at min+2.
func (n *rnode) genInstance(rng *rand.Rand, out []string) []string {
	upper := n.min + 2
	if n.max >= 0 && n.max < upper {
		upper = n.max
	}
	for range n.min + rng.IntN(upper-n.min+1) {
		out = n.genOnce(rng, out)
	}
	return out
}

// genOnce appends the names of one random occurrence of n's term.
func (n *rnode) genOnce(rng *rand.Rand, out []string) []string {
	switch n.kind {
	case "elem":
		return append(out, n.name)
	case "wild":
		return append(out, "w")
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
	models, instances, decoded, tries, choices, nested := 0, 0, 0, 0, 0, 0
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
		splice, err := goshape.Splices(schema, group, content.Particle.Occurs())
		if err != nil {
			t.Fatal(err)
		}
		flat, err := goshape.Flattens(schema, group, content.Particle.Occurs())
		if err != nil {
			t.Fatal(err)
		}
		if splice || flat {
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
			names := top.genInstance(rng, nil)
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
			holdDecoded(t, schema, top, rt, names, found[0])
		}
	}
	if models < 2000 {
		t.Fatalf("%d models kept of %d drawn, want 2000", models, tries)
	}
	t.Logf("%d models of %d drawn, %d of them choices and %d holding a model group particle; %d instances, %d decoded through reflect-built types",
		models, tries, choices, nested, instances, decoded)
}

// occurrenceGoType returns codegen's type for s:r of top's model, reflect
// built, and true, where top is a sequence whose particles are elements and
// a wildcard only, so that the item is a struct of no sealed interface.
func occurrenceGoType(top *rnode) (reflect.Type, bool) {
	if top.kind != "sequence" {
		return nil, false
	}
	var fields []reflect.StructField
	for _, k := range top.kids {
		if k.kind != "elem" && k.kind != "wild" {
			return nil, false
		}
		v, name := reflect.TypeFor[string](), strings.ToUpper(k.name)
		if k.kind == "wild" {
			v, name = reflect.TypeFor[anyElement](), "Any"
		}
		fields = append(fields, reflect.StructField{Name: name, Type: shapedType(k.min, k.max, v)})
	}
	item := reflect.StructOf(fields)
	return reflect.StructOf([]reflect.StructField{{Name: "Sequence", Type: shapedType(top.min, top.max, item)}}), true
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
	var body strings.Builder
	for i, n := range names {
		if n == "w" {
			fmt.Fprintf(&body, `<o:w xmlns:o="urn:o">%d</o:w>`, i)
			continue
		}
		fmt.Fprintf(&body, "<s:%s>%d</s:%s>", n, i, n)
	}
	out := reflect.New(rt).Elem()
	if err := decodeValue(schema, strict.New(), instanceOf("", body.String()), out, nil); err != nil {
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
