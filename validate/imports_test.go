package validate

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// The boundary this file pins: the engine reaches no source decoder, so an
// instance format is a property of an adapter and never of the engine
// (PRINCIPLES 8, validate/doc.go). It is pinned over the TRANSITIVE closure,
// which is the only place it can break — this package's own import block is
// one grep and needs no test, while a decoder reaching the engine four
// packages down is invisible to that grep.
//
// The closure is an ALLOWED-TO-GROW set, not an allowlist: nothing here says
// which packages may join it, only which may not. docs/ARCHITECTURE.md note [2]
// is where the closure of the day is stated.
//
// The two bans below differ because encoding/json is unavoidable and the
// others are not: log/slog imports it for its JSONHandler, so it sits in the
// closure of every package in this module that logs (STYLE L1), and banning
// it over the whole closure would ban logging. What can be banned, and is,
// is any package of THIS module in the closure importing it — the only route
// by which the engine could actually decode with it.
var (
	// bannedInClosure must not appear anywhere in validate's import
	// closure. parser/... is here as well as the decoders themselves,
	// because parser/xmltree is how encoding/xml would arrive without any
	// file in this package naming it.
	bannedInClosure = []string{
		"encoding/xml",
		"encoding/asn1",
		"github.com/kud360/goxsd8/parser",
	}
	// bannedFromModulePackages must not be imported by any package of this
	// module in validate's closure.
	bannedFromModulePackages = append([]string{"encoding/json"}, bannedInClosure...)
)

const modulePath = "github.com/kud360/goxsd8"

func TestImportClosureExcludesDecoders(t *testing.T) {
	// One line per package in the closure: its import path, then its direct
	// imports.
	out, err := exec.Command("go", "list", "-deps",
		"-f", "{{.ImportPath}}{{range .Imports}} {{.}}{{end}}",
		modulePath+"/validate").Output()
	if err != nil {
		// The toolchain writes the reason to stderr; %v on the error alone
		// renders "exit status 1" and nothing a reader can act on.
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			t.Fatalf("go list -deps: %v: %s", err, ee.Stderr)
		}
		t.Fatalf("go list -deps: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var observed []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg, imports := fields[0], fields[1:]
		observed = append(observed, pkg)
		for _, banned := range bannedInClosure {
			if matchesPackage(pkg, banned) {
				t.Errorf("validate's closure reaches %s (banned: %s)", pkg, banned)
			}
		}
		if !matchesPackage(pkg, modulePath) {
			continue
		}
		for _, imp := range imports {
			for _, banned := range bannedFromModulePackages {
				if matchesPackage(imp, banned) {
					t.Errorf("%s imports %s (banned: %s)", pkg, imp, banned)
				}
			}
		}
	}

	// A scan that walks nothing passes every assertion it makes. The
	// anti-vacuity claim is that three packages the closure cannot lose were
	// each SEEN — validate itself, and the xsd and xsderr it is written
	// against — rather than a count, which changes every time the closure
	// legitimately grows and says nothing about what was walked.
	for _, want := range []string{modulePath + "/validate", modulePath + "/xsd", modulePath + "/xsderr"} {
		if !slices.Contains(observed, want) {
			t.Errorf("go list -deps did not report %s among validate's closure of %d packages; the scan is not walking it",
				want, len(observed))
		}
	}
}

// matchesPackage reports whether pkg is path or a package under it.
func matchesPackage(pkg, path string) bool {
	return pkg == path || strings.HasPrefix(pkg, path+"/")
}
