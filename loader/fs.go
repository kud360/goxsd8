package loader

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
)

// fsResolver serves locations as entries of an fs.FS; see FS.
type fsResolver struct {
	fsys fs.FS
}

// FS returns a Resolver that serves each location as a path within fsys
// (an embed.FS, os.DirFS, testing fstest.MapFS, etc.).
//
// FS refuses a location that fs.ValidPath rejects, such as "/m.xsd",
// "./m.xsd" or "a/../m.xsd", before calling fsys.Open, so the refusal does
// not depend on what fsys would answer; like Dir's refusal of a location
// that escapes its root, it wraps ErrNotFound (an engineering decision, not
// a spec rule). A valid location that fsys answers with fs.ErrNotExist also
// maps to ErrNotFound. A caller that needs to tell this refusal from an
// absent location calls fs.ValidPath on the location it passed.
//
// Because the refusal wraps ErrNotFound, [Chain] tries its next resolver, and
// package parser treats a refused location as one that does not resolve. A
// directive whose resolved location FS refuses is recorded as unfollowed with
// reason [github.com/kud360/goxsd8/parser.UnfollowedLocationUnresolved], and
// for a non-empty <redefine> it is also an error (src-redefine clause 1). A
// [github.com/kud360/goxsd8/parser.HintAt] root so refused is recorded as an
// unfollowed root, and a [github.com/kud360/goxsd8/parser.RootAt] root so
// refused is an error.
//
// Unlike Dir, FS does NOT canonicalize the resolved location it returns: it
// hands back location as given. fs.FS guarantees no directory-listing
// capability in general, so there is nothing to canonicalize against; when
// fsys is case-insensitive (an os.DirFS over a case-insensitive filesystem,
// say), keeping Resolver's one-resource-one-identity contract by spelling
// locations consistently is the caller's responsibility.
func FS(fsys fs.FS) Resolver {
	return fsResolver{fsys: fsys}
}

// Resolve implements Resolver.
func (r fsResolver) Resolve(namespace, location string) (io.ReadCloser, string, error) {
	if !fs.ValidPath(location) {
		return nil, "", invalidPathError{location}
	}
	f, err := r.fsys.Open(location)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, "", fmt.Errorf("loader: %q not found in fs: %w", location, ErrNotFound)
	}
	if err != nil {
		return nil, "", fmt.Errorf("loader: opening %q in fs: %w", location, err)
	}
	// f streams from fsys; the caller closes it (P4 — no buffering here).
	return f, location, nil
}

// invalidPathError is the refusal FS's doc describes; see FS.
type invalidPathError struct{ location string }

// Error names the location and fs.ValidPath, without ErrNotFound's text.
func (e invalidPathError) Error() string {
	return fmt.Sprintf("loader: %q is not a valid fs.FS path (fs.ValidPath)", e.location)
}

// Unwrap returns ErrNotFound, for Chain's and parser's errors.Is reads.
func (e invalidPathError) Unwrap() error { return ErrNotFound }
