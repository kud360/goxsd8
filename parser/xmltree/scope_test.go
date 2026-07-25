package xmltree_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/parser/xmltree"
)

// startElements returns the start tags of doc in document order.
func startElements(t *testing.T, doc string) []*xmltree.StartElement {
	t.Helper()
	nodes, err := collect(t, "t.xml", doc)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	var starts []*xmltree.StartElement
	for _, n := range nodes {
		if se, ok := n.(*xmltree.StartElement); ok {
			starts = append(starts, se)
		}
	}
	return starts
}

// formatPrefixes renders an in-scope binding list as "p=uri" entries joined by
// spaces, so a test can compare the whole result — including its order — at once.
func formatPrefixes(nss []xmltree.Namespace) string {
	parts := make([]string, 0, len(nss))
	for _, ns := range nss {
		parts = append(parts, ns.Prefix()+"="+ns.URI())
	}
	return strings.Join(parts, " ")
}

func TestInScopePrefixesSortedAndAccumulated(t *testing.T) {
	starts := startElements(t, `<a xmlns:z="urn:z" xmlns:b="urn:b"><c xmlns:m="urn:m"/></a>`)
	if len(starts) != 2 {
		t.Fatalf("got %d start elements, want 2", len(starts))
	}
	// Sorted by prefix, inner declaration folded in, "xml" always present.
	if got, want := formatPrefixes(starts[0].InScopePrefixes()),
		"b=urn:b xml="+xmltree.XMLNamespaceURI+" z=urn:z"; got != want {
		t.Errorf("<a> prefixes = %q, want %q", got, want)
	}
	if got, want := formatPrefixes(starts[1].InScopePrefixes()),
		"b=urn:b m=urn:m xml="+xmltree.XMLNamespaceURI+" z=urn:z"; got != want {
		t.Errorf("<c> prefixes = %q, want %q", got, want)
	}
}

func TestInScopePrefixesExcludesDefaultNamespace(t *testing.T) {
	starts := startElements(t, `<a xmlns="urn:default" xmlns:p="urn:p"/>`)
	if got, want := formatPrefixes(starts[0].InScopePrefixes()),
		"p=urn:p xml="+xmltree.XMLNamespaceURI; got != want {
		t.Errorf("prefixes = %q, want %q", got, want)
	}
	// The default namespace is still reachable, just not enumerated.
	uri, ok := starts[0].LookupPrefix("")
	if !ok || uri != "urn:default" {
		t.Errorf(`LookupPrefix("") = %q, %v; want "urn:default", true`, uri, ok)
	}
}

func TestInScopePrefixesAgreesWithLookupOnShadowing(t *testing.T) {
	starts := startElements(t, `<a xmlns:p="urn:outer"><b xmlns:p="urn:inner"/></a>`)
	if got, want := formatPrefixes(starts[1].InScopePrefixes()),
		"p=urn:inner xml="+xmltree.XMLNamespaceURI; got != want {
		t.Errorf("inner prefixes = %q, want %q", got, want)
	}
	if got, want := formatPrefixes(starts[0].InScopePrefixes()),
		"p=urn:outer xml="+xmltree.XMLNamespaceURI; got != want {
		t.Errorf("outer prefixes = %q, want %q", got, want)
	}
}

func TestInScopePrefixesDropsUndeclaredPrefix(t *testing.T) {
	// xmlns:p="" undeclares p (Namespaces in XML 1.1); it must not be enumerated
	// as a binding to the empty namespace name.
	starts := startElements(t, `<a xmlns:p="urn:p"><b xmlns:p=""/></a>`)
	if got, want := formatPrefixes(starts[1].InScopePrefixes()),
		"xml="+xmltree.XMLNamespaceURI; got != want {
		t.Errorf("prefixes after undeclaration = %q, want %q", got, want)
	}
}

func TestInScopePrefixesReservedPrefixes(t *testing.T) {
	// An explicit xmlns:xml declaration is recorded as a real binding while
	// lookup short-circuits "xml": the result must still carry exactly one xml
	// entry. An xmlns:xmlns declaration must never surface at all.
	starts := startElements(t,
		`<a xmlns:xml="`+xmltree.XMLNamespaceURI+`" xmlns:xmlns="urn:bogus" xmlns:p="urn:p"/>`)
	if got, want := formatPrefixes(starts[0].InScopePrefixes()),
		"p=urn:p xml="+xmltree.XMLNamespaceURI; got != want {
		t.Errorf("prefixes = %q, want %q", got, want)
	}
}

func TestInScopePrefixesAlwaysHasXML(t *testing.T) {
	starts := startElements(t, `<a/>`)
	nss := starts[0].InScopePrefixes()
	if len(nss) != 1 || nss[0].Prefix() != "xml" || nss[0].URI() != xmltree.XMLNamespaceURI {
		t.Fatalf("prefixes = %q, want the implicit xml binding only", formatPrefixes(nss))
	}
}
