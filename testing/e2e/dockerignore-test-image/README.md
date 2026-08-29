# Dockerignore

This represents a repo built with `--respect-dockerignore` (the default as of `v3`).
`build/.dockerignore` excludes `build/ignored/`, and the e2e test writes a fresh
random file into that directory on every run - the same way `base-changing-test-image`
does into `build/` itself. Because the ignored directory is excluded from both the
hash and the build context, the hash must stay stable across runs and this should
always hit the copy path, the same way `base-test-image` does.