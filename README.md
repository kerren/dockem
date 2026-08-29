![Dockem](docs/logo.png)

So what the heck is this?? Well, it's a `cli` that helps optimise your CI/CD Docker build processes. This tool uses hashes to calculate whether or not the Docker image should be rebuilt or the tag should just be copied. If it should be copied, it connects to the registry and copies the tag without having to do a docker push... And that makes it super fast! Shoutout to [regclient](https://github.com/regclient/regclient) for the API they provide to allow us to do this.


[![Unit Tests](https://github.com/kerren/dockem/actions/workflows/testing.yaml/badge.svg?branch=main)](https://github.com/kerren/dockem/actions/workflows/testing.yaml)


# The Argument

For my full argument, refer to [The Long Argument](#the-long-argument) below. But in short, you can control what files or directories get hashed and only trigger a rebuild if they change. If there is no change, the `cli` will quickly copy the tag to the new one on the registry and you'll be good to go with your new tag.

# Getting Started

This library has been built in `go` in order for me to be able to build binaries for a bunch of different platforms. The easiest way to use this would be to go to the [Releases Page](https://github.com/kerren/dockem/releases) and download the binary that suits you.

## Quick Install

I've created a quick install script that will download the latest version of the binary for you. This is still a work in progress and I need to test this on Mac and Windows. If you're running a Linux system, this should work without a hassle. For the other systems, I suggest you check out the [Releases Page](https://github.com/kerren/dockem/releases) and download the binary from there.

```shell
curl -s https://raw.githubusercontent.com/kerren/dockem/main/scripts/get_dockem.sh | bash
```

Note, the above script **requires sudo** to move the binary to `/usr/local/bin`. If you don't want to use sudo, you can download the binary to the current directory using,

```shell
curl -s https://raw.githubusercontent.com/kerren/dockem/main/scripts/get_dockem_local.sh | bash
```

### Installing a Specific Version
If you're running an AMD64 Linux system and don't want the hassle of figuring things out, you can use the quick install script buy running the following in terminal.

```shell
wget https://github.com/kerren/dockem/releases/download/v2.5.0/dockem-v2.5.0-linux-amd64
chmod 755 dockem-v2.5.0-linux-amd64
sudo mv ./dockem-v2.5.0-linux-amd64 /usr/local/bin/dockem
```

If you're running an ARM64 Linux system, you can run the following,
```shell
wget https://github.com/kerren/dockem/releases/download/v2.5.0/dockem-v2.5.0-linux-arm64
chmod 755 dockem-v2.5.0-linux-arm64
sudo mv ./dockem-v2.5.0-linux-arm64 /usr/local/bin/dockem
```


## Usage

```
Usage:
  dockem build [flags]


Flags:
  -d, --directory string              (required) The directory that should be used as the context for the Docker build (default "./")
  -p, --docker-password string        The password that should be used to authenticate the docker client. Ignore if you have already logged in.
  -u, --docker-username string        The username that should be used to authenticate the docker client. Ignore if you have already logged in.
  -f, --dockerfile-path string        (required) The path to the Dockerfile that should be used to build the image (default "./Dockerfile")
      --exclude stringArray           Extra .dockerignore-style pattern(s) to exclude from both the input hash and the build context. Repeatable. Always applied, even without --respect-dockerignore.
  -h, --help                          help for build
  -I, --ignore-build-directory        Whether to ignore the build directory in the hashing process, this is useful when you are watching a specific file or directory.
      --ignore-file string            The path to an alternative ignore file to use instead of <directory>/.dockerignore. Only consulted when --respect-dockerignore is set.
  -i, --image-name string             (required) The name of the image you are building
  -l, --latest                        Whether to push the latest tag with this image
  -m, --main-version                  Whether to push this as the main version of the repository. This is done automatically if you do not specify tags or the latest flag.
      --no-respect-dockerignore       Explicitly do NOT respect the .dockerignore file, opting back into the pre-v3 behaviour of hashing and sending everything in the build directory.
      --output-file string            The path of a file to write the --output-format=json build result to, instead of stdout. Ignored when --output-format is not 'json'.
      --output-format string          The format to print the build result in, either 'text' (the default, today's human-readable logging) or 'json' (an indented JSON BuildResult on stdout, or --output-file if set). $GITHUB_OUTPUT is always written to when set, regardless of this flag. (default "text")
  -r, --registry string               The registry that should be used when pulling/pushing the image, Dockerhub is used by default
      --respect-dockerignore          Respect the .dockerignore file in the build directory (and any --ignore-file) when hashing the inputs and building the context, excluding matching files from both. Defaults to true as of v3; this reset every published hash tag when it flipped from v2's default of false. (default true)
  -s, --strict-registry               Whether to abort the build if the registry cannot be reliably checked for the image hash (eg. an authentication failure or a rate limit), instead of logging a warning and continuing with the build.
  -t, --tag stringArray               The tag or tags that should be attached to image
  -F, --version-file string           (required) The name of the JSON file that holds the version to be used in the build. This JSON file must have the 'version' key. (default "./package.json")
  -W, --watch-directory stringArray   Watch for changes in a directory or directories
  -w, --watch-file stringArray        Watch for changes on a specific file or files

```

Here on some examples on how you would use this `cli`,

```shell
dockem build --directory=./apps/backend --dockerfile-path=./devops/prod/backend/Dockerfile --image-name=my-repo/backend --tag=stable --main-version

dockem build --directory=./apps/backend --watch-directory=./libs/shared --dockerfile-path=./apps/backend/Dockerfile --image-name=my-repo/backend --tag=dev --latest

dockem build --image-name=my-repo/backend --registry=eu.reg.io --docker-username=uname --docker-password=1234 --tag=alpha --tag=test

dockem build --image-name=my-repo/backend --tag=stable --output-format=json | jq -r '.primaryTag'
```

## Usage in Actions

I've also created a Github action for this, check out [kerren/setup-dockem](https://github.com/kerren/setup-dockem) to see details. In essence, you'll just need to add the following to your pipeline,

```yaml
    - name: Setup Dockem
      uses: kerren/setup-dockem@v2

    - name: Run Dockem
      run: dockem build --directory=./apps/backend --dockerfile-path=./devops/prod/backend/Dockerfile --image-name=my-repo/backend --tag=stable --main-version
```

## Concepts

In this section, I'll run through the different conecpts to fully explain the `cli` and how it can be used.

### The Version File
The version file is a `JSON` file that holds a `"version"` key. The version inside the key could be anything, however, it's most likely generated using semantic versioning. When a build is run, this version is extracted from the key and added to the tag.

*NOTE*: The version should not start with a `v` as this is added automatically.

An example of the version file is as follows,

```json
{
    "version": "1.0.0"
}
```

### Ignore Build Directory
In most cases (I think), you'd want to trigger a build when the build directory hash has changed. However, there are times that you may not want to do that and instead you would like to watch the hash of other directories or files.

In this case, you can use the `--ignore-build-directory` flag to ignore the build directory in the hashing process.

An example of where this may be useful is if you build base images that other Docker images use in the `FROM` statement. In this case, you may only want to trigger a build when the `Dockerfile` changes and not the code that is copied into the base image.


### Respecting `.dockerignore`
As of `v3`, `--respect-dockerignore` defaults to **`true`**: `dockem` reads the `.dockerignore` at the root of the build directory and applies its patterns to **both** the input hash and the build context. Because the same pattern list drives both, the files that decide whether a build is skipped are exactly the files that get built. Two files are always kept regardless of the patterns: the `Dockerfile` (Docker never lets you ignore it) and `.dockerignore` itself (editing your ignore rules must invalidate the cache).

Before `v3`, every file under the build directory fed the hash and was streamed to the daemon &mdash; including anything CI generates before the build (`node_modules`, `dist/`, coverage output, `.git/`, and so on), which could quietly change the hash on every run. If you need that behaviour back, pass `--no-respect-dockerignore`.

- `--respect-dockerignore` / `--no-respect-dockerignore` &mdash; turn `.dockerignore` handling on or off. **Default `true` as of `v3`** (it was opt-in, default `false`, in `v2`). Pass `--no-respect-dockerignore` to opt back into the pre-`v3` behaviour of hashing and sending everything.
- `--ignore-file` &mdash; use an alternative ignore file instead of `<directory>/.dockerignore`. Only consulted when `--respect-dockerignore` is set.
- `--exclude` &mdash; add extra `.dockerignore`-style patterns on the command line, repeatable. These are always applied, even without `--respect-dockerignore`.

See [Cache identity](#cache-identity) below for what flipping this default means for hash tags you published before `v3`.


### Cache identity
The registry tag that decides whether a build is skipped or performed is a hash of everything that feeds it &mdash; the **image hash**. It is the concatenation of, in this fixed order:

1. A hash-format prefix, `dockem-hash-v2` today, identifying the *shape* of everything below it (see below).
2. Every `--watch-file`, in the order given.
3. Every `--watch-directory`, sorted, each one minus anything matched by `.dockerignore` / `--ignore-file` / `--exclude` when `--respect-dockerignore` is set.
4. The build directory (unless `--ignore-build-directory`), likewise minus any `.dockerignore` / `--exclude` matches.
5. The `Dockerfile`.

That concatenation is then hashed once to produce the image hash used as the registry tag. Change any of the inputs above &mdash; edit a watched file, add a non-ignored file to the build directory, change the Dockerfile, or change what `.dockerignore` excludes &mdash; and the hash changes, so the next build is a real build-and-push rather than a server-side tag copy. Leave all of them the same and the hash repeats, so the build is skipped and the tag is just copied.

The prefix in step 1 exists so a change to *how* the hash is put together (as opposed to what happens to go into it on a given run) can be signalled explicitly. Every release before `v3` used an implicit, unmarked hash format &mdash; `overallHash` simply started as `""`. `v3` is the first release to mark that format, as `dockem-hash-v2`, at the same time as flipping `--respect-dockerignore` to default `true`. Bumping this prefix in some future release resets the cache for every image on purpose, as a deliberate, visible line in that release's diff, rather than silently as a side effect of some unrelated change.

**Upgrading to `v3` invalidates every hash tag your pipelines have ever published**, because both of the changes above &mdash; respecting `.dockerignore` by default and introducing the `dockem-hash-v2` prefix &mdash; change what the hash is computed from. The first `v3` build of every image will be a full build-and-push, even if nothing about the image itself has changed since the last `v2` build. After that first build, caching behaves exactly as before: unchanged inputs mean a cache hit and a tag copy, changed inputs mean a rebuild.


### Main Version
The `--main-version` flag is used to specify that this build should be the main version of the repository.

So for instance, if you have an image called `example-org/backend` and you use the `--main-version` flag, it would push the following image to the registry,
```
example-org/backend:v1.0.0
```
Assuming the version in the version file is `1.0.0`.


### Latest

The `--latest` flag is used to specify that this build should be the latest version of the repository.

So for instance, if you have an image called `example-org/backend` and you use the `--latest` flag, it would push the following image to the registry,
```
example-org/backend:latest
```


### Watch File / Watch Directory
The hash is generated from the files and/or directories you specify. You can specify as many as you'd like.

When you use the `--watch-file` and/or the `--watch-directory` flags, the build will trigger whenever something in the specified files or directories change.

An example of where this might be useful is if you have a base image that other `Dockerfiles` build from. You may only want to watch the `package-lock.json` file or some other lock file to trigger a build because you don't care about the source but you do care when the base dependencies change.


### Tag

The `--tag` flag can be used to push to a specific tag on the image. At the moment, the version is appended to the tag before it pushes.

So for instance, if you have an image name `example-org/backend` and you use the `--tag=alpha` flag, it would push the following image to the registry,
```
example-org/backend:alpha-v1.0.0
```

Assuming the version in the version file is `1.0.0`.


### Strict Registry

By default, if `dockem` is unable to reliably check the registry for the image hash (for
example, an authentication failure or a rate limit rather than a genuine "this tag
doesn't exist yet"), it logs a warning and carries on as if the hash was not found,
triggering a build. This matches the behaviour `dockem` has always had.

The `--strict-registry` flag changes this: if the registry check fails for any reason
other than the hash genuinely not existing, the build aborts immediately instead of
proceeding. This is useful in CI/CD pipelines where you'd rather fail fast on a
credentials or registry problem than pay for a full build and push that may well fail
again at the push step for the same reason.


### Output Format

By default (`--output-format=text`), `dockem` just logs its human-readable progress
to stderr as it runs, exactly as it always has.

Passing `--output-format=json` additionally prints an indented JSON object describing
the result of the build to stdout once the build finishes - or to the file given by
`--output-file`, instead of stdout, if you'd rather not deal with capturing it from a
subprocess. Nothing else is written to stdout in this mode, so it's always safe to
pipe straight into something like `jq`.

```shell
$ dockem build --image-name=my-repo/backend --tag=stable --output-format=json | jq .
{
  "hash": "1a2b3c4d5e6f...",
  "cacheHit": true,
  "image": "my-repo/backend:1a2b3c4d5e6f...",
  "version": "v1.0.0",
  "tags": [
    "my-repo/backend:stable-v1.0.0"
  ],
  "primaryTag": "my-repo/backend:stable-v1.0.0",
  "platforms": [],
  "registry": "",
  "durationMs": 842
}
```

If the build fails, `dockem` still writes whatever it managed to compute before the
failure (eg. the hash, if it got that far) as JSON before it exits non-zero, so a
pipeline can tell how far the build got even when something went wrong.

### GitHub Actions Output

Whenever the `$GITHUB_OUTPUT` environment variable is set - which GitHub Actions sets
automatically for every step - `dockem build` appends its result to that file as step
outputs on a successful build, regardless of `--output-format`. This is what lets a
later step in the same job reference eg. `steps.<id>.outputs.primary-tag` without
having to scrape it out of the logs.

The following keys are written:

| Key | Description |
|---|---|
| `hash` | The computed image hash used as the cache key. |
| `cache-hit` | `"true"` if the hash already existed on the registry (the build was skipped and the tag copied instead), `"false"` if a build actually happened. |
| `image` | The fully-qualified `image:hash` name that was checked/pushed. |
| `version` | The version extracted from the version file, eg. `v1.0.0`. |
| `primary-tag` | The first tag that was pushed or copied - typically the one you want to deploy. |
| `tags` | Every tag that was pushed or copied, comma-separated. |
| `platforms` | The platform(s) the image was built for, comma-separated. Empty until multi-platform builds are supported. |

Here's a worked example that skips the deploy step entirely on a cache hit, and
otherwise deploys the primary tag,

```yaml
    - name: Setup Dockem
      uses: kerren/setup-dockem@v2

    - name: Run Dockem
      id: dockem
      run: dockem build --image-name=my-repo/backend --tag=stable --main-version

    - name: Deploy
      if: steps.dockem.outputs.cache-hit != 'true'
      run: ./deploy.sh ${{ steps.dockem.outputs.primary-tag }}
```

# Roadmap
There are a few tweaks and features I'd like to implement to improve the overall project.

 - [x] Create a Github Action that pulls the `dockem` binary
 - [x] Add to documentation on how to install for different platforms, like ARM and Apple Silicon
 - [x] Create end-to-end tests to ensure the core is working, this allows for faster refactoring and feature development
 - [x] Add more examples to the documentation on how to use the `cli` effectively
 - [ ] Add documentation to the `utils` functions
 - [ ] Add a Homebrew tap
 - [ ] Add the ability to enable `buildx` caching for Github Actions. This could make the builds faster in future.
 - [ ] Add the ability to specify the platform(s) you'd like to build for using a `buildx` builder. This would be cool to be able to build ARM images using a standard runner. For now, I recommend deploying a custom ARM runner and building on that (it'll also be a lot faster)

# The Long Argument
So now you may ask, why? What's the point?

I've always found Docker builds to be quite slow (and frustrating), especially when the build doesn't even need to take place because nothing has changed but the action triggers on push or when you open a PR. At this point you may say, "well why don't you enable caching using `buildx` and let that speed it up for you?". And to be fair, that is a valid question. It does make the builds faster because all of the layers are cached, BUT, why even push to the registry if you don't have to?

So that's when I started thinking, why don't we push the hash of whatever we want to trigger a build as a tag to the registry? At that point, if the hash is the same, there is no need to rebuild. If there is a different hash, then we trigger a build and push the new hash in the process.

What I really love about this is that we can choose what "changes" the hash. It doesn't even have to be code in your Docker image. For instance, if you have a "base" image that other images extend off, potentially, you'll only want to trigger a build when the `Dockerfile` changes, not the code that would be copied into the base because maybe that's on different layers further up.





