package xmlsrc

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// The boundary this file pins: the adapter surfaces xsi:schemaLocation and
// xsi:noNamespaceSchemaLocation (§3.2.7.3, §3.2.7.4) as ordinary attributes
// and follows neither, so schema loading policy stays the caller's. A
// resolver it cannot reach is one it cannot call, which is a stronger
// statement than any single call site could make, and it is pinned over the
// TRANSITIVE closure — a resolver arriving four packages down is invisible
// to a grep of this package's import block.
//
// The ban is EXACT, not by prefix: parser/xmltree is this package's whole
// decoder and its legitimate dependency, while parser itself is the schema
// document layer, which reaches loader.
var banned = []string{
	"github.com/kud360/goxsd8/loader",
	"github.com/kud360/goxsd8/parser",
}

const pkgPath = "github.com/kud360/goxsd8/validate/xmlsrc"

func TestImportClosureExcludesSchemaLoading(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", pkgPath).Output()
	if err != nil {
		// The toolchain writes the reason to stderr; %v on the error alone
		// renders "exit status 1" and nothing a reader can act on.
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			t.Fatalf("go list -deps: %v: %s", err, ee.Stderr)
		}
		t.Fatalf("go list -deps: %v", err)
	}

	pkgs := strings.Fields(string(out))
	for _, pkg := range pkgs {
		for _, ban := range banned {
			if pkg == ban {
				t.Errorf("xmlsrc's closure reaches %s", pkg)
			}
		}
	}

	// A scan that walks nothing passes every assertion it makes, and the ban
	// would also pass if the closure lost the decoder the package is built
	// on.
	if !strings.Contains(string(out), "github.com/kud360/goxsd8/parser/xmltree\n") {
		t.Errorf("go list -deps reported %d packages and no parser/xmltree among them", len(pkgs))
	}
}
