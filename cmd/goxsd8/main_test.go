package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestRunHelp pins issue #251: a help request — including the bare
// invocation — never reaches the not-yet-implemented stub. Usage goes to
// stdout, stderr stays empty, exit code 0.
func TestRunHelp(t *testing.T) {
	cases := [][]string{
		nil,
		{"-h"},
		{"-help"},
		{"--help"},
		{"validate", "-help"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr)
		if code != 0 {
			t.Errorf("run(%q) = %d, want 0", args, code)
		}
		if stdout.String() != usage {
			t.Errorf("run(%q) stdout = %q, want the usage contract", args, stdout.String())
		}
		if stderr.Len() != 0 {
			t.Errorf("run(%q) stderr = %q, want empty", args, stderr.String())
		}
	}
}

// TestRunNotImplemented pins the other half of #251: anything that is not a
// help request keeps the pre-fix behavior exactly.
func TestRunNotImplemented(t *testing.T) {
	cases := [][]string{
		{"parse", "schema.xsd"},
		{"frobnicate"},
		{"-q"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr)
		if code != 2 {
			t.Errorf("run(%q) = %d, want 2", args, code)
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%q) stdout = %q, want empty", args, stdout.String())
		}
		if got := strings.TrimSuffix(stderr.String(), "\n"); got != notImplemented {
			t.Errorf("run(%q) stderr = %q, want %q", args, got, notImplemented)
		}
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("closed pipe") }

// TestRunHelpWriteFailure covers the usage/IO exit code: help the user never
// received is not a success.
func TestRunHelpWriteFailure(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"-help"}, errWriter{}, &stderr); code != 2 {
		t.Errorf("run with a failing stdout = %d, want 2", code)
	}
}

// TestUsageCoversContract guards the usage constant against drifting away
// from the doc.go contract it renders.
func TestUsageCoversContract(t *testing.T) {
	want := []string{
		"goxsd8 parse <schema.xsd>...",
		"goxsd8 validate -schema <schema.xsd>... <instance>...",
		"goxsd8 gen -schema <schema.xsd> -out <dir>",
		"GOXSD_DEBUG=parser,validate,codec",
	}
	for _, w := range want {
		if !strings.Contains(usage, w) {
			t.Errorf("usage is missing %q", w)
		}
	}
}
