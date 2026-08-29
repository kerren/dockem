# Secret

This exercises `--secret`. Like `base-changing-test-image`, a tempfile is written
into `build/` on every run so the hash is always new and the build path actually
runs rather than being short-circuited by a cache hit.

The Dockerfile **fails** unless a non-empty secret is mounted at
`/run/secrets/dockem_e2e_secret`, so a green build is itself the assertion that
the secret reached BuildKit — no image introspection needed. Nothing secret is
ever copied into the image, and the value the test supplies is a visible
non-secret string, so a leak into a CI log would be both harmless and obvious.
