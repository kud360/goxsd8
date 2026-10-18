package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readmeOrderSchema is README's order.xsd: {urn:o}order with an attribute kind
// and a repeated child item of xs:string.
const readmeOrderSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:o" elementFormDefault="qualified">
  <xs:element name="order">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:attribute name="kind" type="xs:string"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// writeFiles writes each name's content into dir and returns dir.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestValidateJSONMemberLine pins the line a JSON instance's charge at a
// member jsonsrc names prints: the engine's rule and location, the exit code
// an XML instance's charge earns, and the member's key as spelled with what
// was expected at the end of the line. Each row's line is
// "<instance><at>: [<rule>] " then the engine's message, then hint.
func TestValidateJSONMemberLine(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"order.xsd":   readmeOrderSchema,
		"kind.json":   `{"o:order":{"$xmlns":{"o":"urn:o"},"@kind":"x","item":["a","b"]}}`,
		"colour.json": `{"o:order":{"$xmlns":{"o":"urn:o"},"kind":"x","colour":"red","item":["a","b"]}}`,
		"text.json":   `{"order":"just text"}`,
		"nested.json": `{"order":{"item":"a","item":{"y":"b"}}}`,
	})
	for _, c := range []struct {
		instance, at, rule, hint string
	}{
		{"kind.json", ":1:36", "cvc-complex-content",
			`; JSON member "@kind" names no child element declaration and no attribute use of the type of "o:order", whose members are keyed "kind", "item"; an attribute member is keyed by the attribute's own name, "kind", not "@kind"`},
		{"colour.json", ":1:47", "cvc-complex-content",
			`; JSON member "colour" names no child element declaration and no attribute use of the type of "o:order", whose members are keyed "kind", "item"`},
		{"text.json", ":1:10", "cvc-complex-type",
			`; JSON member "order" has the scalar value "just text", where the type of its element admits no character content: object expected, {"order":{...}}`},
		{"nested.json", ":1:30", "cvc-type",
			`; JSON member "y" is a member of the object of "item", whose type is a simple type: scalar expected, {"item":<string, number or boolean>}`},
	} {
		t.Run(c.instance, func(t *testing.T) {
			instance := filepath.Join(dir, c.instance)
			var stdout, stderr bytes.Buffer
			args := []string{"validate", "-schema", filepath.Join(dir, "order.xsd"), instance}
			if code := run(args, &stdout, &stderr); code != exitInvalid {
				t.Errorf("run(%q) = %d, want %d (stderr %q)", args, code, exitInvalid, stderr.String())
			}
			line := strings.TrimSuffix(stdout.String(), "\n")
			prefix := instance + c.at + ": [" + c.rule + "] "
			if strings.Contains(line, "\n") || !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, c.hint) {
				t.Errorf("run(%q) stdout = %q, want one line opening %q and ending %q", args, stdout.String(), prefix, c.hint)
			}
		})
	}
}

// TestValidateFormatMismatchHint pins the clause a -format forcing a format
// the instance's extension does not name adds to the line of an instance not
// well-formed in it, in both directions, and the lines it adds nothing to: a
// right-format malformed instance, one no extension names a format of, one
// whose extension names the reserved ber, standard input, a decline, and a
// fault reading the instance. want is the whole of stdout, and the rule token
// and exit code are those of the same fault without the clause.
func TestValidateFormatMismatchHint(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"order.xml":    "<order/>\n",
		"order.json":   `{"order":{}}` + "\n",
		"hello.json":   "hello\n",
		"declined.xml": `{"z:nosuch":{}}` + "\n",
		"order.txt":    "<order/>\n",
		"order.ber":    "<order/>\n",
	})
	at := func(name string) string { return filepath.Join(dir, name) }
	for _, c := range []struct {
		name   string
		args   []string
		stdin  string
		code   int
		want   string
		faults bool
	}{
		{"json forced on xml", []string{"-format", "json", at("order.xml")}, "", exitUndecided,
			at("order.xml") + ":1:1: [jsonsrc] malformed JSON: invalid character '<' looking for beginning of value; -format json read " + at("order.xml") + " as json, but its extension names xml: drop -format, or pass -format xml\n", false},
		{"xml forced on json", []string{"-format", "xml", at("order.json")}, "", exitInvalid,
			at("order.json") + ":1:1: [xml-wf] character data before the document element: only an XML declaration, a DOCTYPE, comments, processing instructions and white space may precede it (XML 1.0 [1] document, [22] prolog, [27] Misc); -format xml read " + at("order.json") + " as xml, but its extension names json: drop -format, or pass -format json\n", false},
		{"json forced on malformed json", []string{"-format", "json", malformedJSON}, "", exitUndecided,
			malformedJSON + ":2:1: [jsonsrc] malformed JSON: unexpected EOF\n", false},
		{"malformed json by extension", []string{at("hello.json")}, "", exitUndecided,
			at("hello.json") + ":1:1: [jsonsrc] malformed JSON: invalid character 'h' looking for beginning of value\n", false},
		{"json forced on an extension naming none", []string{"-format", "json", at("order.txt")}, "", exitUndecided,
			at("order.txt") + ":1:1: [jsonsrc] malformed JSON: invalid character '<' looking for beginning of value\n", false},
		{"json forced on an extension naming the reserved ber", []string{"-format", "json", at("order.ber")}, "", exitUndecided,
			at("order.ber") + ":1:1: [jsonsrc] malformed JSON: invalid character '<' looking for beginning of value\n", false},
		{"json forced on standard input", []string{"-format", "json", "-"}, "<order/>\n", exitUndecided,
			"-:1:1: [jsonsrc] malformed JSON: invalid character '<' looking for beginning of value\n", false},
		{"json forced on a decline", []string{"-format", "json", at("declined.xml")}, "", exitUndecided,
			at("declined.xml") + `:1:2: [jsonsrc] member "z:nosuch": no "$xmlns" in scope binds its prefix "z"; a prefixed key names the expanded name its prefix's binding gives it, so bind "z" in a "$xmlns" member, {"z":"<namespace name>"}, leading the object of the element the key names or is an attribute of, or of an ancestor` + "\n", false},
		{"json forced on a read fault", []string{"-format", "json", at("fault.xml")}, "", exitUndecided,
			at("fault.xml") + ":1:1: [jsonsrc] fault reading the input: read " + at("fault.xml") + ": input/output error\n", true},
		{"xml forced on a read fault", []string{"-format", "xml", at("fault.json")}, "", exitInvalid,
			at("fault.json") + ":1:1: [xml-wf] read " + at("fault.json") + ": input/output error\n", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.faults {
				// /proc/self/mem opens, and its first read faults (EIO at
				// offset 0), as TestValidateJSONReadFaultIsUndecided reads it;
				// the link gives it an extension naming the other format.
				const faulting = "/proc/self/mem"
				if _, err := os.ReadFile(faulting); err == nil || !strings.Contains(err.Error(), "input/output error") {
					t.Skipf("reading %s gives %v here, not the EIO this row needs", faulting, err)
				}
				if err := os.Symlink(faulting, c.args[len(c.args)-1]); err != nil {
					t.Fatal(err)
				}
			}
			if c.stdin != "" {
				r, w, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := w.WriteString(c.stdin); err != nil {
					t.Fatal(err)
				}
				_ = w.Close()
				old := os.Stdin
				os.Stdin = r
				t.Cleanup(func() { os.Stdin = old; _ = r.Close() })
			}
			args := append([]string{"validate", "-schema", orderSchema}, c.args...)
			var stdout, stderr bytes.Buffer
			if code := run(args, &stdout, &stderr); code != c.code {
				t.Errorf("run(%q) = %d, want %d (stderr %q)", args, code, c.code, stderr.String())
			}
			if stdout.String() != c.want {
				t.Errorf("run(%q) stdout =\n%q\nwant\n%q", args, stdout.String(), c.want)
			}
		})
	}
}
