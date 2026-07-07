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
