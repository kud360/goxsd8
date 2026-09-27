package main

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The pinned fixtures (#1740 Acceptance), each gzipped:
//
//   - instance-61db25a.fails.gz: the `fail` lines of
//     conformance/testdata/expectations/instance.txt at 61db25a;
//   - schema-b52ce09.fails.gz: the `fail` lines of schema.txt at b52ce09;
//   - b52ce09-declines.log.gz: the schema and instance census listings of
//     `GOXSD_DECLINES=1 go test ./conformance -run TestConformance -count=1
//     -timeout 30m -v` run on b52ce09 with this issue's census listing
//     applied (its conformance code is otherwise identical to the commit that
//     introduced the listing), over the suite pinned then and now.
//
// A banked pass decides no figure here, so the lane files carry only fails.
const (
	fixtureInstance61db25a = "testdata/instance-61db25a.fails.gz"
	fixtureSchemaB52ce09   = "testdata/schema-b52ce09.fails.gz"
	fixtureLogB52ce09      = "testdata/b52ce09-declines.log.gz"
)

// pinnedSuite is the suite submodule from this package's directory.
const pinnedSuite = "../../testdata/xsdtests"

// gunzipTo decompresses a fixture to dst.
func gunzipTo(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("opening %s: %v", src, err)
	}
	defer func() { _ = in.Close() }()
	zr, err := gzip.NewReader(in)
	if err != nil {
		t.Fatalf("reading %s: %v", src, err)
	}
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("decompressing %s: %v", src, err)
	}
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		t.Fatalf("writing %s: %v", dst, err)
	}
}

// runPinned runs the command over the pinned fixtures and the live suite
// catalog, which has not moved since either pin.
func runPinned(t *testing.T, withLog bool, lane string) string {
	t.Helper()
	if _, err := os.Stat(filepath.Join(pinnedSuite, "suite.xml")); err != nil {
		t.Skip("suite submodule absent: git submodule update --init testdata/xsdtests")
	}
	dir := t.TempDir()
	gunzipTo(t, fixtureInstance61db25a, filepath.Join(dir, "instance.txt"))
	gunzipTo(t, fixtureSchemaB52ce09, filepath.Join(dir, "schema.txt"))
	args := []string{"-suite", pinnedSuite, "-expectations", dir}
	if withLog {
		log := filepath.Join(dir, "run.log")
		gunzipTo(t, fixtureLogB52ce09, log)
		args = append(args, "-log", log)
	}
	var out strings.Builder
	if err := run(&out, strings.NewReader(""), append(args, lane)); err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	return out.String()
}

// TestInstanceAt61db25aSplitsDeclaredValidFromCandidates reproduces the
// instance bullet: 14,695 declared-valid banked fails, which could not flip
// at 61db25a (#1561), and 419 candidates, from the catalog alone.
func TestInstanceAt61db25aSplitsDeclaredValidFromCandidates(t *testing.T) {
	got := runPinned(t, false, "instance")
	wantTableRow(t, got, "banked fail", "14695", "419", "15114")
}

// TestInstanceAtB52ce09SplitsDeclinesFromDecided reproduces the instance side
// of the b52ce09 run: 15,114 = 15,103 declined + 5 indeterminate + 6 decided,
// the 419 candidates splitting into 414 invalid and the 5 indeterminate.
func TestInstanceAtB52ce09SplitsDeclinesFromDecided(t *testing.T) {
	got := runPinned(t, true, "instance")
	wantTableRow(t, got, "declined", "14689", "414", "0", "15103")
	wantTableRow(t, got, "indeterminate", "0", "0", "5", "5")
	wantTableRow(t, got, "decided and wrong", "6", "0", "0", "6")
}

// TestSchemaAtB52ce09SplitsDeclinesFromDecidedByCharge reproduces the schema
// bullet: 699 = 126 declined + 11 indeterminate + 562 decided, the 562 being
// 8 suite-valid and 554 suite-invalid — 551 plus the 3 PrecisionDecimalTests
// cases — and every decided case clustered under a charge.
func TestSchemaAtB52ce09SplitsDeclinesFromDecidedByCharge(t *testing.T) {
	got := runPinned(t, true, "schema")
	wantTableRow(t, got, "declined", "81", "45", "0", "126")
	wantTableRow(t, got, "indeterminate", "0", "0", "11", "11")
	wantTableRow(t, got, "decided and wrong", "8", "554", "0", "562")
	rows := clusterLines(got)
	found := false
	for _, r := range rows {
		if r == "3 decided and wrong invalid PrecisionDecimalTests [(accepted)]" {
			found = true
		}
		if strings.Contains(r, "decided and wrong") && !strings.Contains(r, "[") {
			t.Errorf("decided cluster with no charge: %q", r)
		}
	}
	if !found {
		t.Errorf("no PrecisionDecimalTests cluster of 3 in:\n%s", got)
	}
}
