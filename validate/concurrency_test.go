package validate

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file is the committed proof behind validate/doc.go's concurrency
// sentence (#1004): one finalized *xsd.Schema and one constructed *Validator,
// shared across N goroutines, each independently assessing its own document.
// It is not a substitute for the field-by-field read that sentence rests
// on — a clean run only exercises the paths these goroutines take — but it
// is what stops that sentence from going stale silently if a later change
// adds unsynchronized per-call state to Validator, walk, or the shipped
// backend.

// concurrencySchema declares "root", governed by a named complex type
// carrying both an attribute use and element content, so a concurrent Assess
// call exercises the backend's value mapping (cvc-attribute, over the
// attribute), the content matcher (cvc-complex-content, over the children)
// and the walk's own per-call state together — not a schema so bare that a
// race would have nothing to find.
func concurrencySchema(t *testing.T) *xsd.Schema {
	t.Helper()
	uses := []xsd.AttributeUse{typedUse(t, "n", integerType(), true, nil, nil)}
	content := cSequence(t, false, cParticle(t, "item", 0, 3))
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "RootType"}, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, uses, nil, nil, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, nil, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	for _, st := range seeded {
		b.AddType(st)
	}
	b.AddType(ct)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the concurrency schema: %v", err)
	}
	return schema
}

// concurrencyDocument builds one <root n="{n}"><item>.../item>...</root>
// instance, distinct per call so no two goroutines ever touch the same
// Element, Attribute, or Children cursor — sharing THOSE across goroutines
// is an adapter's contract, not Schema's or Validator's, and this test's
// job is to isolate the latter.
func concurrencyDocument(n int) *testElement {
	root := &testElement{
		name: xsd.QName{Local: "root"},
		attrs: []Attribute{&testAttribute{
			name: local("n"), value: strconv.Itoa(n), loc: loc(1, 10),
		}},
		loc: loc(1, 1),
	}
	for i := 0; i < 3; i++ {
		root.kids = append(root.kids, ElementChild(&testElement{
			name: xsd.QName{Local: "item"},
			kids: []Child{TextChild(&testText{data: strconv.Itoa(i), loc: loc(2+i, 3)})},
			loc:  loc(2+i, 1),
		}))
	}
	return root
}

// TestValidatorAssessIsSafeForConcurrentUse is the race test validate/doc.go's
// concurrency sentence cites: one *xsd.Schema and one *Validator, built once,
// then driven by 50 goroutines — matching the external measurement #1004
// reports — each assessing its own document and reporting back over a
// channel, since only the goroutine that owns a *testing.T may call its
// Fatal/Error methods.
//
// Run with -race: go test -race ./validate/... -run TestValidatorAssessIsSafeForConcurrentUse -v
func TestValidatorAssessIsSafeForConcurrentUse(t *testing.T) {
	const goroutines = 50

	schema := concurrencySchema(t)
	v, err := New(schema, testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res := v.Assess(concurrencyDocument(i))
			if res.Err() != nil {
				errs <- fmt.Errorf("goroutine %d: Err() = %w, want nil", i, res.Err())
				return
			}
			if got := res.Violations(); len(got) != 0 {
				errs <- fmt.Errorf("goroutine %d: Violations() = %v, want none", i, got)
				return
			}
			errs <- nil
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
}
