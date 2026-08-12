package validate

import (
	"os/exec"
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
		t.Fatalf("go list -deps: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	seen := 0
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		seen++
		pkg, imports := fields[0], fields[1:]
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

	// A scan that walks nothing passes every assertion it makes: the
	// closure holds at least validate itself and the two packages it is
	// allowed to reach in this module.
	if seen < 3 {
		t.Errorf("go list -deps reported %d packages, want validate plus at least xsd and xsderr", seen)
	}
}

// matchesPackage reports whether pkg is path or a package under it.
func matchesPackage(pkg, path string) bool {
	return pkg == path || strings.HasPrefix(pkg, path+"/")
}
