# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`dockem` is a Go CLI that speeds up CI/CD Docker builds by hashing the inputs of an image and using that hash as a registry tag. If the hash tag already exists on the registry, the build is skipped entirely and the existing manifest is copied to the requested tags server-side via [regclient](https://github.com/regclient/regclient) (no pull, no push). Otherwise it builds through the Docker daemon and pushes.

The Go module lives in `cli/` (module name `dockem`), not the repo root.

## Commands

Tasks are driven by [Task](https://taskfile.dev) from the repo root (`taskfile.yml`); each one `cd`s into `cli/` first.

```shell
task install-deps      # cd cli && go get .
task build-binary      # cd cli && go build  -> ./cli/dockem
task test              # cd cli && go test -count=1 ./...
task test-verbose      # same, with -v
task build             # cross-compile release binaries into ./release (run from main; uses `git describe --tags HEAD`)
task release           # entro-version release develop -> main (git flow); `task release-major` for a major bump
```

Run a single test:

```shell
cd cli && go test -count=1 -run TestStandardBuildWhereHashExists ./utils/
```

`-count=1` matters — the tests hit a real registry and must not be cached.

### Tests need a registry

`cli/utils/build_docker_image_test.go` is an end-to-end suite: it builds real images and pushes to a real registry. It `t.Fatal`s unless `DOCKER_USERNAME`, `DOCKER_PASSWORD`, and `TEST_IMAGE_NAME` are set. Locally, `.autoenv.zsh` sources `.env` (gitignored) for these. CI supplies them from repo secrets/vars. The pure-unit tests (`assert_*_test.go`, `file_exists_test.go`, `directory_exists_test.go`) need nothing.

Fixtures in `testing/e2e/`:
- `base-test-image/` — never changes, so its hash should always already exist on the registry (exercises the copy path).
- `base-changing-test-image/` — tests write a random temp file into `build/` so the hash is always new (exercises the build-and-push path).
- `dockerfile-context/` — a Dockerfile living outside its build context.

Version numbers in fixture `version.json` files feed the tag assertions, so changing them changes what the tests expect on the registry.

## Architecture

`cli/main.go` → `cli/cmd/` (cobra) → `cli/utils/` (everything else). The `cmd` layer only parses/validates flags and packs them into `utils.BuildDockerImageParams`; all logic lives in `utils`, one exported function per file, named after the file.

`utils.BuildDockerImage` (`cli/utils/build_docker_image.go`) is the whole pipeline and the best entry point for understanding the tool:

1. **Hash the inputs.** `overallHash` is seeded with the `hashVersion` constant (`cli/utils/build_docker_image.go`, currently `"dockem-hash-v2"`), then concatenated, in fixed order: watch files (`HashWatchFiles`), watch directories (`HashWatchDirectories`, sorted), the build directory (unless `--ignore-build-directory`), the Dockerfile, and finally the sorted, comma-joined `--platform` list (`hashPlatforms`, which appends **nothing** when `--platform` is unset, so single-platform users see no change). Watch directories and the build directory go through `HashDirectory`, which drops anything matched by `.dockerignore`/`--exclude` when `--respect-dockerignore` is set (the default as of `v3`). All the file hashing uses `golang.org/x/mod/sumdb/dirhash`. The concatenated string is then hashed once by `HashString` to produce the image hash. The platform list feeds the hash because the same inputs built for a different set of architectures are a different required output — without it, dockem would copy a single-arch image forward even after the caller started asking for `linux/amd64,linux/arm64`. **Order and content of this concatenation define cache identity — any change to it invalidates every published hash tag. Whenever you change what feeds the hash, or how, bump `hashVersion` in the same change — that makes the cache reset a deliberate, visible line in the diff instead of an accidental side effect of some unrelated fix.** `--cache-from`/`--cache-to` (see below) are the deliberate exception that proves this rule: they are excluded from `overallHash` on purpose, because they only change how fast a build runs, never the image it produces.
2. **Read the version** from the JSON `version` key of `--version-file`; `ExtractVersion` prefixes a `v`.
3. **Check the registry** with a regclient `ManifestHead` on `image:hash` (`CheckManifestHead`). A HEAD, not a pull, to stay under registry rate limits. Any error is treated as "does not exist" and the build proceeds.
4. **Hash exists** → `CopyExistingImageTag` copies the manifest registry-side to each target tag (this copy handles a multi-platform image *index* correctly — regclient's `ImageCopy` recurses over every child manifest before pushing the index, so a cached multi-arch tag stays multi-arch). **Hash missing** → one of the two build paths below runs.

`CopyExistingImageTag` and `TagAndPushNewImages` must stay behaviourally identical — they are the two halves of the same tag-resolution rule, and a change to one without the other silently makes cached builds differ from fresh builds. That rule (`ResolveTargetTags`): each `--tag X` becomes `X-vVERSION`; `--latest` adds `latest`; `--main-version` adds `vVERSION`; and if none of tags/latest/main-version were given, it falls back to `vVERSION`.

### Two build paths

On a hash miss dockem builds one of two ways, chosen by `ResolveBuilder` (`cli/utils/build_docker_image.go` calls it before any hashing so an impossible request fails fast). `--builder` is `auto` (default), `buildx`, or `docker`, and `buildLog.builder` records the resolved choice (`"buildx"` or `"docker"`):

- **`buildx` path** (`BuildImageBuildx`, `cli/utils/build_image_buildx.go`) — shells out to a single `docker buildx build --file <dockerfile> [--platform <list>] [--cache-from <value>]… [--cache-to <value>]… --tag <image:hash> --tag <each target> --push [--progress plain] <context>`. This is the only path that can build multi-platform images, and it pushes **every** tag — the hash tag and all `ResolveTargetTags` tags — in one invocation, so `TagAndPushImage`/`TagAndPushNewImages` are **not** called here. `BuildImageBuildx` therefore resolves the tag list up front, emits the same per-branch log lines and populates `buildLog.outputTags` exactly as the classic push path does. It streams the subprocess stdout+stderr to `os.Stderr` (keeping dockem's stdout clean for `--output-format=json`), and passes the real Dockerfile path as `--file` — buildx accepts a Dockerfile outside the context natively, so it does **not** use `TarBuildContext`'s temp-file workaround. The argument-assembly logic lives in the pure, separately-tested `assembleBuildxArgs` helper in the same file. `--cache-from`/`--cache-to` (repeatable, `BuildDockerImageParams.CacheFrom`/`CacheTo`) are forwarded here **verbatim**, one flag per value — dockem never interprets their contents, so any BuildKit cache backend (`type=gha`, `type=registry`, …) works without dockem knowing about it. They only take effect on this path; when the classic builder is resolved instead, `BuildDockerImage` logs a `LogWarn` that they are being ignored, rather than silently dropping them.

- **`docker` (classic) path** (`BuildImage` → `TagAndPushImage` → `TagAndPushNewImages`) — builds via the Docker daemon to a `local:<hash>` tag, pushes the hash tag, then pushes each resolved target tag one at a time. It is single-arch only; `ResolveBuilder` hard-errors (never silently falls back) if more than one `--platform` is requested without buildx.

**Selection:** `auto` → buildx when `docker buildx version` succeeds, else classic (unless multi-platform was requested, which errors). `buildx` → buildx or error. `docker` → always classic (and must stay behaviourally identical to before buildx existed). When touching either path, keep the two behaviourally aligned on tag resolution and `outputTags`.

### Subprocess credentials (buildx)

`docker buildx` reads `~/.docker/config.json` and cannot take a username/password on the command line. When `--docker-username`/`--docker-password` are given, `TempDockerConfig` writes a throwaway `config.json` (0600, in a 0700 temp dir) and `BuildImageBuildx` points the subprocess at it with `DOCKER_CONFIG=<dir>` on `exec.Cmd.Env` **only** — never dockem's own environment — removing it via `defer cleanup()`. With no explicit credentials the subprocess inherits dockem's environment unchanged, so an existing `docker login` works. The password never reaches the command line, the logs, `BuildLog`, or the JSON result.

### Two clients, two auth paths

The tool talks to registries two different ways and each has its own client constructor:

- `CreateRegclientClient` — for the HEAD check and server-side copy. Builds a `config.Host`; falls back to `regclient.WithDockerCreds()` (i.e. `~/.docker/config.json`) whenever no password was passed.
- `CreateDockerClient` — for build and push. With no explicit credentials it loads hosts via regclient's `config.DockerLoad()` and matches on registry name, so `docker login` works without flags.

An empty `--registry` means Docker Hub, and `GenerateDockerImageName` omits the host prefix in that case. Auth regressions are the recurring bug class here (see CHANGELOG #15, #17) — changes to either constructor should be exercised against both an authenticated and an already-logged-in-only environment.

### Dockerfile outside the build context

This applies to the **classic path only**. `TarBuildContext` handles a `--dockerfile-path` that resolves outside `--directory` (relative path starts with `../`): it copies the Dockerfile to a temp file inside the context, tars the context, then **reads the whole tar into memory** so the temp file can be deleted before the build starts rather than being left behind if the user Ctrl-Cs. Keep that ordering if touching this file. The buildx path sidesteps all of this — it hands buildx the real `--file` path directly.

### BuildLog

`utils.BuildLog` is an unexported-fields struct threaded by pointer through the pipeline purely so the e2e tests can assert on what actually happened (`hashExists`, `outputTags`, `localTag`, `customDockerfile`, …). It is not user-facing output. When adding a branch to the build pipeline, record it here so it can be tested.

## Conventions

- Human-readable output goes through `LogInfo`/`LogWarn`/`LogError` in `cli/utils/log.go` (all writing to `os.Stderr`) rather than raw `fmt.Printf`/`fmt.Print`/`fmt.Println`. Errors are surfaced by calling `LogError` with a sentence explaining what the user should check (`LogWarn`/`LogError` prepend the `WARN: ` / `ERROR: ` prefix themselves), then returning the raw error up to `cmd/build.go`, which panics. Match this style rather than wrapping errors.
- One exported function per file in `utils/`, file named in snake_case after the function.
- Commits follow Conventional Commits — the CHANGELOG is generated by `commit-and-tag-version` via `entro-version`, and scopes like `feat(auth):` / `fix(auth):` with a trailing `#issue` reference are the norm.
- Git flow: work lands on `develop`, releases merge to `main` and tag. `package.json` at the root exists only to hold the version that the release tooling bumps.
- The release binary's `--version` comes from `-ldflags "-X dockem/cmd.Version=$version"`; `cmd.Version` defaults to `dev-build` in local builds.
- The README documents every flag and concept; update it alongside any flag change in `cli/cmd/build.go`.
