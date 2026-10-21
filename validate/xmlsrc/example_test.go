package xmlsrc_test

import (
	"fmt"
	"strings"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
	"github.com/kud360/goxsd8/xsderr"
)

// ExampleValidate_sourceFault applies the test [Validate]'s doc gives for a
// source fault to four instances of README's order.xsd: one read clean, one
// whose second root faults after the walk returned, so no Result stands, and
// two the walk stopped in when <o:order> was never closed, the first with
// nothing charged before the fault and the second after an undeclared
// attribute was charged.
func ExampleValidate_sourceFault() {
	const orderXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
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

	backend := strict.New()
	schema, err := parser.Parse("order.xsd",
		parser.WithBackend(backend),
		parser.WithResolver(loader.Map(map[string]string{"order.xsd": orderXSD})))
	if err != nil {
		fmt.Println(err)
		return
	}
	v, err := validate.New(schema, backend)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, doc := range []string{
		`<o:order xmlns:o="urn:o"><o:item>a</o:item></o:order>`,
		`<o:order xmlns:o="urn:o"><o:item>a</o:item></o:order><o:order xmlns:o="urn:o"/>`,
		`<o:order xmlns:o="urn:o"><o:item>a</o:item>`,
		`<o:order xmlns:o="urn:o" bogus="x"><o:item>a</o:item>`,
	} {
		res, err := xmlsrc.Validate(v, strings.NewReader(doc), xmlsrc.WithURI("order.xml"))
		fault := "source fault: none"
		if err != nil || res.Err() != nil {
			channel, cause := "err", err
			if err == nil {
				channel, cause = "Result.Err", res.Err()
			}
			rule, _ := xsderr.RuleOf(cause)
			fault = fmt.Sprintf("source fault: %s [%s]", channel, rule)
		}
		if err != nil {
			fmt.Printf("%s; no Result stands\n", fault)
			continue
		}
		fmt.Printf("%s; %s, violations: %d\n", fault, standing(res.Standing()), len(res.Violations()))
	}

	// Output:
	// source fault: none; NotProvenInvalid, violations: 0
	// source fault: err [xml-wf]; no Result stands
	// source fault: Result.Err [xml-wf]; Undecided, violations: 0
	// source fault: Result.Err [xml-wf]; Charged, violations: 1
}

// standing names s; Standing has no String method, and its constants rank
// by meaning rather than by value.
func standing(s validate.Standing) string {
	switch s {
	case validate.Charged:
		return "Charged"
	case validate.Undecided:
		return "Undecided"
	case validate.NotProvenInvalid:
		return "NotProvenInvalid"
	}
	return fmt.Sprintf("Standing(%d)", int(s))
}
