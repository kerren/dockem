# Dockem Development Plan

This document tracks the next major body of work on `dockem`. It is organised into
phases, each with a concrete checklist. Phases are ordered by dependency — Phase 0
is groundwork that later phases build on.

**Status legend:** `[ ]` not started · `[~]` in progress · `[x]` done

---

## Goals

Four outcomes, in priority order:

1. **Respect `.dockerignore`** — today it is honoured neither in the hash nor in the
   build context tar, which causes spurious cache misses and slow uploads.
2. **Machine-readable output** — emit the hash, cache-hit status and resolved tags as
   JSON and to `$GITHUB_OUTPUT` so pipelines can consume the result.
3. **Fix `CheckManifestHead` error handling** — any registry error is currently treated
   as "image does not exist", which silently triggers a full rebuild and push.
4. **BuildKit + multi-platform builds** — move off the deprecated classic daemon builder
   and support `--platform linux/amd64,linux/arm64`.

### Non-goals for this cycle

- Monorepo config file (`dockem.yaml`) and parallel multi-image builds.
- Base-image digest watching (`--watch-base-image`). Tracked separately; see
  [Backlog](#backlog).
- Homebrew tap.

---

## Versioning strategy

| Phase | Release | Breaking? |
|---|---|---|
| 0 — Groundwork | internal | No |
| 1 — `CheckManifestHead` | `v2.6.0` | No |
| 2 — Machine-readable output | `v2.6.0` | No |
| 3 — `.dockerignore` | `v3.0.0` | **Yes** — changes cache identity |
| 4 — BuildKit / multi-platform | `v3.0.0` | **Yes** — requires `docker buildx` |

Phases 1 and 2 ship first as a non-breaking `v2.6.0` so the output contract is
available before the cache reset lands. Phases 3 and 4 ship together as `v3.0.0`.

> **Cache identity warning.** Phase 3 changes the set of files that feed the hash.
> Every previously published hash tag becomes unreachable and the first build after
> upgrading will be a full rebuild for every image. This must be called out
> prominently in the release notes and the README.

---

## The branch ladder

The phases below describe *what* to build. This section describes *how to land it* —
a stack of small branches, each rung sitting on the one below, so that no single pull
request is large enough to be reviewed badly.

All branches use the `feature/` prefix so the git flow tooling picks them up. The base
of the stack is `develop`; releases merge `develop` into `main` and tag, as usual.

```
  develop
     │
     ├──▶  RUNG 1   feature/log-helpers
     │        │                                   Phase 0.1
     │        │
     │        ├──▶  RUNG 2   feature/resolve-target-tags
     │        │                                   Phase 0.2
     │        │
     │        └──▶  RUNG 3   feature/manifest-head-errors
     │                 │                          Phase 1
     │                 │
     │                 └──▶  RUNG 4   feature/build-result
     │                                            Phase 2
     ▼
  ═══════  RELEASE  v2.6.0  ═══════════════════════════════════
     │
     ├──▶  RUNG 5   feature/dockerignore-hashing
     │        │                                   Phase 3.1 – 3.3
     │        │
     │        └──▶  RUNG 6   feature/dockerignore-default
     │                 │                          Phase 3.4 – 3.5
     │                 │
     │                 └──▶  RUNG 7   feature/buildx-detect
     │                          │                 Phase 4.2, 4.5
     │                          │
     │                          └──▶  RUNG 8   feature/buildx-build
     │                                   │        Phase 4.3, 4.4, 4.6
     │                                   │
     │                                   └──▶  RUNG 9   feature/buildx-cache
     ▼                                                   Phase 4.7
  ═══════  RELEASE  v3.0.0  ═══════════════════════════════════
```

Rungs 2 and 3 are the only pair that can be worked in parallel — they both sit on
rung 1 but touch disjoint files. Everything else is strictly sequential.

### Rung summary

| # | Branch | Base | Phase | Breaking | Size |
|---|---|---|---|---|---|
| 1 | `feature/log-helpers` | `develop` | 0.1 | No | Wide, shallow |
| 2 | `feature/resolve-target-tags` | rung 1 | 0.2 | No | Small |
| 3 | `feature/manifest-head-errors` | rung 1 | 1 | No | Small |
| 4 | `feature/build-result` | rung 3 | 2 | No | Medium |
| — | **Release `v2.6.0`** | `develop` → `main` | — | No | — |
| 5 | `feature/dockerignore-hashing` | `develop` | 3.1 – 3.3 | No (opt-in) | Medium |
| 6 | `feature/dockerignore-default` | rung 5 | 3.4 – 3.5 | **Yes** | Small |
| 7 | `feature/buildx-detect` | rung 6 | 4.2, 4.5 | No | Medium |
| 8 | `feature/buildx-build` | rung 7 | 4.3, 4.4, 4.6 | **Yes** | Large |
| 9 | `feature/buildx-cache` | rung 8 | 4.7 | No | Small |
| — | **Release `v3.0.0`** | `develop` → `main` | — | **Yes** | — |

---

### Rung 1 — `feature/log-helpers`

**Base:** `develop` · **Covers:** Phase 0.1 · `feat(refactor):`

Introduce `cli/utils/log.go` and mechanically replace every `fmt.Printf` / `fmt.Print`
in `cli/utils/` with `LogInfo` / `LogWarn` / `LogError`, all writing to stderr.

- **Why first:** rung 4 needs a clean stdout for JSON, and this touches nearly every
  file in `utils/`. Landing it alone means it never collides with a substantive change.
- **Review focus:** that no message wording changed. A reviewer should be able to read
  the diff as a pure destination swap.
- **Exit criteria:** `task test` green; `dockem build … 1>/dev/null` still shows all
  progress output.
- **Revert risk:** trivial, no behaviour depends on it yet.

### Rung 2 — `feature/resolve-target-tags`

**Base:** rung 1 · **Covers:** Phase 0.2 · `feat(refactor):`

Add `ResolveTargetTags` and rewrite `CopyExistingImageTag` and `TagAndPushNewImages`
to consume it.

- **Why here:** rung 8 needs the resolved tag list *before* the build starts, because
  `buildx build --push` takes every `-t` in one invocation. Doing it now, while the two
  existing implementations are still the only callers, means the refactor can be proven
  correct against the current e2e suite before buildx complicates it.
- **Review focus:** the four branches of the tag rule, and that `buildLog.outputTags`
  ordering is byte-identical to before — the existing e2e assertions depend on it.
- **Exit criteria:** new pure unit tests pass with no credentials; the full e2e suite
  passes unchanged, with no edits to `build_docker_image_test.go`.
- **Revert risk:** low, but this is the rung most likely to surface a latent
  behavioural difference between the copy path and the push path. If it does, that is a
  bug find, not a blocker — fix it here.

### Rung 3 — `feature/manifest-head-errors`

**Base:** rung 1 · **Covers:** Phase 1 · `fix(registry):`

Return `(bool, error)` from `CheckManifestHead`, classify with `errors.Is` against
`errs.ErrNotFound` / `ErrHTTPUnauthorized` / `ErrHTTPRateLimit`, and add
`--strict-registry`.

- **Parallel with rung 2.** Disjoint files. Whichever lands second rebases onto the first.
- **Review focus:** that the default path is behaviourally unchanged — an unauthenticated
  or flaky registry must still fall through to a build unless `--strict-registry` is set.
  This rung must not change what happens today for anyone who does not opt in.
- **Exit criteria:** `httptest` unit tests cover 404 / 401 / 429 / 500 and need no real
  registry; e2e suite unchanged.
- **Revert risk:** low. The signature change is contained to one caller.

### Rung 4 — `feature/build-result`

**Base:** rung 3 · **Covers:** Phase 2 · `feat(output):`

`BuildResult`, `WriteBuildOutput`, `WriteGitHubOutput`, and the `--output-format` /
`--output-file` flags.

- **Why on rung 3:** both touch `BuildLog`, `BuildDockerImageParams`, `cmd/build.go` and
  `build_docker_image.go`. Stacking avoids a conflict-heavy merge.
- **Review focus:** that `--output-format=json` puts *only* JSON on stdout — this is the
  contract downstream pipelines depend on, and it is only true because of rung 1. Also
  that `$GITHUB_OUTPUT` is opened append-only; truncating it would destroy other steps'
  output.
- **Exit criteria:** `dockem build --output-format=json | jq .` succeeds; a temp-file
  `$GITHUB_OUTPUT` test asserts exact key/value lines.
- **Revert risk:** low. Purely additive; the default `text` format is today's behaviour.
- **Follow-up:** open the issue on `kerren/setup-dockem` to expose these as action
  outputs once this is released.

---

### Release `v2.6.0`

Merge `develop` → `main`, tag, `task release`. Everything to this point is additive and
no hash changes, so existing users upgrade with no cache impact. Getting the output
contract published *before* the cache reset means pipelines can adopt `cache-hit` and
`primary-tag` while their caches are still warm.

---

### Rung 5 — `feature/dockerignore-hashing`

**Base:** `develop` (post-`v2.6.0`) · **Covers:** Phase 3.1 – 3.3 · `feat(hash):`

`ReadDockerignore`, `HashDirectory`, the `ExcludePatterns` wiring in `TarBuildContext`,
and the `--respect-dockerignore` / `--ignore-file` / `--exclude` flags — with
`--respect-dockerignore` defaulting to **`false`**.

- **Deliberately opt-in on this rung.** Shipping the capability with the old default
  keeps this branch non-breaking and independently mergeable, and it lets the new hashing
  path be exercised against a real registry before it becomes mandatory.
- **Review focus:** that the tar and the hash derive from the *same* pattern list. If they
  diverge, dockem builds something other than what it hashed, which is the worst failure
  this tool can have. Also the two carve-outs: `Dockerfile` and `.dockerignore` stay in
  the hash regardless of patterns, and the temporary `Dockerfile.` written for the
  out-of-context case must survive a `Dockerfile*` pattern.
- **Exit criteria:** new `t.TempDir()` unit tests pass with no credentials; with the flag
  off, hashes are bit-identical to `v2.6.0`.
- **Revert risk:** low while the default is off.

### Rung 6 — `feature/dockerignore-default`

**Base:** rung 5 · **Covers:** Phase 3.4 – 3.5 · `feat(hash)!:`

Flip `--respect-dockerignore` to default `true`, add the `dockem-hash-v2` prefix
constant, add the `testing/e2e/dockerignore-test-image/` fixture, and write the README
cache-identity section.

- **This is the cache reset.** Small diff, large consequence — which is exactly why it
  is its own rung. A reviewer looking at three lines and a fixture can reason about
  "every published hash tag is now unreachable" far better than they could inside
  rung 5's diff.
- **Review focus:** the breaking-change note. Commit must carry `!` and a
  `BREAKING CHANGE:` footer so `commit-and-tag-version` picks up the major bump.
- **Exit criteria:** e2e fixture writes a random file into an ignored directory and the
  hash still hits the copy path across runs.
- **Revert risk:** the flag default is a one-line revert, but any hash tag published
  from this rung onward is orphaned by reverting. Do not release this rung and then
  reverse it.

### Rung 7 — `feature/buildx-detect`

**Base:** rung 6 · **Covers:** Phase 4.2, 4.5 · `feat(buildx):`

`DetectBuildx`, the `--platform` / `--builder` flags, builder resolution logic, and
`TempDockerConfig` for subprocess credentials. **No build path yet** — `--builder`
resolves and logs its decision, then falls through to the classic builder.

- **Why split from rung 8:** credential handling deserves its own review. Writing a
  `config.json` containing a password to disk is the highest-risk code in this cycle and
  should not be buried inside a large build-path diff.
- **Review focus:** the temp config directory is `0600`, is always removed via `defer`
  including on failure, `DOCKER_CONFIG` is set on the subprocess environment only and
  never exported into the parent, and the password reaches neither `BuildLog` nor the
  rung 4 JSON output. Also: multi-platform with no buildx must **error**, not silently
  fall back — a single-arch image published under a multi-arch-looking tag is worse than
  a failed build.
- **Exit criteria:** unit test for the error case; e2e suite unchanged, since no build
  behaviour has changed yet.
- **Revert risk:** low, nothing depends on it yet.

### Rung 8 — `feature/buildx-build`

**Base:** rung 7 · **Covers:** Phase 4.3, 4.4, 4.6 · `feat(buildx)!:`

`BuildImageBuildx`, the platform list folded into `overallHash`, and the manifest-list
copy verification.

- **The largest rung, and unavoidably so** — the build path and its hash input cannot be
  landed separately without an intermediate state where multi-arch builds are published
  under a hash that ignores the platform list. That state would poison the cache.
- **Review focus:** platforms sorted and joined before hashing, and nothing appended when
  `--platform` is unset so non-adopters see no extra invalidation. Then the buildx path
  bypassing `TagAndPushImage` / `TagAndPushNewImages` entirely — confirm every tag rung 2
  resolves actually reaches the registry. Finally 4.6: prove regclient's `ImageCopy`
  copies an image *index*, not one manifest. That is the single assumption which, if
  wrong, silently publishes a broken multi-arch tag.
- **Exit criteria:** two-platform e2e build; re-run hits the hash; `ManifestGet` on the
  copied tag resolves for both platforms; `--builder=docker` still exercises the classic
  path unchanged.
- **Revert risk:** high. This changes both what is built and how the cache is keyed. Do
  not stack anything but rung 9 on it before it has been exercised against a real
  multi-arch registry.

### Rung 9 — `feature/buildx-cache`

**Base:** rung 8 · **Covers:** Phase 4.7 · `feat(buildx):`

`--cache-from` / `--cache-to`, passed through verbatim.

- **Why last:** it is pure passthrough and worth nothing until rung 8 works. Splitting it
  off keeps rung 8's diff focused on correctness rather than ergonomics.
- **Review focus:** that these are documented as affecting build *speed* only and are
  deliberately excluded from the hash.
- **Exit criteria:** a GitHub Actions run using `type=gha` completes and reuses cache on
  a second run.
- **Revert risk:** trivial.

---

### Release `v3.0.0`

Merge `develop` → `main`, `task release-major`. Release notes must lead with the two
breaking changes:

1. Every previously published hash tag is invalidated. The first v3 build of each image
   is a full rebuild.
2. `docker buildx` is now required for multi-platform builds; `--builder=docker`
   preserves the classic single-platform path.

---

### Ladder discipline

- **Rebase, never merge, within the stack.** When a lower rung changes after review,
  rebase every rung above it. Merging `develop` into a mid-stack branch makes the
  diffs unreadable for everyone above.
- **One rung in review at a time**, except the rung 2 / rung 3 pair. Reviewing a stack
  concurrently means re-reviewing after every rebase.
- **A rung must be green on its own.** `task test` passes at every rung, not just at the
  top of the stack — otherwise a bisect through this range is worthless.
- **If a rung is rejected**, the rungs above it rebase onto its base and continue. This
  is why rung 5 ships opt-in and rung 6 flips the default: if the ordering has to change,
  only rung 6 is at risk, and it is three lines.
- **Do not batch rungs into one pull request to save time.** The stack exists because
  rung 8 is genuinely difficult to review, and the only way to make it reviewable is for
  everything mechanical to already be behind it.

---

## Phase 0 — Groundwork

Two refactors that Phases 2 and 4 both depend on. Doing them first keeps the later
diffs small and reviewable.

### 0.1 Route human-readable logging to stderr

Phase 2 needs stdout to be clean so JSON can be piped. Today every message is a bare
`fmt.Printf` to stdout.

- [x] Add `cli/utils/log.go` exposing `LogInfo(format string, a ...any)`,
      `LogWarn(...)` and `LogError(...)`, all writing to `os.Stderr`.
      `LogWarn`/`LogError` prepend the existing `WARN: ` / `ERROR: ` prefixes.
- [x] Replace every `fmt.Printf` / `fmt.Print` in `cli/utils/*.go` with the new helpers.
      Keep the wording of each message identical — only the destination changes.
- [x] Leave the `panic(err)` in `cli/cmd/build.go` as-is; error propagation is unchanged.
- [x] Confirm `jsonmessage.DisplayJSONMessagesStream` calls in `build_image.go` and
      `tag_and_push_image.go` already target `os.Stderr` (they do) — no change needed.
- [x] Update the "Conventions" section of `CLAUDE.md` to reference the helpers instead
      of raw `fmt.Printf`.

### 0.2 Extract the tag-resolution rule into one function

`CopyExistingImageTag` and `TagAndPushNewImages` each implement the same rule, and
`CLAUDE.md` already flags that they must not drift. Phase 4 adds a third consumer
(buildx needs the full tag list *before* the build starts), so the duplication has to go.

The rule, unchanged: each `--tag X` becomes `X-vVERSION`; `--latest` adds `latest`;
`--main-version` adds `vVERSION`; if none of tags/latest/main-version were given, it
falls back to `vVERSION`.

- [x] Add `cli/utils/resolve_target_tags.go` with
      `ResolveTargetTags(params BuildDockerImageParams, version string) []string`
      returning fully-qualified image names via `GenerateDockerImageName`.
- [x] Add `cli/utils/resolve_target_tags_test.go` — a pure unit test (no registry
      needed) covering all four branches and their combinations:
      tags only; `--latest` only; `--main-version` only; none; tags + latest + main.
- [x] Rewrite `CopyExistingImageTag` to iterate `ResolveTargetTags`.
- [x] Rewrite `TagAndPushNewImages` to iterate `ResolveTargetTags`.
- [x] Preserve the per-branch log lines (the "WARN: No tags were specified…" message in
      particular) by returning a small struct or by having the resolver record the reason
      for each tag.
- [x] Verify `buildLog.outputTags` ordering is unchanged so the existing e2e assertions
      in `build_docker_image_test.go` still pass.

---

## Phase 1 — Fix `CheckManifestHead` error handling

**File:** `cli/utils/check_manifest_head.go`

`CheckManifestHead` returns a bare `bool` and maps *every* error to `false`. An expired
token, a rate limit or a registry 5xx is therefore indistinguishable from a genuine
404. The result is a full rebuild and push — and if the cause was auth, the push then
fails too, after the build has already been paid for.

`regclient/types/errs` (already available in v0.6.0) provides `ErrNotFound`,
`ErrHTTPUnauthorized`, `ErrHTTPRateLimit` and `ErrHTTPStatus`.

- [x] Change the signature to
      `CheckManifestHead(tag string, ref ref.Ref, client *regclient.RegClient) (bool, error)`.
- [x] Classify with `errors.Is`:
  - [x] `errs.ErrNotFound` → `(false, nil)`. This is the normal "build it" path.
  - [x] `errs.ErrHTTPUnauthorized` → `(false, err)` and log a loud `ERROR:` telling the
        user to check `--docker-username` / `--docker-password` or their `docker login`.
  - [x] `errs.ErrHTTPRateLimit` → `(false, err)` with a message about registry limits.
  - [x] Anything else → `(false, err)` with the raw error.
- [x] Delete the `strings.Contains(manifestError.Error(), "failed to request manifest head")`
      string-matching heuristic — it is replaced by the typed checks.
- [x] Add a `StrictRegistry bool` field to `BuildDockerImageParams`.
- [x] Add the `--strict-registry` flag to `cli/cmd/build.go` (default `false`).
- [x] In `BuildDockerImage`, on a non-nil error from `CheckManifestHead`:
  - [x] If `params.StrictRegistry` → return the error and abort.
  - [x] Otherwise → log `WARN: … the build will continue, but this should be investigated`
        and proceed to build (preserves today's behaviour by default).
- [x] Record the outcome on `BuildLog`: add `headCheckError error` and
      `headCheckSkipped bool` so tests can assert on the branch taken.
- [x] Add unit tests in `cli/utils/check_manifest_head_test.go` using a stub registry
      (`httptest.Server` returning 404 / 401 / 429 / 500) to confirm the classification.
      These need no real registry credentials.
- [x] Document `--strict-registry` in the README flag table.

---

## Phase 2 — Machine-readable output

Everything needed is already collected on `BuildLog` — it is just unexported and
described as test-only. Rather than exporting it wholesale, build a dedicated result
type at the end of the pipeline.

**Why this matters:** a pipeline currently has no way to learn the tag it should deploy,
or whether a build actually happened, short of scraping prose from stdout. Since
`kerren/setup-dockem` already exists, wiring `$GITHUB_OUTPUT` makes the action
immediately more useful.

### 2.1 The result type

- [x] Add `cli/utils/build_result.go` with an exported `BuildResult`:

  ```go
  type BuildResult struct {
      Hash       string   `json:"hash"`
      CacheHit   bool     `json:"cacheHit"`
      Image      string   `json:"image"`      // hashed image name
      Version    string   `json:"version"`    // e.g. v1.0.0
      Tags       []string `json:"tags"`       // fully-qualified, pushed or copied
      PrimaryTag string   `json:"primaryTag"` // first of Tags, for convenience
      Platforms  []string `json:"platforms"`  // populated in Phase 4
      Registry   string   `json:"registry"`
      DurationMs int64    `json:"durationMs"`
  }
  ```

- [x] Add `func (b BuildLog) Result() BuildResult` to map the internal log onto it.
- [x] Have `BuildDockerImage` return `(BuildLog, error)` as it does today — `cmd` calls
      `.Result()`. Keeps the test surface untouched.

### 2.2 Emitters

- [x] Add `cli/utils/write_build_output.go` with
      `WriteBuildOutput(result BuildResult, format string, path string) error`.
- [x] `format=json` → marshal indented to **stdout** (or to `path` if given).
- [x] `format=text` (default) → today's behaviour, nothing extra written.
- [x] Add `cli/utils/write_github_output.go` with `WriteGitHubOutput(result BuildResult) error`:
  - [x] No-op when `$GITHUB_OUTPUT` is unset.
  - [x] Append `hash=`, `cache-hit=`, `image=`, `version=`, `primary-tag=`,
        `tags=` (comma-separated), `platforms=` (comma-separated).
  - [x] Open with `os.OpenFile(..., os.O_APPEND|os.O_WRONLY, 0644)` — never truncate,
        other steps write to the same file.
  - [x] Use the heredoc form (`key<<EOF`) for any value that could contain a newline.

### 2.3 Wiring

- [x] Add `OutputFormat string` and `OutputFile string` to `BuildDockerImageParams`.
- [x] Add `--output-format` (`text`|`json`, default `text`) and `--output-file` flags to
      `cli/cmd/build.go`; validate `--output-format` is one of the two.
- [x] Call `WriteBuildOutput` and `WriteGitHubOutput` from `cmd/build.go` after
      `BuildDockerImage` returns successfully.
- [x] On failure, still emit the partial result when `--output-format=json` so callers
      can see the hash that was computed before the error.
- [x] Add `cli/utils/write_github_output_test.go` — point `$GITHUB_OUTPUT` at a temp file
      and assert the exact key/value lines. Pure unit test, no registry.
- [ ] Assert in an existing e2e test that `--output-format=json` produces parseable JSON
      on stdout with nothing else mixed in (depends on Phase 0.1). (deferred: needs a real registry)
- [x] Document both flags and the full `$GITHUB_OUTPUT` key list in the README, with a
      worked GitHub Actions example that consumes `cache-hit` and `primary-tag`.
- [ ] Open a follow-up issue on `kerren/setup-dockem` to surface these as action outputs. (deferred: external repo)

---

## Phase 3 — Respect `.dockerignore`

**The problem.** `cli/utils/build_docker_image.go:33` hashes with
`dirhash.HashDir(params.Directory, "", dirhash.Hash1)`. `dirhash.DirFiles` is a bare
`filepath.Walk` that excludes nothing. Separately,
`cli/utils/tar_build_context.go:63` calls `archive.TarWithOptions(absDirectoryPath,
&archive.TarOptions{})` with no `ExcludePatterns`.

So `.dockerignore` is honoured in neither the hash nor the build context. Consequences:

- `node_modules`, `dist/`, `.env`, coverage output and editor droppings all feed the
  hash. Anything CI generates before the build step silently changes cache identity.
- With the default `--directory=./` at a repo root, `.git/` is inside the walk. Its
  index and ref logs change on every commit, so the hash never repeats and the cache
  hit rate is effectively zero.
- On a genuine rebuild the entire tree — `node_modules` and all — is streamed to the
  daemon.

`github.com/moby/patternmatcher` is already in `go.sum` as an indirect dependency, so
this needs no new module: `ignorefile.ReadAll` parses the file and
`patternmatcher.New(...).MatchesOrParentMatches(...)` applies it.

### 3.1 Read and match the ignore patterns

- [x] Promote `github.com/moby/patternmatcher` to a direct dependency in `cli/go.mod`.
- [x] Add `cli/utils/read_dockerignore.go` with
      `ReadDockerignore(contextDir string) ([]string, error)`:
  - [x] Read `<contextDir>/.dockerignore` via `ignorefile.ReadAll`.
  - [x] Return an empty slice and no error when the file is absent.
  - [x] Honour an override path from `--ignore-file`.
- [x] Append any `--exclude` patterns supplied on the command line to the parsed set.

### 3.2 Hash with exclusions

- [x] Add `cli/utils/hash_directory.go` with
      `HashDirectory(dir string, excludePatterns []string) (string, error)`:
  - [x] List candidates with `dirhash.DirFiles(dir, "")`.
  - [x] Build a `patternmatcher.New(excludePatterns)` and drop any file where
        `MatchesOrParentMatches(file)` is true.
  - [x] **Always retain `Dockerfile` and `.dockerignore`** even if a pattern excludes
        them — this mirrors Docker's own behaviour, and a change to `.dockerignore`
        must invalidate the cache because it changes what is included.
  - [x] Hash the surviving list with `dirhash.Hash1` and an `open` closure that joins
        `dir` (the same one `dirhash.HashDir` uses internally).
  - [x] Skip broken symlinks instead of hard-failing. Today a dangling link anywhere
        under the build directory makes `os.Open` error and aborts the whole hash.
- [x] Replace the `dirhash.HashDir` call in `build_docker_image.go` with `HashDirectory`.
- [x] Apply the same exclusions in `HashWatchDirectories` so watch directories behave
      consistently with the build directory.
- [x] Add `cli/utils/hash_directory_test.go` — pure unit tests over a `t.TempDir()`:
  - [x] A file matching a pattern does not change the hash.
  - [x] A file not matching a pattern does change the hash.
  - [x] A negation pattern (`!keep.txt`) re-includes a file.
  - [x] Editing `.dockerignore` itself changes the hash.
  - [x] A broken symlink does not fail the hash.

### 3.3 Exclude from the build context tar

- [x] In `TarBuildContext`, pass the same patterns as
      `archive.TarOptions{ExcludePatterns: patterns}`.
- [x] Ensure the temporary Dockerfile written for the out-of-context case is never
      excluded — it is created inside the context directory with a `Dockerfile.` prefix
      and a pattern such as `Dockerfile*` would otherwise drop it and break the build.
- [x] Sanity-check that the tar and the hash are computed from the *same* pattern list,
      so the thing that was hashed is the thing that gets built.

### 3.4 Hash version prefix

Since Phase 3 resets cache identity anyway, this is the right moment to make future
resets explicit rather than accidental.

- [x] Define `const hashVersion = "dockem-hash-v2"` in `cli/utils/build_docker_image.go`.
- [x] Seed `overallHash` with it before appending any component hashes.
- [x] Document in `CLAUDE.md` that this constant must be bumped whenever the
      composition of the hash changes.

### 3.5 Flags and rollout

- [x] Add to `BuildDockerImageParams`: `RespectDockerignore bool`, `IgnoreFile string`,
      `Exclude []string`.
- [x] Add flags to `cli/cmd/build.go`:
  - [x] `--respect-dockerignore` / `--no-respect-dockerignore` (**default `true`** in v3).
  - [x] `--ignore-file` — path to an alternative ignore file.
  - [x] `--exclude` (string array) — extra patterns, repeatable.
- [x] Record `excludePatterns` and `respectDockerignore` on `BuildLog` for assertions.
- [x] Add an e2e fixture `testing/e2e/dockerignore-test-image/` with a `.dockerignore`
      that excludes a directory the test then writes a random file into. The hash must
      stay stable across runs and hit the copy path.
- [x] README: document the three flags, and add a "Cache identity" section explaining
      what feeds the hash and what invalidates it.
- [x] Release notes: state plainly that all existing hash tags are invalidated and the
      first v3 build of each image will be a full rebuild.

---

## Phase 4 — BuildKit and multi-platform builds

`cli/utils/build_image.go` uses `dockerClient.ImageBuild` with
`types.ImageBuildOptions` — the classic, deprecated daemon builder. It cannot produce
multi-architecture images, and the classic push path cannot assemble a manifest list.

### 4.1 Approach

Shell out to `docker buildx build`, rather than embedding `moby/buildkit/client`.

Rationale: buildx handles manifest-list assembly, registry push and cache
import/export for free, and it is already present on GitHub-hosted runners
(v0.36.1 locally). Embedding the buildkit client means a large dependency, manual
exporter and session wiring, and a buildkit worker to manage regardless. The cost is
an external binary dependency and hand-rolled credential passing — both manageable,
and detailed below.

Keep the existing classic-builder path as a fallback so users without buildx are not
stranded.

### 4.2 Builder selection

- [x] Add `Platform []string` and `Builder string` to `BuildDockerImageParams`.
- [x] Add `--platform` (string array, repeatable and comma-splittable) and
      `--builder` (`auto`|`buildx`|`docker`, default `auto`) to `cli/cmd/build.go`.
- [x] Add `cli/utils/detect_buildx.go` with `DetectBuildx() (bool, string, error)`
      running `docker buildx version` and returning availability plus the version.
- [x] Resolution logic for `auto`: use buildx when available, otherwise fall back to
      the classic builder.
- [x] **Error, do not silently fall back**, when `--platform` names more than one
      platform and buildx is unavailable — a silent single-arch build published under a
      multi-arch-looking tag is the worst possible outcome.
- [x] Record `builder` and `platforms` on `BuildLog`.

### 4.3 Platforms must feed the hash

Critical correctness point. If `linux/amd64` is built today and
`linux/amd64,linux/arm64` tomorrow, the inputs are unchanged but the required output
is not — without this the tool would copy the single-arch image forward forever.

- [x] Sort the platform list, join it, and append it to `overallHash` in
      `BuildDockerImage`, immediately after the Dockerfile hash.
- [x] Append nothing when `--platform` is unset, so existing users who do not adopt the
      flag see no additional cache invalidation beyond Phase 3's.
- [x] Add a unit test asserting that two different platform lists yield different
      hashes, and that an unset list matches the pre-Phase-4 hash.

### 4.4 The buildx build path

- [x] Add `cli/utils/build_image_buildx.go` with
      `BuildImageBuildx(params, imageHash, targetTags []string, buildLog) error`.
      (Implemented as `BuildImageBuildx(params, imageHash string, targetTags []ResolvedTag, buildLog *BuildLog) error` — `[]ResolvedTag` rather than `[]string` so the per-branch log lines and `outputTags` can be produced on this path exactly as the classic push path does.)
- [x] Assemble the argument list:

  ```
  docker buildx build
    --file <relativeDockerfilePath>
    --platform <comma-separated>          # omitted when unset
    --tag <image:hash>
    --tag <each target tag>               # from ResolveTargetTags (Phase 0.2)
    --push
    --progress plain                      # when not attached to a TTY
    <context directory>
  ```

- [x] Note the structural change: buildx pushes **all** tags in one invocation, so on
      the buildx path `TagAndPushImage` and `TagAndPushNewImages` are not called. This
      is exactly why Phase 0.2 must land first — the tag list is needed up front.
- [x] Stream the subprocess `stdout`/`stderr` straight through to `os.Stderr` so build
      logs keep flowing and stdout stays clean for JSON output.
- [x] Check the exit code and return a descriptive error on failure.
- [x] Pass the build context correctly for the Dockerfile-outside-context case. buildx
      accepts a `--file` outside the context directory natively, so the temp-file
      workaround in `TarBuildContext` is **not needed on this path** — pass the real
      Dockerfile path directly. Keep `TarBuildContext` intact for the classic path.

### 4.5 Credentials for the subprocess

`docker buildx` reads `~/.docker/config.json`; it cannot take a username and password
on the command line. When `--docker-username` / `--docker-password` are supplied
explicitly, the subprocess needs those credentials without mutating the user's real
config.

- [x] Add `cli/utils/temp_docker_config.go` with
      `TempDockerConfig(registry, username, password string) (dir string, cleanup func(), error)`.
- [x] Write a minimal `config.json` containing a base64 `auths` entry for the registry
      into a `t.TempDir()`-style temporary directory with `0600` permissions.
- [x] Set `DOCKER_CONFIG=<dir>` on the subprocess environment only — never export it
      into the parent process.
- [x] Always remove the temp directory via `defer`, including on build failure.
- [x] When no explicit credentials are given, inherit the environment unchanged so an
      existing `docker login` continues to work.
- [x] Never log the password, and confirm it does not reach `BuildLog` output or the
      JSON result from Phase 2.

### 4.6 Verify the copy path handles manifest lists

- [x] Confirm `CopyDockerImage` (regclient `ImageCopy`) copies a multi-platform image
      index, not just a single manifest. It is expected to, but this is the single point
      where a wrong assumption would silently publish a broken multi-arch tag.
      (Verified against regclient v0.6.0 `image.go`: `imageCopyOpt` gets the source
      manifest, and when it is a `manifest.Indexer` it iterates `GetManifestList()`,
      recursively copying every child manifest by digest, then `ManifestPut`s the index
      itself last — the whole image index is preserved.)
- [x] Add an e2e test: build a two-platform image, then re-run so the hash hits, and
      assert the copied tag still resolves for both platforms via
      `client.ManifestGet` on the target. (test added as
      `TestMultiPlatformBuildCopiesImageIndex` and compiles via `go vet`; execution
      deferred: needs a real registry)

### 4.7 Build cache passthrough

Cheap to add once the subprocess exists, and directly addresses the existing README
roadmap item.

- [x] Add `--cache-from` and `--cache-to` (string arrays) passed through verbatim to
      buildx, enabling `type=gha` on GitHub Actions. (`BuildDockerImageParams.CacheFrom`/
      `CacheTo`, repeatable `--cache-from`/`--cache-to` flags on `cli/cmd/build.go`,
      forwarded one `--cache-from <value>`/`--cache-to <value>` pair per element by the
      new pure `assembleBuildxArgs` helper in `cli/utils/build_image_buildx.go`. Warns via
      `LogWarn` in `BuildDockerImage` when either is supplied but the resolved builder is
      not `buildx`.)
- [x] Document that these affect build *speed* only and are deliberately **not** part of
      the hash. (README's "Build Cache" section and "Cache identity" note; `CLAUDE.md`'s
      hash-inputs paragraph and buildx-path bullet. Enforced, not just documented: a new
      pure unit test, `TestCacheFromCacheToExcludedFromImageHash` in
      `cli/utils/build_docker_image_cache_hash_test.go`, fails the build if
      `build_docker_image.go`'s `overallHash` accumulation ever comes to reference
      `CacheFrom`/`CacheTo`.)

### 4.8 Tests and docs

- [ ] e2e: single-platform buildx build, hash miss then hash hit. (test added as
      `TestSinglePlatformBuildxBuildHashMissThenHashHit` in
      `cli/utils/build_docker_image_test.go` and compiles via `go vet`; execution
      deferred: needs a real registry)
- [ ] e2e: `linux/amd64,linux/arm64` build, asserting the pushed tag is an image index.
      (already added in Phase 4.6 as `TestMultiPlatformBuildCopiesImageIndex`; still
      unticked here for the same reason as the other two e2e rows on this rung -
      execution deferred: needs a real registry)
- [ ] e2e: `--builder=docker` still exercises the classic path unchanged. (test added as
      `TestBuilderDockerForcesClassicPathUnchanged` in
      `cli/utils/build_docker_image_test.go`, asserting `buildLog.builder == "docker"`
      and that `buildLog.localTag` is set - the classic-only signal buildx never
      populates - and compiles via `go vet`; execution deferred: needs a real registry)
- [x] Unit: `--platform` with multiple values and no buildx returns a clear error.
      (pure unit test, no registry needed - already present from rung 8 as
      `TestResolveBuilderMultiPlatformWithoutBuildxErrors` in
      `cli/utils/resolve_builder_test.go`; re-ran it during this rung to confirm)
- [x] CI: confirm the GitHub Actions runner has buildx, and add
      `docker/setup-buildx-action` to `.github/workflows/testing.yaml` if not. (added -
      `ubuntu-latest` ships a buildx plugin, but not the `docker-container` driver that
      `--cache-to`/multi-platform pushes need, so `docker/setup-buildx-action@v3` was
      added as an explicit step rather than relying on the runner image's default)
- [x] README: document `--platform`, `--builder`, `--cache-from`, `--cache-to`, and add
      a multi-arch example. (all four present in the flag table, verified against real
      `dockem build --help` output; `--platform`/`--builder` prose and multi-arch example
      already present from rung 7/8; added a "Build Cache" section with a
      `--cache-from=type=gha --cache-to=type=gha,mode=max` example)
- [x] README: tick the two buildx roadmap items.
- [x] `CLAUDE.md`: describe the two build paths and when each is selected. (already done
      in rung 8; extended the buildx-path bullet and the hash-inputs paragraph with the
      cache flags this rung)

---

## Cross-cutting checklist

Run before each release.

- [ ] `task test` passes with real registry credentials (`-count=1` — results must not
      be cached). (deferred: needs a real registry - not available in this environment;
      `go build ./... && go vet ./... && go test -count=1 ./...` with every e2e test name
      skipped was run instead, see the rung 9 commit)
- [x] The pure unit tests pass with **no** credentials set. (re-ran this rung with
      `DOCKER_PASSWORD`/`TEST_IMAGE_NAME` unset; every test that ran passed)
- [x] `task build-binary` succeeds. (re-ran this rung)
- [x] README flag table matches `cli/cmd/build.go` exactly. (diffed the README's Flags
      block against real `dockem build --help` output this rung: all four buildx flags
      and every other flag match verbatim. One pre-existing, unrelated gap noted for
      honesty rather than silently left: `-v, --version`, cobra's auto-generated flag
      from `buildCmd.Version`, is missing from the README table - it predates this rung
      and no flag in `cli/cmd/build.go` defines it, so it is out of this rung's scope)
- [x] `CLAUDE.md` updated for any architectural change. (cache-flag passthrough and the
      classic-builder warning, this rung)
- [ ] Conventional Commits used throughout, with scopes and trailing `#issue` refs
      (`feat(hash):`, `fix(registry):`, `feat(buildx):`). (this rung's commit is a
      Conventional Commit with a `feat(buildx):` scope but, per its dictated exact
      subject, no trailing `#issue` ref; left unticked rather than claiming the full
      history - which is outside a single rung's agent's ability to verify - conforms)
- [ ] Work merged into `develop`, released to `main` via `task release`
      (`task release-major` for v3.0.0). (not done - out of scope for this rung; local
      commits only, per this rung's constraints)

---

## Backlog

Considered and deliberately deferred.

- **`--watch-base-image`** — the `FROM` reference is not part of the hash today, so when
  an upstream base image is repointed to a patched build, dockem sees no change and
  copies the old, unpatched image forward indefinitely. There is currently no input that
  ever triggers a rebuild for an upstream security patch. Fix: parse the `FROM` lines,
  resolve each to a digest using the regclient HEAD that already exists, and fold those
  digests into `overallHash`. Cheap at runtime — HEAD requests only, no pulls, no
  rate-limit pressure. Arguably the highest-value item after this cycle.
- **`--build-arg`** — removed in v2.0.0 and never replaced. Note that build args change
  the resulting image and therefore **must** be folded into the hash, or the cache
  becomes incorrect.
- **Non-JSON version sources** — a plain `VERSION` file, `git describe`, `Cargo.toml`,
  `pyproject.toml`, or a direct `--version-string` flag. Today a JSON file with a
  `version` key is mandatory, which is awkward outside the Node ecosystem.
- **`dockem check` subcommand** — compute the hash, HEAD the registry, print the result
  and exit 0/1 without needing a Docker daemon. Makes dockem usable as a pipeline gate.
  Mostly free once Phases 1 and 2 land.
- **Monorepo config file** — `dockem.yaml` describing many images with per-image watch
  configuration, built in parallel.
- **Homebrew tap** — existing README roadmap item.
