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

## Phase 0 — Groundwork

Two refactors that Phases 2 and 4 both depend on. Doing them first keeps the later
diffs small and reviewable.

### 0.1 Route human-readable logging to stderr

Phase 2 needs stdout to be clean so JSON can be piped. Today every message is a bare
`fmt.Printf` to stdout.

- [ ] Add `cli/utils/log.go` exposing `LogInfo(format string, a ...any)`,
      `LogWarn(...)` and `LogError(...)`, all writing to `os.Stderr`.
      `LogWarn`/`LogError` prepend the existing `WARN: ` / `ERROR: ` prefixes.
- [ ] Replace every `fmt.Printf` / `fmt.Print` in `cli/utils/*.go` with the new helpers.
      Keep the wording of each message identical — only the destination changes.
- [ ] Leave the `panic(err)` in `cli/cmd/build.go` as-is; error propagation is unchanged.
- [ ] Confirm `jsonmessage.DisplayJSONMessagesStream` calls in `build_image.go` and
      `tag_and_push_image.go` already target `os.Stderr` (they do) — no change needed.
- [ ] Update the "Conventions" section of `CLAUDE.md` to reference the helpers instead
      of raw `fmt.Printf`.

### 0.2 Extract the tag-resolution rule into one function

`CopyExistingImageTag` and `TagAndPushNewImages` each implement the same rule, and
`CLAUDE.md` already flags that they must not drift. Phase 4 adds a third consumer
(buildx needs the full tag list *before* the build starts), so the duplication has to go.

The rule, unchanged: each `--tag X` becomes `X-vVERSION`; `--latest` adds `latest`;
`--main-version` adds `vVERSION`; if none of tags/latest/main-version were given, it
falls back to `vVERSION`.

- [ ] Add `cli/utils/resolve_target_tags.go` with
      `ResolveTargetTags(params BuildDockerImageParams, version string) []string`
      returning fully-qualified image names via `GenerateDockerImageName`.
- [ ] Add `cli/utils/resolve_target_tags_test.go` — a pure unit test (no registry
      needed) covering all four branches and their combinations:
      tags only; `--latest` only; `--main-version` only; none; tags + latest + main.
- [ ] Rewrite `CopyExistingImageTag` to iterate `ResolveTargetTags`.
- [ ] Rewrite `TagAndPushNewImages` to iterate `ResolveTargetTags`.
- [ ] Preserve the per-branch log lines (the "WARN: No tags were specified…" message in
      particular) by returning a small struct or by having the resolver record the reason
      for each tag.
- [ ] Verify `buildLog.outputTags` ordering is unchanged so the existing e2e assertions
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

- [ ] Change the signature to
      `CheckManifestHead(tag string, ref ref.Ref, client *regclient.RegClient) (bool, error)`.
- [ ] Classify with `errors.Is`:
  - [ ] `errs.ErrNotFound` → `(false, nil)`. This is the normal "build it" path.
  - [ ] `errs.ErrHTTPUnauthorized` → `(false, err)` and log a loud `ERROR:` telling the
        user to check `--docker-username` / `--docker-password` or their `docker login`.
  - [ ] `errs.ErrHTTPRateLimit` → `(false, err)` with a message about registry limits.
  - [ ] Anything else → `(false, err)` with the raw error.
- [ ] Delete the `strings.Contains(manifestError.Error(), "failed to request manifest head")`
      string-matching heuristic — it is replaced by the typed checks.
- [ ] Add a `StrictRegistry bool` field to `BuildDockerImageParams`.
- [ ] Add the `--strict-registry` flag to `cli/cmd/build.go` (default `false`).
- [ ] In `BuildDockerImage`, on a non-nil error from `CheckManifestHead`:
  - [ ] If `params.StrictRegistry` → return the error and abort.
  - [ ] Otherwise → log `WARN: … the build will continue, but this should be investigated`
        and proceed to build (preserves today's behaviour by default).
- [ ] Record the outcome on `BuildLog`: add `headCheckError error` and
      `headCheckSkipped bool` so tests can assert on the branch taken.
- [ ] Add unit tests in `cli/utils/check_manifest_head_test.go` using a stub registry
      (`httptest.Server` returning 404 / 401 / 429 / 500) to confirm the classification.
      These need no real registry credentials.
- [ ] Document `--strict-registry` in the README flag table.

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

- [ ] Add `cli/utils/build_result.go` with an exported `BuildResult`:

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

- [ ] Add `func (b BuildLog) Result() BuildResult` to map the internal log onto it.
- [ ] Have `BuildDockerImage` return `(BuildLog, error)` as it does today — `cmd` calls
      `.Result()`. Keeps the test surface untouched.

### 2.2 Emitters

- [ ] Add `cli/utils/write_build_output.go` with
      `WriteBuildOutput(result BuildResult, format string, path string) error`.
- [ ] `format=json` → marshal indented to **stdout** (or to `path` if given).
- [ ] `format=text` (default) → today's behaviour, nothing extra written.
- [ ] Add `cli/utils/write_github_output.go` with `WriteGitHubOutput(result BuildResult) error`:
  - [ ] No-op when `$GITHUB_OUTPUT` is unset.
  - [ ] Append `hash=`, `cache-hit=`, `image=`, `version=`, `primary-tag=`,
        `tags=` (comma-separated), `platforms=` (comma-separated).
  - [ ] Open with `os.OpenFile(..., os.O_APPEND|os.O_WRONLY, 0644)` — never truncate,
        other steps write to the same file.
  - [ ] Use the heredoc form (`key<<EOF`) for any value that could contain a newline.

### 2.3 Wiring

- [ ] Add `OutputFormat string` and `OutputFile string` to `BuildDockerImageParams`.
- [ ] Add `--output-format` (`text`|`json`, default `text`) and `--output-file` flags to
      `cli/cmd/build.go`; validate `--output-format` is one of the two.
- [ ] Call `WriteBuildOutput` and `WriteGitHubOutput` from `cmd/build.go` after
      `BuildDockerImage` returns successfully.
- [ ] On failure, still emit the partial result when `--output-format=json` so callers
      can see the hash that was computed before the error.
- [ ] Add `cli/utils/write_github_output_test.go` — point `$GITHUB_OUTPUT` at a temp file
      and assert the exact key/value lines. Pure unit test, no registry.
- [ ] Assert in an existing e2e test that `--output-format=json` produces parseable JSON
      on stdout with nothing else mixed in (depends on Phase 0.1).
- [ ] Document both flags and the full `$GITHUB_OUTPUT` key list in the README, with a
      worked GitHub Actions example that consumes `cache-hit` and `primary-tag`.
- [ ] Open a follow-up issue on `kerren/setup-dockem` to surface these as action outputs.

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

- [ ] Promote `github.com/moby/patternmatcher` to a direct dependency in `cli/go.mod`.
- [ ] Add `cli/utils/read_dockerignore.go` with
      `ReadDockerignore(contextDir string) ([]string, error)`:
  - [ ] Read `<contextDir>/.dockerignore` via `ignorefile.ReadAll`.
  - [ ] Return an empty slice and no error when the file is absent.
  - [ ] Honour an override path from `--ignore-file`.
- [ ] Append any `--exclude` patterns supplied on the command line to the parsed set.

### 3.2 Hash with exclusions

- [ ] Add `cli/utils/hash_directory.go` with
      `HashDirectory(dir string, excludePatterns []string) (string, error)`:
  - [ ] List candidates with `dirhash.DirFiles(dir, "")`.
  - [ ] Build a `patternmatcher.New(excludePatterns)` and drop any file where
        `MatchesOrParentMatches(file)` is true.
  - [ ] **Always retain `Dockerfile` and `.dockerignore`** even if a pattern excludes
        them — this mirrors Docker's own behaviour, and a change to `.dockerignore`
        must invalidate the cache because it changes what is included.
  - [ ] Hash the surviving list with `dirhash.Hash1` and an `open` closure that joins
        `dir` (the same one `dirhash.HashDir` uses internally).
  - [ ] Skip broken symlinks instead of hard-failing. Today a dangling link anywhere
        under the build directory makes `os.Open` error and aborts the whole hash.
- [ ] Replace the `dirhash.HashDir` call in `build_docker_image.go` with `HashDirectory`.
- [ ] Apply the same exclusions in `HashWatchDirectories` so watch directories behave
      consistently with the build directory.
- [ ] Add `cli/utils/hash_directory_test.go` — pure unit tests over a `t.TempDir()`:
  - [ ] A file matching a pattern does not change the hash.
  - [ ] A file not matching a pattern does change the hash.
  - [ ] A negation pattern (`!keep.txt`) re-includes a file.
  - [ ] Editing `.dockerignore` itself changes the hash.
  - [ ] A broken symlink does not fail the hash.

### 3.3 Exclude from the build context tar

- [ ] In `TarBuildContext`, pass the same patterns as
      `archive.TarOptions{ExcludePatterns: patterns}`.
- [ ] Ensure the temporary Dockerfile written for the out-of-context case is never
      excluded — it is created inside the context directory with a `Dockerfile.` prefix
      and a pattern such as `Dockerfile*` would otherwise drop it and break the build.
- [ ] Sanity-check that the tar and the hash are computed from the *same* pattern list,
      so the thing that was hashed is the thing that gets built.

### 3.4 Hash version prefix

Since Phase 3 resets cache identity anyway, this is the right moment to make future
resets explicit rather than accidental.

- [ ] Define `const hashVersion = "dockem-hash-v2"` in `cli/utils/build_docker_image.go`.
- [ ] Seed `overallHash` with it before appending any component hashes.
- [ ] Document in `CLAUDE.md` that this constant must be bumped whenever the
      composition of the hash changes.

### 3.5 Flags and rollout

- [ ] Add to `BuildDockerImageParams`: `RespectDockerignore bool`, `IgnoreFile string`,
      `Exclude []string`.
- [ ] Add flags to `cli/cmd/build.go`:
  - [ ] `--respect-dockerignore` / `--no-respect-dockerignore` (**default `true`** in v3).
  - [ ] `--ignore-file` — path to an alternative ignore file.
  - [ ] `--exclude` (string array) — extra patterns, repeatable.
- [ ] Record `excludePatterns` and `respectDockerignore` on `BuildLog` for assertions.
- [ ] Add an e2e fixture `testing/e2e/dockerignore-test-image/` with a `.dockerignore`
      that excludes a directory the test then writes a random file into. The hash must
      stay stable across runs and hit the copy path.
- [ ] README: document the three flags, and add a "Cache identity" section explaining
      what feeds the hash and what invalidates it.
- [ ] Release notes: state plainly that all existing hash tags are invalidated and the
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

- [ ] Add `Platform []string` and `Builder string` to `BuildDockerImageParams`.
- [ ] Add `--platform` (string array, repeatable and comma-splittable) and
      `--builder` (`auto`|`buildx`|`docker`, default `auto`) to `cli/cmd/build.go`.
- [ ] Add `cli/utils/detect_buildx.go` with `DetectBuildx() (bool, string, error)`
      running `docker buildx version` and returning availability plus the version.
- [ ] Resolution logic for `auto`: use buildx when available, otherwise fall back to
      the classic builder.
- [ ] **Error, do not silently fall back**, when `--platform` names more than one
      platform and buildx is unavailable — a silent single-arch build published under a
      multi-arch-looking tag is the worst possible outcome.
- [ ] Record `builder` and `platforms` on `BuildLog`.

### 4.3 Platforms must feed the hash

Critical correctness point. If `linux/amd64` is built today and
`linux/amd64,linux/arm64` tomorrow, the inputs are unchanged but the required output
is not — without this the tool would copy the single-arch image forward forever.

- [ ] Sort the platform list, join it, and append it to `overallHash` in
      `BuildDockerImage`, immediately after the Dockerfile hash.
- [ ] Append nothing when `--platform` is unset, so existing users who do not adopt the
      flag see no additional cache invalidation beyond Phase 3's.
- [ ] Add a unit test asserting that two different platform lists yield different
      hashes, and that an unset list matches the pre-Phase-4 hash.

### 4.4 The buildx build path

- [ ] Add `cli/utils/build_image_buildx.go` with
      `BuildImageBuildx(params, imageHash, targetTags []string, buildLog) error`.
- [ ] Assemble the argument list:

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

- [ ] Note the structural change: buildx pushes **all** tags in one invocation, so on
      the buildx path `TagAndPushImage` and `TagAndPushNewImages` are not called. This
      is exactly why Phase 0.2 must land first — the tag list is needed up front.
- [ ] Stream the subprocess `stdout`/`stderr` straight through to `os.Stderr` so build
      logs keep flowing and stdout stays clean for JSON output.
- [ ] Check the exit code and return a descriptive error on failure.
- [ ] Pass the build context correctly for the Dockerfile-outside-context case. buildx
      accepts a `--file` outside the context directory natively, so the temp-file
      workaround in `TarBuildContext` is **not needed on this path** — pass the real
      Dockerfile path directly. Keep `TarBuildContext` intact for the classic path.

### 4.5 Credentials for the subprocess

`docker buildx` reads `~/.docker/config.json`; it cannot take a username and password
on the command line. When `--docker-username` / `--docker-password` are supplied
explicitly, the subprocess needs those credentials without mutating the user's real
config.

- [ ] Add `cli/utils/temp_docker_config.go` with
      `TempDockerConfig(registry, username, password string) (dir string, cleanup func(), error)`.
- [ ] Write a minimal `config.json` containing a base64 `auths` entry for the registry
      into a `t.TempDir()`-style temporary directory with `0600` permissions.
- [ ] Set `DOCKER_CONFIG=<dir>` on the subprocess environment only — never export it
      into the parent process.
- [ ] Always remove the temp directory via `defer`, including on build failure.
- [ ] When no explicit credentials are given, inherit the environment unchanged so an
      existing `docker login` continues to work.
- [ ] Never log the password, and confirm it does not reach `BuildLog` output or the
      JSON result from Phase 2.

### 4.6 Verify the copy path handles manifest lists

- [ ] Confirm `CopyDockerImage` (regclient `ImageCopy`) copies a multi-platform image
      index, not just a single manifest. It is expected to, but this is the single point
      where a wrong assumption would silently publish a broken multi-arch tag.
- [ ] Add an e2e test: build a two-platform image, then re-run so the hash hits, and
      assert the copied tag still resolves for both platforms via
      `client.ManifestGet` on the target.

### 4.7 Build cache passthrough

Cheap to add once the subprocess exists, and directly addresses the existing README
roadmap item.

- [ ] Add `--cache-from` and `--cache-to` (string arrays) passed through verbatim to
      buildx, enabling `type=gha` on GitHub Actions.
- [ ] Document that these affect build *speed* only and are deliberately **not** part of
      the hash.

### 4.8 Tests and docs

- [ ] e2e: single-platform buildx build, hash miss then hash hit.
- [ ] e2e: `linux/amd64,linux/arm64` build, asserting the pushed tag is an image index.
- [ ] e2e: `--builder=docker` still exercises the classic path unchanged.
- [ ] Unit: `--platform` with multiple values and no buildx returns a clear error.
- [ ] CI: confirm the GitHub Actions runner has buildx, and add
      `docker/setup-buildx-action` to `.github/workflows/testing.yaml` if not.
- [ ] README: document `--platform`, `--builder`, `--cache-from`, `--cache-to`, and add
      a multi-arch example.
- [ ] README: tick the two buildx roadmap items.
- [ ] `CLAUDE.md`: describe the two build paths and when each is selected.

---

## Cross-cutting checklist

Run before each release.

- [ ] `task test` passes with real registry credentials (`-count=1` — results must not
      be cached).
- [ ] The pure unit tests pass with **no** credentials set.
- [ ] `task build-binary` succeeds.
- [ ] README flag table matches `cli/cmd/build.go` exactly.
- [ ] `CLAUDE.md` updated for any architectural change.
- [ ] Conventional Commits used throughout, with scopes and trailing `#issue` refs
      (`feat(hash):`, `fix(registry):`, `feat(buildx):`).
- [ ] Work merged into `develop`, released to `main` via `task release`
      (`task release-major` for v3.0.0).

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
