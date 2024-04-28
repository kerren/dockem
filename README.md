![Dockem](docs/logo.png)

So what the heck is this?? Well, it's a `cli` that helps optimise your CI/CD Docker build processes. This tool uses hashes to calculate whether or not the Docker image should be rebuilt or the tag should just be copied. If it should be copied, it connects to the registry and copies the tag without having to do a docker push... And that makes it super fast! Shoutout to [regclient](https://github.com/regclient/regclient) for the API they provide to allow us to do this.


[![Unit Tests](https://github.com/kerren/dockem/actions/workflows/testing.yaml/badge.svg?branch=main)](https://github.com/kerren/dockem/actions/workflows/testing.yaml)


# The Argument

For my full argument, refer to [The Long Argument](#the-long-argument) below. But in short, you can control what files or directories get hashed and only trigger a rebuild if they change. If there is no change, the `cli` will quickly copy the tag to the new one on the registry and you'll be good to go with your new tag.

# Getting Started

This library has been built in `go` in order for me to be able to build binaries for a bunch of different platforms. The easiest way to use this would be to go to the releases page and download the binary that suits you.

## Quick Install

If you're running an AMD64 Linux system and don't want the hassle of figuring things out, you can use the quick install script buy running the following in terminal.

```shell
wget https://github.com/kerren/dockem/releases/download/v2.0.0/dockem-v2.0.0-linux-amd64
chmod 755 dockem-v2.0.0-linux-amd64
sudo mv ./dockem-v2.0.0-linux-amd64 /usr/local/bin/dockem
```

## Usage

```
Usage:
  dockem build [flags]


Flags:
  -d, --directory string                 (required) The directory that should be used as the context for the Docker build (default "./")
  -b, --docker-build-flags stringArray   Any additional build flags you would like to pass directly into the docker build command
  -p, --docker-password string           The password that should be used to authenticate the docker client. Ignore if you have already logged in.
  -u, --docker-username string           The username that should be used to authenticate the docker client. Ignore if you have already logged in.
  -f, --dockerfile-path string           (required) The path to the Dockerfile that should be used to build the image (default "./Dockerfile")
  -h, --help                             help for build
  -I, --ignore-build-directory           Whether to ignore the build directory in the hashing process, this is useful when you are watching a specific file or directory.
  -i, --image-name string                (required) The name of the image you are building
  -l, --latest                           Whether to push the latest tag with this image
  -m, --main-version                     Whether to push this as the main version of the repository. This is done automatically if you do not specify tags or the latest flag.
  -r, --registry string                  The registry that should be used when pulling/pushing the image, Dockerhub is used by default
  -t, --tag stringArray                  The tag or tags that should be attached to image
  -F, --version-file string              (required) The name of the JSON file that holds the version to be used in the build. This JSON file must have the 'version' key. (default "./package.json")
  -W, --watch-directory stringArray      Watch for changes in a directory or directories
  -w, --watch-file stringArray           Watch for changes on a specific file or files

```

Here on some examples on how you would use this `cli`,

```shell
dockem build --directory=./apps/backend --dockerfile-path=./devops/prod/backend/Dockerfile --image-name=my-repo/backend --tag=stable --main-version

dockem build --directory=./apps/backend --watch-directory=./libs/shared --dockerfile-path=./apps/backend/Dockerfile --image-name=my-repo/backend --tag=dev --latest

dockem build --image-name=my-repo/backend --registry=eu.reg.io --docker-username=uname --docker-password=1234 --tag=alpha --tag=test
```

# Roadmap
There are a few tweaks and features I'd like to implement to improve the overall project.

 - [ ] Add more examples to the documentation on how to use the `cli` effectively
 - [ ] Add documentation to the `utils` functions
 - [ ] Create a Github Action that pulls the `dockem` binary
 - [ ] Create end-to-end tests to ensure the core is working, this allows for faster refactoring and feature development
 - [ ] Add to documentation on how to install for different platforms, like ARM and Apple Silicon
 - [ ] Add a Homebrew tap
 - [ ] Add the ability to enable `buildx` caching for Github Actions. This could make the builds faster in future.
 - [ ] Add the ability to specify the platform(s) you'd like to build for using a `buildx` builder. This would be cool to be able to build ARM images using a standard runner. For now, I recommend deploying a custom ARM runner and building on that (it'll also be a lot faster)

# The Long Argument
So now you may ask, why? What's the point?

I've always found Docker builds to be quite slow (and frustrating), especially when the build doesn't even need to take place because nothing has changed but the action triggers on push or when you open a PR. At this point you may say, "well why don't you enable caching using `buildx` and let that speed it up for you?". And to be fair, that is a valid question. It does make the builds faster because all of the layers are cached, BUT, why even push to the registry if you don't have to?

So that's when I started thinking, why don't we push the hash of whatever we want to trigger a build as a tag to the registry? At that point, if the hash is the same, there is no need to rebuild. If there is a different hash, then we trigger a build and push the new hash in the process.

What I really love about this is that we can choose what "changes" the hash. It doesn't even have to be code in your Docker image. For instance, if you have a "base" image that other images extend off, potentially, you'll only want to trigger a build when the `Dockerfile` changes, not the code that would be copied into the base because maybe that's on different layers further up.





