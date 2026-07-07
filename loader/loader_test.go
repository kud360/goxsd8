package loader

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// readAll drains and closes a resolved reader for assertions. Tests may read
// fully; the package itself never buffers (P4).
func readAll(t *testing.T, rc io.ReadCloser) string {
	t.Helper()
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading resolved body: %v", err)
	}
	return string(b)
}

func TestMapResolve(t *testing.T) {
	r := Map(map[string]string{"a.xsd": "<a/>"})

	rc, resolved, err := r.Resolve("urn:ns", "a.xsd")
	if err != nil {
		t.Fatalf("Resolve(a.xsd): %v", err)
	}
	if resolved != "a.xsd" {
		t.Errorf("resolved location = %q, want a.xsd", resolved)
	}
	if got := readAll(t, rc); got != "<a/>" {
		t.Errorf("body = %q, want <a/>", got)
	}

	_, _, err = r.Resolve("", "missing.xsd")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Resolve(missing) err = %v, want ErrNotFound", err)
	}
}

func TestMapCopiesInput(t *testing.T) {
	src := map[string]string{"a.xsd": "<a/>"}
	r := Map(src)
	src["a.xsd"] = "<mutated/>"
	delete(src, "a.xsd")

	rc, _, err := r.Resolve("", "a.xsd")
	if err != nil {
		t.Fatalf("Resolve after mutating source map: %v", err)
	}
	if got := readAll(t, rc); got != "<a/>" {
		t.Errorf("body = %q, want <a/> (source mutation must not leak)", got)
	}
}

func TestFSResolve(t *testing.T) {
	fsys := fstest.MapFS{"schemas/a.xsd": {Data: []byte("<a/>")}}
	r := FS(fsys)

	rc, resolved, err := r.Resolve("", "schemas/a.xsd")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved != "schemas/a.xsd" {
		t.Errorf("resolved = %q", resolved)
	}
	if got := readAll(t, rc); got != "<a/>" {
		t.Errorf("body = %q", got)
	}

	_, _, err = r.Resolve("", "schemas/missing.xsd")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("missing entry err = %v, want ErrNotFound", err)
	}
}

func TestDirResolve(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.xsd"), []byte("<a/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.xsd"), []byte("<b/>"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := Dir(root)

	rc, resolved, err := r.Resolve("", "a.xsd")
	if err != nil {
		t.Fatalf("Resolve(a.xsd): %v", err)
	}
	if resolved != filepath.Join(root, "a.xsd") {
		t.Errorf("resolved = %q", resolved)
	}
	if got := readAll(t, rc); got != "<a/>" {
		t.Errorf("body = %q", got)
	}

	// Legitimate nested relative path.
	rc, _, err = r.Resolve("", "sub/b.xsd")
	if err != nil {
		t.Fatalf("Resolve(sub/b.xsd): %v", err)
	}
	if got := readAll(t, rc); got != "<b/>" {
		t.Errorf("nested body = %q", got)
	}

	_, _, err = r.Resolve("", "missing.xsd")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("missing err = %v, want ErrNotFound", err)
	}
}

func TestDirRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	// A secret sitting beside (outside) the root.
	outside := filepath.Join(filepath.Dir(root), "secret.txt")
	if err := os.WriteFile(outside, []byte("SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })

	r := Dir(root)
	for _, loc := range []string{
		"../secret.txt",
		"../../etc/passwd",
		"sub/../../secret.txt",
	} {
		_, _, err := r.Resolve("", loc)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Resolve(%q) err = %v, want ErrNotFound (traversal must be refused)", loc, err)
		}
	}
}

func TestHTTPResolve(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/a.xsd":
			_, _ = w.Write([]byte("<a/>"))
		case "/gone.xsd":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	r := HTTP(srv.Client())

	rc, resolved, err := r.Resolve("", srv.URL+"/a.xsd")
	if err != nil {
		t.Fatalf("Resolve(a.xsd): %v", err)
	}
	if resolved != srv.URL+"/a.xsd" {
		t.Errorf("resolved = %q", resolved)
	}
	if got := readAll(t, rc); got != "<a/>" {
		t.Errorf("body = %q", got)
	}

	// 404 → ErrNotFound.
	_, _, err = r.Resolve("", srv.URL+"/gone.xsd")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("404 err = %v, want ErrNotFound", err)
	}

	// Non-404 non-2xx → real error, NOT ErrNotFound.
	_, _, err = r.Resolve("", srv.URL+"/boom.xsd")
	if err == nil {
		t.Fatal("500 status: want error")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("500 err = %v, must NOT be ErrNotFound", err)
	}
}

func TestHTTPNilClientUsesDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("<a/>"))
	}))
	defer srv.Close()

	r := HTTP(nil)
	rc, _, err := r.Resolve("", srv.URL+"/a.xsd")
	if err != nil {
		t.Fatalf("nil client Resolve: %v", err)
	}
	if got := readAll(t, rc); got != "<a/>" {
		t.Errorf("body = %q", got)
	}
}

func TestChainFirstHitWins(t *testing.T) {
	first := Map(map[string]string{"a.xsd": "<first/>"})
	second := Map(map[string]string{"a.xsd": "<second/>"})
	r := Chain(first, second)

	rc, _, err := r.Resolve("", "a.xsd")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := readAll(t, rc); got != "<first/>" {
		t.Errorf("body = %q, want <first/> (earliest resolver wins)", got)
	}
}

func TestChainFallsThroughNotFound(t *testing.T) {
	first := Map(map[string]string{"other.xsd": "<other/>"})
	second := Map(map[string]string{"a.xsd": "<second/>"})
	r := Chain(first, second)

	rc, resolved, err := r.Resolve("", "a.xsd")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved != "a.xsd" {
		t.Errorf("resolved = %q", resolved)
	}
	if got := readAll(t, rc); got != "<second/>" {
		t.Errorf("body = %q, want <second/>", got)
	}
}

func TestChainAllFailAggregates(t *testing.T) {
	r := Chain(
		Map(map[string]string{"x.xsd": "<x/>"}),
		Map(map[string]string{"y.xsd": "<y/>"}),
	)

	_, _, err := r.Resolve("urn:ns", "a.xsd")
	if err == nil {
		t.Fatal("all-fail: want error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("aggregated err = %v, want errors.Is ErrNotFound", err)
	}
}

func TestChainEmpty(t *testing.T) {
	_, _, err := Chain().Resolve("", "a.xsd")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("empty chain err = %v, want ErrNotFound", err)
	}
}

// errResolver returns a fixed non-ErrNotFound error, standing in for a
// permission-denied-style real failure.
type errResolver struct{ err error }

func (e errResolver) Resolve(namespace, location string) (io.ReadCloser, string, error) {
	return nil, "", e.err
}

func TestChainRealErrorShortCircuits(t *testing.T) {
	realErr := os.ErrPermission
	reached := false
	sentinel := ResolverFunc(func(namespace, location string) (io.ReadCloser, string, error) {
		reached = true
		return Map(map[string]string{"a.xsd": "<a/>"}).Resolve(namespace, location)
	})

	r := Chain(errResolver{err: realErr}, sentinel)
	_, _, err := r.Resolve("", "a.xsd")

	if !errors.Is(err, realErr) {
		t.Errorf("err = %v, want the real permission error returned as-is", err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("real error must NOT be reported as ErrNotFound")
	}
	if reached {
		t.Errorf("Chain reached the second resolver after a real error; it must short-circuit")
	}
}

func TestResolverFuncAdapts(t *testing.T) {
	var r Resolver = ResolverFunc(func(namespace, location string) (io.ReadCloser, string, error) {
		return nil, "", ErrNotFound
	})
	_, _, err := r.Resolve("", "x")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}
