package xmltree

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// An internal subset that ends inside a comment, processing instruction or
// markup declaration is no [28b] intSubset: a fault, never a decline. The
// Reader never hands doctypeEntities such a subset — the decoder ends a
// DOCTYPE directive outside every such construct — so these rows read the
// directive text directly.
func TestSubsetEndingInsideMarkupIsAFault(t *testing.T) {
	loc := xsderr.Loc{URI: "t.xml", Line: 1, Col: 1}
	const rule = ` open where it ends (XML 1.0 [28b] intSubset, [29] markupdecl)`
	for _, tc := range []struct {
		directive string
		want      string
	}{
		{`DOCTYPE r [<?x a `, `t.xml:1:1: [xml-wf] DOCTYPE internal subset leaves "<?x"` + rule},
		{`DOCTYPE r [<!-- a `, `t.xml:1:1: [xml-wf] DOCTYPE internal subset leaves "<!--"` + rule},
		{`DOCTYPE r [<!ENTITY e 'v'`, `t.xml:1:1: [xml-wf] DOCTYPE internal subset leaves "<!ENTITY"` + rule},
	} {
		t.Run(tc.directive, func(t *testing.T) {
			_, _, unread, err := doctypeEntities(tc.directive, false, loc)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("doctypeEntities(%q) error = %v, want %q", tc.directive, err, tc.want)
			}
			if unread {
				t.Errorf("doctypeEntities(%q) unread = true, want false: a fault declines nothing", tc.directive)
			}
		})
	}
}
