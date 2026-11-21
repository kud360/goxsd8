# testdata

`xsdtests/` is the W3C XML Schema test suite — a pinned git submodule
(~215 MB), not vendored content:

```sh
git submodule update --init testdata/xsdtests
```

Conformance runs skip with a clear message when it is absent.

To bump the pin: advance the submodule, re-run the ratchet, and commit
the gitlink together with the expectation movement it causes — in the
same commit, so `git blame` on an expectations file always explains the
flip.

`readme/` holds every file the top-level README's CLI and Library examples
name. The examples after the first run from it; the first writes its files
with heredocs into a fresh directory, and the copies here are the ones
`TestReadmeTranscripts` in `cmd/goxsd8/readme_test.go` runs its transcripts
against. `TestReadmeFixtures` in the root `readme_test.go` holds each file
README shows to the text README shows for it, and its doc comment names
them; `TestReadmeSharedDocument` does the same for the two-target `gen`
example's files. `main.go` is the Library program README's Library section names,
and `TestReadmeProgram` holds it to README's blocks.
