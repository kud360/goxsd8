package codec

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// TestImportsExcludeXPath pins that codec imports no xpath directly: the
// evaluator a union's identification runs under is validate's
// (validate.Result.FacetAssertions), so codec cannot build one at a current
// dateTime of its own. The ban is on DIRECT imports, since validate imports
// xpath; validate itself among them shows the scan read the import list.
func TestImportsExcludeXPath(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`, "github.com/kud360/goxsd8/codec").Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			t.Fatalf("go list: %v: %s", err, ee.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}
	imports := strings.Fields(string(out))
	if slices.Contains(imports, "github.com/kud360/goxsd8/xpath") {
		t.Errorf("codec imports github.com/kud360/goxsd8/xpath directly")
	}
	if !slices.Contains(imports, "github.com/kud360/goxsd8/validate") {
		t.Errorf("go list reported %d imports and no validate among them", len(imports))
	}
}
