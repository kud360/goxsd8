package main

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pinnedLog is the GOXSD_DECLINES=1 census `go tool lanepartition` pins #1740
// against: the schema and instance listings of a b52ce09 run. It lives with
// that tool's fixtures, and this test reads it there rather than keeping a
// second copy.
const pinnedLog = "../lanepartition/testdata/b52ce09-declines.log.gz"

// pinnedSuite and pinnedExpectations are the suite submodule and the committed
// expectations directory from this package's directory.
const (
	pinnedSuite        = "../../testdata/xsdtests"
	pinnedExpectations = "../../conformance/testdata/expectations"
)

// pinnedIndeterminate are the five instance cases the b52ce09 run lists as
// indeterminate declines, and pinnedPaths the schema documents they are
// assessed against. Without a log the join counts all five as candidates
// (#1785).
var (
	pinnedIndeterminate = []string{
		"MS-Attribute2006-07-15/attP029/instance/attP029.v",
		"MS-Attribute2006-07-15/attP031/instance/attP031.i",
		"MS-Schema2006-07-15/schA2/instance/schA2.i",
		"MS-Schema2006-07-15/schA5/instance/schA5.i",
		"MS-Schema2006-07-15/schA7/instance/schA7.i",
	}
	pinnedPaths = []string{
		"msData/attribute/attP029.xsd",
		"msData/attribute/attP031.xsd",
		"msData/schema/schA2_a.xsd",
		"msData/schema/schA5_a.xsd",
		"msData/schema/schA7_a.xsd",
	}
)

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

// runPinned joins the pinned paths against the committed instance lane over
// the live suite catalog, with the pinned log when withLog is set.
func runPinned(t *testing.T, withLog bool) string {
	t.Helper()
	if _, err := os.Stat(filepath.Join(pinnedSuite, suiteIndexName)); err != nil {
		t.Skip("suite submodule absent: git submodule update --init testdata/xsdtests")
	}
	args := []string{"-suite", pinnedSuite, "-expectations", pinnedExpectations}
	if withLog {
		log := filepath.Join(t.TempDir(), "run.log")
		gunzipTo(t, pinnedLog, log)
		args = append(args, "-log", log)
	}
	args = append(append(args, "join", "instance"), pinnedPaths...)
	var out strings.Builder
	if err := run(&out, strings.NewReader(""), args); err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	return out.String()
}

// TestPinnedInstanceJoinWithoutALogCountsTheIndeterminateCases is the defect
// #1785 measured: with no log, all five indeterminate cases are candidates.
func TestPinnedInstanceJoinWithoutALogCountsTheIndeterminateCases(t *testing.T) {
	got := runPinned(t, false)
	wantLines(t, got, "casejoin: 5 path(s) → 10 catalog entry(ies) → 5 candidate case(s) in lane instance")
	wantLines(t, got, "\n  "+strings.Join(pinnedIndeterminate, "\n  ")+"\n")
	wantRow(t, got, "banked fail — CANDIDATE", 5)
}

// TestPinnedInstanceJoinWithTheLogSubtractsThem is #1785's pinned acceptance:
// fed the b52ce09 log, the five move to the indeterminate row and no candidate
// remains.
func TestPinnedInstanceJoinWithTheLogSubtractsThem(t *testing.T) {
	got := runPinned(t, true)
	wantLines(t, got, "casejoin: 5 path(s) → 10 catalog entry(ies) → 0 candidate case(s) in lane instance")
	wantRow(t, got, indeterminateRow, 5)
	wantRow(t, got, "banked fail — CANDIDATE", 0)
	for _, id := range pinnedIndeterminate {
		if strings.Contains(got, "\n  "+id+"\n") {
			t.Errorf("%s is listed as a candidate:\n%s", id, got)
		}
	}
}
