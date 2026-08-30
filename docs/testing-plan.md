# Dockem Testing Plan

This document tracks the work to build out the test suite for `dockem`. It is
organised into phases, each with a concrete checklist. Phases are ordered by value,
not by dependency — T1 and T2 can be worked in either order, though T2 is the
cheapest way to move the coverage number.

**Status legend:** `[ ]` not started · `[~]` in progress · `[x]` done

**Progress:** T0, T2 and T3 are complete (one T0 item needs a real CI run to close).
T1, T4, T5 and T6 remain — each depends on a production refactor listed under
[Refactors this plan depends on](#refactors-this-plan-depends-on), which is why they
were not done alongside the rest.

> **Location note.** This plan lives in `docs/`, not at the repository root. A CI
> guard (Phase T0) asserts that no `plan.md` exists in the root — see that phase for
> the rationale.

---

## Where we are today

Coverage from a run of only the tests a contributor without registry credentials can
actually execute (ie. excluding everything in `build_docker_image_test.go` that
`t.Fatal`s on missing `DOCKER_USERNAME` / `DOCKER_PASSWORD` / `TEST_IMAGE_NAME`):

| Package | Coverage |
|---|---|
| `dockem/utils` | 52.6% |
| `dockem/cmd` | 33.8% |
| `dockem` | 0.0% |

Functions at zero coverage: `ExtractVersion`, `ParseVersionFileJson`,
`HashWatchFiles`, `HashWatchDirectories`, `ReadDockerignore`, `TarBuildContext`,
`DetectBuildx`, `parseBuildxVersion`, `CreateDockerClient`,
`CreateClientWithAuthConfig`, `CreateRegclientClient`, `BuildImage`,
`BuildImageBuildx`, `describePlatforms`, `TagAndPushImage`, `TagAndPushNewImages`,
`CopyExistingImageTag`, `CopyDockerImage`, `LogWarn`, `osOpen`, and all of `cmd/`.

Two existing patterns are the ones to clone rather than invent alternatives to:

- **`check_manifest_head_test.go`** — an `httptest.Server` stub with regclient retries
  disabled. Real HTTP status-code classification, no registry, no credentials, fast.
- **`build_docker_image_cache_hash_test.go`** — a source-reading guard test that fails
  if the `overallHash` span ever mentions a flag it must not. The right tool for
  invariants that no runtime assertion can reach.
- **`temp_docker_config_test.go`** — real filesystem work against a temp `DOCKER_CONFIG`
  tree, asserting both what is written and what is left intact on cleanup. The pattern
  Phase T4's auth tests should reuse.

---

## Phase T0 — Repo hygiene and CI guards

- [x] Add a GitHub Action that asserts **no `plan.md` exists in the root of the
      repository**. Planning documents belong in `docs/`; a stray root `plan.md` is
      scratch state that should never reach `develop` or `main`. The check must be
      case-insensitive (`plan.md`, `PLAN.md`, `Plan.md`) and must fail the build with
      a message pointing at `docs/` as the correct home.
- [x] Wire it into the existing `.github/workflows/testing.yaml`, or a small separate
      `lint`/`hygiene` workflow, running on the same `push` / `pull_request` triggers
      for `main` and `develop`.
- [x] Make it a shell step that is obvious to read and cheap to run, eg. a
      `find . -maxdepth 1 -iname 'plan.md'` that exits non-zero on any hit.
- [ ] Confirm the guard actually fails when a root `plan.md` is present — commit one
      on a scratch branch, watch the check go red, then remove it. A guard that has
      never been seen to fail is not a guard.
- [x] Remove the existing root `PLAN.md` (its v2.6.0 / v3.0.0 phases have landed) so
      the guard passes on `develop` the moment it is added.
- [x] Add a stdout-purity guard in the same spirit: with `--output-format=json`,
      stdout must carry nothing but JSON. A source-level test asserting no
      `fmt.Print*` / direct `os.Stdout` write exists outside `write_build_output.go`
      enforces the `LogInfo` / `LogWarn` / `LogError` convention in `CLAUDE.md`
      directly, rather than by review.
- [x] Add a test asserting every flag registered in `cli/cmd/build.go` appears in
      `README.md`, mechanically enforcing the documented "update the README alongside
      any flag change" convention.

---

## Phase T1 — The invariants that are documented but unenforced

These are the highest-value gaps in the suite. Each is a rule `CLAUDE.md` states
plainly and nothing checks, and each has the same failure mode: a wrong image is
published under a hash tag and then copied forward forever.

### T1.1 Golden image-hash test

- [ ] Extract the hashing span of `BuildDockerImage` into
      `computeImageHash(params) (string, error)` so the hash can be computed without
      any registry work. The span is already delimited by comments (from the
      `overallHash := hashVersion` seed to `imageHash := HashString(overallHash)`).
- [ ] Confirm the extraction keeps the existing source-reading guards
      (`TestCacheFromCacheToExcludedFromImageHash`, `TestSecretsExcludedFromImageHash`)
      working — they read that span as text, so update their file/function target in
      the same change.
- [ ] Build a fixed fixture tree in a temp dir and assert the resulting hash equals a
      hardcoded golden constant.
- [ ] Cover the golden across the axes that feed cache identity: watch files, watch
      directories, build directory, Dockerfile, and the sorted `--platform` list.
- [ ] Document in the test itself that the correct response to a failure is to **bump
      `hashVersion` and update the golden in the same diff** — which is exactly the
      deliberate, visible line the convention asks for.

### T1.2 Hash ↔ tar parity

- [ ] Assert the file set `HashDirectory` keeps is exactly the entry set
      `TarBuildContext` writes into the tar, for a context carrying a `.dockerignore`.
      "If the hash and the tar ever derived from different pattern lists, dockem would
      build something other than what it hashed" is the worst failure this tool can
      have and nothing tests it today.
- [ ] Note this needs no Docker daemon: `TarBuildContext`'s `dockerClient
      *client.Client` parameter is never referenced in the function body, so it can be
      called with `nil`.
- [ ] Delete that dead `dockerClient` parameter and update the single caller in
      `build_image.go`.
- [ ] Cover `--exclude` as well as the `.dockerignore` file: a pattern excluded from
      the hash must not appear in the tar.

### T1.3 Copy path ≡ push path

- [ ] Prove `CopyExistingImageTag` and `TagAndPushNewImages` resolve the same tags in
      the same order. `ResolveTargetTags` is well tested, but nothing proves both
      callers actually use it rather than reimplementing the rule.
- [ ] Prefer a runtime assertion over a source guard here — drive both against fakes
      (see Phase T4) and compare `buildLog.outputTags` directly.
- [ ] Fall back to a source-level guard (no tag string assembled outside
      `resolve_target_tags.go`) if the interface seams land later.
- [ ] Include `assembleBuildxArgs` in the comparison: the buildx path resolves the
      full tag list up front and populates `outputTags` itself, so it is a third
      implementation of the same rule that must agree with the other two.

### T1.4 Builder choice must not change cache identity

- [ ] E2E: build with `--builder=docker`, then re-run identical inputs with
      `--builder=buildx` and assert a hash **hit**. The builder is not an input to the
      hash and must never become one; this is the single test that ties the two build
      paths together.

---

## Phase T2 — Pure unit tests (no Docker, no registry)

Cheapest work in the plan and the biggest single jump in the coverage number. None of
these need a refactor.

### T2.1 Version handling

- [x] `ParseVersionFileJson`: valid JSON, malformed JSON, non-string `version` value.
- [x] `ExtractVersion`: valid file, missing file, unreadable file.
- [x] Decide and then pin the two edges that currently produce tags silently:
      `{}` yields the version string `"v"`, and `{"version": "v1.0.0"}` yields
      `"vv1.0.0"`. Either is a plausible tag name, so neither fails loudly today.

### T2.2 Hashing helpers

- [x] `HashWatchFiles`: empty list returns `""` — the same "contributes nothing"
      contract `hashPlatforms` has, and equally load-bearing for the cache identity of
      users who never adopt the flag.
- [x] `HashWatchFiles`: order invariance, missing file errors, content change changes
      the hash.
- [x] `HashWatchDirectories`: empty list returns `""`; sort invariance; multiple
      directories concatenate; missing directory errors.
- [x] `HashWatchDirectories`: `excludePatterns` are applied consistently with
      `HashDirectory`.
- [x] `HashWatchDirectories` calls `sort.Strings` on the caller's slice, mutating
      `params.WatchDirectory` in place. Either pin that as intended or fix it and test
      that the caller's slice is left untouched.
- [x] `HashString`: a known SHA256 vector, determinism across calls, empty input.

### T2.3 `ReadDockerignore`

- [x] A missing ignore file is not an error and contributes no patterns.
- [x] Comments and blank lines are stripped (via `ignorefile.ReadAll`).
- [x] `--ignore-file` overrides `<directory>/.dockerignore`.
- [x] An unreadable file (mode `000`) surfaces an error.
- [x] `--exclude` patterns land **after** the file's patterns, so a
      `--exclude '!keep-me'` can re-include something the file excluded. That ordering
      is a real behavioural contract and is currently only implied by the code.

### T2.4 Buildx detection

- [x] `parseBuildxVersion`: normal `github.com/docker/buildx v0.36.1 <sha>` output; no
      version-looking token (returns the trimmed output); empty output; a suffixed
      version such as `v0.36.1-desktop.1`.
- [x] `DetectBuildx` via `t.Setenv("PATH", tmpdir)` and a fake `docker` shim: exits 0
      with known output, exits non-zero, and is absent from `PATH` entirely.
- [x] Pin the contract that every failure mode returns `(false, "", nil)` and never an
      error — `ResolveBuilder`, not `DetectBuildx`, decides when that is fatal.

### T2.5 Small helpers

- [x] `GenerateDockerImageName`: empty registry omits the host prefix; a set registry
      includes it; an image name that already contains a slash.
- [x] `RemoveEmptyStringsFromArray`: order preserved; returns `nil` rather than an
      empty slice for all-empty input; whitespace-only strings are **not** removed, so
      `--tag " "` currently survives into a tag name.

---

## Phase T3 — `BuildImageBuildx` subprocess plumbing

`assembleBuildxArgs` is well covered; the function around it is at 0%, and that is
where every rule in the "Subprocess credentials (buildx)" section of `CLAUDE.md`
actually lives. A single test with a fake `docker` on `PATH` — a shell script that
dumps its argv, environment and cwd to a file — covers nearly all of it.

- [x] argv matches what `assembleBuildxArgs` produced.
- [x] `DOCKER_CONFIG` reaches the child pointing at the temp config dir.
- [x] `os.Getenv("DOCKER_CONFIG")` in the parent test process is **unchanged** — it
      must be set on `cmd.Env` only, never on dockem's own environment.
- [x] An arbitrary `FOO=bar` from the parent environment reaches the child. Narrowing
      `cmd.Env` to an allowlist would silently break `--secret id=x,env=VAR`, so this
      guards a real regression.
- [x] The child's cwd equals dockem's cwd (`cmd.Dir` is never set), so a relative
      `src=` path in a `--secret` resolves against dockem's cwd as documented.
- [x] With no credentials, `cmd.Env` is nil and the environment passes through
      untouched, so an existing `docker login` keeps working.
- [x] Any pre-existing `DOCKER_CONFIG` in the parent environment is stripped rather
      than duplicated when dockem sets its own.
- [x] The temp config dir is removed after return — including on a non-zero exit from
      the subprocess.
- [x] A non-zero exit surfaces as an error from `BuildImageBuildx`.
- [x] Subprocess output goes to stderr, not stdout, keeping stdout clean for
      `--output-format=json`.
- [x] The password never appears in argv, in `BuildLog`, or in the JSON result.
- [x] `describePlatforms`: unset, single, and multiple platform lists.

---

## Phase T4 — Registry and daemon seams

`CopyExistingImageTag`, `TagAndPushNewImages`, `CopyDockerImage` and `TagAndPushImage`
are reachable only through a real registry today, which is why they sit at 0%.

- [ ] Introduce narrow interfaces — an `imageCopier` with just `ImageCopy`, a pusher
      with just `ImagePush` — so a fake can record which refs were copied or pushed,
      and in what order.
- [ ] Use those fakes to test the copy/push parity invariant (T1.3) properly, rather
      than by reading source.
- [ ] Assert the per-branch log lines and `outputTags` ordering for every combination
      of `--tag` / `--latest` / `--main-version`.
- [ ] `CreateRegclientClient` and `CreateDockerClient` are the recurring bug class per
      `CLAUDE.md` (CHANGELOG #15, #17) and both are at 0%. Test with `TempDockerConfig`
      writing a config and `t.Setenv("DOCKER_CONFIG", dir)`:
  - [ ] empty registry resolves to Docker Hub;
  - [ ] a registry with an explicit host, and one with a port;
  - [ ] credentials supplied via flags;
  - [ ] no credentials, falling back to the existing `docker login` config;
  - [ ] the password never reaches `BuildLog` or `BuildResult`.
- [ ] `--strict-registry` has no test at all. Extend the `check_manifest_head_test.go`
      httptest pattern: point at a stub returning 401 and assert `BuildDockerImage`
      errors with the flag set, and warns with `headCheckSkipped` true without it.
- [ ] This may need a small seam in `CreateRegclientClient` to accept a plain-HTTP host
      for the stub server.
- [ ] Extend `CheckManifestHead` coverage past the four codes covered today (404, 401,
      429, 500) with 403 and a redirect.

---

## Phase T5 — The `cmd` layer

Real logic lives in `cli/cmd/build.go` and none of it is tested.

- [ ] Extract the `Run` body's parameter assembly into
      `buildParamsFromFlags(cmd *cobra.Command) (utils.BuildDockerImageParams, error)`
      so it can be driven by a cobra command with flags set programmatically.
- [ ] `--platform` comma-splitting, repetition, whitespace trimming, and empty-value
      dropping — all four behaviours live in `build.go` and are entirely untested.
- [ ] `--secret` is deliberately **not** comma-split (`id=npmrc,src=/path` is one
      secret). Pin that asymmetry with `--platform` explicitly.
- [ ] `--no-respect-dockerignore` overrides the `true` default, and
      `--respect-dockerignore=false` does too.
- [ ] A "no flag left behind" test: every registered flag reaches its corresponding
      `BuildDockerImageParams` field, which would catch a typo'd assignment.
- [ ] Validation rejection paths: missing `--image-name`, a bad `--output-format`, a
      bad `--builder`, a missing `--directory`, a missing `--dockerfile-path`.
- [ ] The `Assert*` validators call `os.Exit(1)`, so testing rejection needs either a
      subprocess re-exec (`TestMain` plus a `BE_CRASHER` env guard) or a refactor to
      return errors. Prefer returning errors.
- [ ] A golden `--help` test so flag and documentation drift is visible in the diff.

---

## Phase T6 — E2E suite framework

The e2e tests are thorough but structurally fragile: they need Docker Hub credentials,
they assume a tag already exists on a shared remote repository, and they `t.Fatal`
rather than skip when the environment is absent.

- [ ] Run a local `registry:2` as a CI service container and point the tests at
      `localhost:5000`. This makes hash-hit, hash-miss, copy, multi-arch and
      strict-registry all hermetic on every PR, removes the dependency on secrets
      (forks and outside contributors cannot run them today), and fixes the assumption
      that `base-test-image`'s hash already exists on a shared remote repo.
- [ ] Seed the local registry from within the test rather than relying on prior state.
- [ ] Keep a smaller Docker Hub suite as a nightly job for real-registry auth coverage.
- [ ] Replace `t.Fatal` with `t.Skip` when credentials are absent, gated behind
      `DOCKEM_E2E=1` or a build tag, so `task test` is green offline and a new
      `task test-e2e` runs the real thing.
- [ ] Namespace test tags per run (`test-hash-exists-$RUN_ID`). Two concurrent PR
      builds currently race on the same tag names in the same repository.

Remaining e2e coverage gaps:

- [ ] `--tag` + `--latest` + `--main-version` together in one run, asserted on **both**
      the copy and the push path.
- [ ] Plain idempotency: the same build twice, second run a hash hit with identical
      `outputTags`.
- [ ] `--ignore-build-directory` combined with `--watch-file`: a change in the build
      directory must **not** move the hash, a change in the watch file **must**.
- [ ] An explicit `--registry` host, so both branches of `GenerateDockerImageName` are
      exercised end to end.

---

## Refactors this plan depends on

Small, each unlocking a disproportionate amount of coverage. Listed here so they are
visible as prerequisites rather than surprises inside a phase.

- [ ] `computeImageHash(params)` extracted from `BuildDockerImage` (T1.1).
- [ ] `buildParamsFromFlags(cmd)` extracted from `build.go`'s `Run` (T5).
- [ ] Narrow copier/pusher interfaces in place of concrete clients (T4).
- [ ] `Assert*` validators return errors instead of calling `os.Exit(1)` (T5).
- [ ] The unused `dockerClient` parameter dropped from `TarBuildContext` (T1.2).

---

## Findings surfaced while writing this plan

Recorded because they are latent behaviour, not test gaps. Each needs a decision
before a test can pin it.

- `ExtractVersion` returns `"v"` for a version file of `{}`, and `"vv1.0.0"` for
  `{"version": "v1.0.0"}`. Both become tag names without complaint.
- `HashWatchDirectories` sorts the caller's slice in place, mutating
  `params.WatchDirectory`.
- `TarBuildContext` accepts a `dockerClient *client.Client` it never uses.
- `RemoveEmptyStringsFromArray` drops `""` but keeps `" "`, so a whitespace-only
  `--tag` reaches the registry.
