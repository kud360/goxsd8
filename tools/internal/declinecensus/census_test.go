package declinecensus

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fixtureLog is two lanes' listings inside the noise a real -v log carries
// around them, the schema lane's decided cases charged.
const fixtureLog = `=== RUN   TestConformance
    conformance_test.go:172: lane schema: 9 cases
    conformance_test.go:245: lane schema: decline candidates: [A/g/schema/i1 B/g/schema/v1]
    conformance_test.go:246: lane schema: indeterminate declines: [A/g/schema/n1]
    conformance_test.go:247: lane schema: decided disagreements: [A/g/schema/i2=(accepted) A/g/schema/v1=src-ct]
    conformance_test.go:245: lane instance: decline candidates: []
    conformance_test.go:246: lane instance: indeterminate declines: [I/g/instance/n]
    conformance_test.go:247: lane instance: decided disagreements: [I/g/instance/d]
--- PASS: TestConformance (1.00s)
`

// logFile writes body to a temp file and returns its path.
func logFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "run.log")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// TestReadTakesOnlyTheNamedLanesLists holds the reader to one lane: the other
// lane's lists are in the same log and must not leak into it.
func TestReadTakesOnlyTheNamedLanesLists(t *testing.T) {
	path := logFile(t, fixtureLog)
	schema, err := Read(path, "schema")
	if err != nil {
		t.Fatalf("Read schema: %v", err)
	}
	if want := []string{"A/g/schema/i1", "B/g/schema/v1"}; !slices.Equal(schema.Declined, want) {
		t.Errorf("schema Declined = %v, want %v", schema.Declined, want)
	}
	if want := []string{"A/g/schema/n1"}; !slices.Equal(schema.Indeterminate, want) {
		t.Errorf("schema Indeterminate = %v, want %v", schema.Indeterminate, want)
	}
	if len(schema.Decided) != 2 || schema.Decided["A/g/schema/i2"] != "(accepted)" || schema.Decided["A/g/schema/v1"] != "src-ct" {
		t.Errorf("schema Decided = %v, want the two cases with their charges", schema.Decided)
	}
	instance, err := Read(path, "instance")
	if err != nil {
		t.Fatalf("Read instance: %v", err)
	}
	if len(instance.Declined) != 0 || !slices.Equal(instance.Indeterminate, []string{"I/g/instance/n"}) {
		t.Errorf("instance census = %+v, want no decline and one indeterminate case", instance)
	}
	if charge, ok := instance.Decided["I/g/instance/d"]; !ok || charge != "" {
		t.Errorf("instance Decided = %v, want I/g/instance/d uncharged", instance.Decided)
	}
}

// TestReadRefusesAnUnusableLog covers the errors: no file, a lane the log
// lists nothing for, a lane whose listing lacks a list, and a lane listed
// twice.
func TestReadRefusesAnUnusableLog(t *testing.T) {
	cases := []struct {
		name, path, lane, want string
	}{
		{"unreadable", filepath.Join(t.TempDir(), "absent.log"), "schema", "reading run log"},
		{"no listing for the lane", logFile(t, fixtureLog), "xpath", `carries no "decline candidates" listing for lane xpath`},
		{"list missing", logFile(t, "lane schema: decline candidates: []\n"), "schema", `carries no "indeterminate declines" listing for lane schema`},
		{"listed twice", logFile(t, fixtureLog+fixtureLog), "schema", `lane schema lists "decline candidates" twice`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Read(tc.path, tc.lane)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Read error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}
