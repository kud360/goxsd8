package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// writeTree materializes files (relative path -> source) under a fresh
// t.TempDir() and returns its root, for Surface to walk.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return dir
}

func TestSurface(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{
			name: "exported func found",
			files: map[string]string{
				"a.go": "package p\n\nfunc Foo() {}\n",
			},
			want: []string{"example.com/m func Foo"},
		},
		{
			name: "unexported func not found",
			files: map[string]string{
				"a.go": "package p\n\nfunc foo() {}\n",
			},
			want: nil,
		},
		{
			name: "exported type const and var found",
			files: map[string]string{
				"a.go": "package p\n\ntype Foo struct{}\n\nconst Bar = 1\n\nvar Baz = 2\n\ntype hidden struct{}\n\nconst quiet = 1\n\nvar mute = 2\n",
			},
			want: []string{
				"example.com/m const Bar",
				"example.com/m type Foo",
				"example.com/m var Baz",
			},
		},
		{
			name: "method on exported type found",
			files: map[string]string{
				"a.go": "package p\n\ntype Foo struct{}\n\nfunc (f *Foo) Bar() {}\n\nfunc (f Foo) Qux() {}\n\nfunc (f Foo) baz() {}\n",
			},
			want: []string{
				"example.com/m method (*Foo) Bar",
				"example.com/m method (Foo) Qux",
				"example.com/m type Foo",
			},
		},
		{
			name: "exported struct field on exported type found",
			files: map[string]string{
				"a.go": "package p\n\ntype Foo struct {\n\tBar int\n\tbaz int\n}\n",
			},
			want: []string{
				"example.com/m field Foo.Bar",
				"example.com/m type Foo",
			},
		},
		{
			name: "fields on an unexported type are not surface",
			files: map[string]string{
				"a.go": "package p\n\ntype foo struct {\n\tBar int\n}\n",
			},
			want: nil,
		},
		{
			name: "embedded exported field found",
			files: map[string]string{
				"a.go": "package p\n\ntype Base struct{}\n\ntype Foo struct {\n\tBase\n}\n",
			},
			want: []string{
				"example.com/m field Foo.Base",
				"example.com/m type Base",
				"example.com/m type Foo",
			},
		},
		{
			name: "_test.go file skipped",
			files: map[string]string{
				"a.go":      "package p\n",
				"a_test.go": "package p\n\nfunc Exported() {}\n",
			},
			want: nil,
		},
		{
			name: "subpackage import path derived from directory",
			files: map[string]string{
				"a.go":     "package p\n",
				"sub/b.go": "package sub\n\nfunc Foo() {}\n",
			},
			want: []string{"example.com/m/sub func Foo"},
		},
		{
			name: "internal package skipped",
			files: map[string]string{
				"internal/x.go": "package internalx\n\nfunc Foo() {}\n",
			},
			want: nil,
		},
		{
			name: "tools directory skipped",
			files: map[string]string{
				"tools/x.go": "package toolsx\n\nfunc Foo() {}\n",
			},
			want: nil,
		},
		{
			// conformance is repo infrastructure, not library
			// (docs/ARCHITECTURE.md's two tiers), so its exports are not
			// the surface a T5 review judges.
			name: "conformance package skipped",
			files: map[string]string{
				"conformance/x.go": "package conformance\n\nfunc Foo() {}\n",
			},
			want: nil,
		},
		{
			name: "testdata directory skipped",
			files: map[string]string{
				"testdata/x.go": "package testdatax\n\nfunc Foo() {}\n",
			},
			want: nil,
		},
		{
			name: "output sorted regardless of declaration order",
			files: map[string]string{
				"a.go": "package p\n\nfunc Zeta() {}\n\nfunc Alpha() {}\n\nfunc Mid() {}\n",
			},
			want: []string{
				"example.com/m func Alpha",
				"example.com/m func Mid",
				"example.com/m func Zeta",
			},
		},
		{
			name: "output sorted regardless of file walk order across files and packages",
			files: map[string]string{
				"zzz/z.go": "package zzz\n\nfunc Z() {}\n",
				"aaa/a.go": "package aaa\n\nfunc A() {}\n",
				"b.go":     "package p\n\nfunc B() {}\n",
			},
			want: []string{
				"example.com/m func B",
				"example.com/m/aaa func A",
				"example.com/m/zzz func Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeTree(t, tt.files)
			got, err := Surface(dir, "example.com/m")
			if err != nil {
				t.Fatalf("Surface: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Surface() = %v, want %v", got, tt.want)
			}
			if !slices.IsSorted(got) {
				t.Errorf("Surface() = %v, not sorted", got)
			}
		})
	}
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name        string
		base, cur   []string
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name:        "added and removed",
			base:        []string{"pkg func A", "pkg func B"},
			cur:         []string{"pkg func B", "pkg func C"},
			wantAdded:   []string{"pkg func C"},
			wantRemoved: []string{"pkg func A"},
		},
		{
			name:        "identical inputs report no change",
			base:        []string{"pkg func A", "pkg func B"},
			cur:         []string{"pkg func A", "pkg func B"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "unsorted input still produces sorted output",
			base:        []string{"pkg func Z", "pkg func A"},
			cur:         []string{"pkg func Q", "pkg func A"},
			wantAdded:   []string{"pkg func Q"},
			wantRemoved: []string{"pkg func Z"},
		},
		{
			name:        "empty base is all additions",
			base:        nil,
			cur:         []string{"pkg func B", "pkg func A"},
			wantAdded:   []string{"pkg func A", "pkg func B"},
			wantRemoved: nil,
		},
		{
			name:        "empty cur is all removals",
			base:        []string{"pkg func B", "pkg func A"},
			cur:         nil,
			wantAdded:   nil,
			wantRemoved: []string{"pkg func A", "pkg func B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdded, gotRemoved := Diff(tt.base, tt.cur)
			if !slices.Equal(gotAdded, tt.wantAdded) {
				t.Errorf("added = %v, want %v", gotAdded, tt.wantAdded)
			}
			if !slices.Equal(gotRemoved, tt.wantRemoved) {
				t.Errorf("removed = %v, want %v", gotRemoved, tt.wantRemoved)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	tests := []struct {
		name           string
		added, removed []string
		want           string
	}{
		{"no changes", nil, nil, "surface: unchanged"},
		{"additions only", []string{"a"}, nil, "surface: +1 -0"},
		{"removals only", nil, []string{"a", "b"}, "surface: +0 -2"},
		{"both directions", []string{"a", "b", "c"}, []string{"d"}, "surface: +3 -1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summary(tt.added, tt.removed); got != tt.want {
				t.Errorf("summary(%v, %v) = %q, want %q", tt.added, tt.removed, got, tt.want)
			}
		})
	}
}
