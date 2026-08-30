# Changelog

All notable changes to this project will be documented in this file. See [commit-and-tag-version](https://github.com/absolute-version/commit-and-tag-version) for commit guidelines.

## [3.1.0](https://github.com/kerren/dockem/compare/v3.0.0...v3.1.0) (2026-08-30)


### Features

* **testing:** Add comprehensive test coverage for utils and cmd packages ([ea4a623](https://github.com/kerren/dockem/commit/ea4a623d5d7d00cff08aef773c182fef8d1881b0)), closes [#32](https://github.com/kerren/dockem/issues/32)

## [3.0.0](https://github.com/kerren/dockem/compare/v2.5.0...v3.0.0) (2026-08-30)


### ⚠ BREAKING CHANGES

* **buildx:** Build and push multi-platform images through docker buildx
* **hash:** Respect .dockerignore by default and version the hash
* **buildx:** docker buildx is now required for multi-platform builds. A
build that names more than one --platform errors when buildx is unavailable
rather than silently producing a single-arch image. --builder=docker preserves
the classic single-platform daemon path unchanged.
* **hash:** every hash tag published by dockem before this change is
now unreachable, because both the new default (build directories are now
hashed with .dockerignore patterns applied) and the new dockem-hash-v2
prefix change what the image hash is computed from. The first build of
each image after upgrading will be a full build-and-push even if nothing
about the image itself has changed; after that first build, caching
behaves as before.

### Features

* **buildx:** Build and push multi-platform images through docker buildx ([dfc1a63](https://github.com/kerren/dockem/commit/dfc1a632924cb1968747d67f7c3d5e90135b69db)), closes [#27](https://github.com/kerren/dockem/issues/27)
* **buildx:** Build and push multi-platform images through docker buildx ([02ca3cd](https://github.com/kerren/dockem/commit/02ca3cded5bc81397a5ff35efd8ad5f6ff154cbe))
* **buildx:** Detect buildx and resolve the builder without changing the build path ([9d57265](https://github.com/kerren/dockem/commit/9d5726573695b7f65ac4260348923718297309c0)), closes [#26](https://github.com/kerren/dockem/issues/26)
* **buildx:** Detect buildx and resolve the builder without changing the build path ([9bb26bf](https://github.com/kerren/dockem/commit/9bb26bf9061e3d2e7b273372d5476209c165bed3))
* **buildx:** Pass build cache import and export through to buildx ([313896b](https://github.com/kerren/dockem/commit/313896b96e42da1d5478562b76737f3bef555b32)), closes [#28](https://github.com/kerren/dockem/issues/28)
* **buildx:** Pass build cache import and export through to buildx ([6013f14](https://github.com/kerren/dockem/commit/6013f14aeb4aded4e19da1a4f8b602f004630589))
* **buildx:** Pass build secrets through to buildx ([ca56a16](https://github.com/kerren/dockem/commit/ca56a162d4c35e4fb1fb8b87e1f1d88a447942e4))
* **docs:** This feature add comments to document all util functions ([957e13d](https://github.com/kerren/dockem/commit/957e13d3a631d2c2ec0afca32d3888cbd923850c)), closes [#18](https://github.com/kerren/dockem/issues/18)
* **hash:** Respect .dockerignore by default and version the hash ([26c4904](https://github.com/kerren/dockem/commit/26c4904a65af02e658a724270b672ab649ccd7f0)), closes [#25](https://github.com/kerren/dockem/issues/25)
* **hash:** Respect .dockerignore by default and version the hash ([010d46d](https://github.com/kerren/dockem/commit/010d46d500b3c154ce4b6dbaa0b47ae224a45203))
* **hash:** Respect .dockerignore in the hash and the build context ([6374837](https://github.com/kerren/dockem/commit/6374837f91315d83bcd2e5ccb2ec1fd158b8a163)), closes [#24](https://github.com/kerren/dockem/issues/24)
* **hash:** Respect .dockerignore in the hash and the build context ([89f47ea](https://github.com/kerren/dockem/commit/89f47eafcd3243fe23bb09d9c7bc2483d7f486d7))
* **output:** Emit a machine-readable build result as JSON and GitHub output ([9c39ca8](https://github.com/kerren/dockem/commit/9c39ca82a87abccb7da96df0f8449f2166af1172)), closes [#23](https://github.com/kerren/dockem/issues/23)
* **output:** Emit a machine-readable build result as JSON and GitHub output ([ad5926f](https://github.com/kerren/dockem/commit/ad5926fe00c86ab932eafacd78ce85b1ef667d6b))
* **refactor:** Extract the tag resolution rule into ResolveTargetTags ([c7b955e](https://github.com/kerren/dockem/commit/c7b955e642b670fe6afbc6963eafff1e7ef5a850)), closes [#21](https://github.com/kerren/dockem/issues/21)
* **refactor:** Extract the tag resolution rule into ResolveTargetTags ([097c002](https://github.com/kerren/dockem/commit/097c002696156284530ad4967fa432ca310d0fd5))
* **refactor:** Route human-readable logging through stderr helpers ([352b2e0](https://github.com/kerren/dockem/commit/352b2e062105fba48d10dd315fe95e582a6c6573)), closes [#20](https://github.com/kerren/dockem/issues/20)
* **refactor:** Route human-readable logging through stderr helpers ([e95277e](https://github.com/kerren/dockem/commit/e95277e05a8cbb013b1856dc1ae931cd575a3deb))
* **utils:** update hash watch files to have an early return ([c75bf34](https://github.com/kerren/dockem/commit/c75bf34164b13c67ffdb6b049346480df2dac75f))


### Bug Fixes

* **buildx:** Carry buildx builder state into the throwaway docker config ([0ee6a16](https://github.com/kerren/dockem/commit/0ee6a16e2e4003339a90ba956069202ade094cc4)), closes [#31](https://github.com/kerren/dockem/issues/31)
* **buildx:** Carry the buildx builder state into the throwaway docker config ([f9a7ba6](https://github.com/kerren/dockem/commit/f9a7ba6d540b0827db0a6aee8f98fb72761d52b1))
* **buildx:** Record the out-of-context Dockerfile on the buildx path ([4496e61](https://github.com/kerren/dockem/commit/4496e615b312177b9bb4ef7bed73b95af4c80b58))
* **logging:** Route the pre-panic separator through LogInfo ([a71b0e4](https://github.com/kerren/dockem/commit/a71b0e453a165294b54591569e71ab1d60f7002c))
* **registry:** Classify manifest head errors instead of assuming not found ([df0a6d1](https://github.com/kerren/dockem/commit/df0a6d1aa6a4064a10fc57321cda7016741d9ab0)), closes [#22](https://github.com/kerren/dockem/issues/22)
* **registry:** Classify manifest head errors instead of assuming not found ([2a664ea](https://github.com/kerren/dockem/commit/2a664ea44b47dd313bdc8d853db1543f4cf44b95))

## [2.5.0](https://github.com/kerren/dockem/compare/v2.4.0...v2.5.0) (2024-05-06)


### Features

* **auth:** Use the regclient code to extract the docker host authentication from ~/.docker/config.json [#17](https://github.com/kerren/dockem/issues/17) ([6d8a237](https://github.com/kerren/dockem/commit/6d8a237f404bac19af9c84c5a6450d8d358a0129))

## [2.4.0](https://github.com/kerren/dockem/compare/v2.3.0...v2.4.0) (2024-05-05)


### Features

* Add the license file ([86c130c](https://github.com/kerren/dockem/commit/86c130c5bbc94dfa293f91f9e2311465b4265d2f))


### Bug Fixes

* **auth:** Ensure that the Docker client uses the auth config file if the password has not been specified [#15](https://github.com/kerren/dockem/issues/15) ([135ffab](https://github.com/kerren/dockem/commit/135ffab0246f7c070ee3a57e42116016778d448c))

## [2.3.0](https://github.com/kerren/dockem/compare/v2.2.0...v2.3.0) (2024-05-01)


### Features

* **tests:** Add more tests around tagging ([fa7d864](https://github.com/kerren/dockem/commit/fa7d8643315f1cb3f2051aa421efb75eb9f69b62)), closes [#14](https://github.com/kerren/dockem/issues/14)

## [2.2.0](https://github.com/kerren/dockem/compare/v2.1.0...v2.2.0) (2024-04-30)


### Features

* **devops:** Add a get script to pull the latest binary [#13](https://github.com/kerren/dockem/issues/13) ([6ac0af8](https://github.com/kerren/dockem/commit/6ac0af842c203f1316ce1e66cbe1daa5bda54076))
* **devops:** Split the build workflow out of the tests [#12](https://github.com/kerren/dockem/issues/12) ([6a44e85](https://github.com/kerren/dockem/commit/6a44e85fb0f72e27decb30e0dd08a8bf074cd05f))
* **refactor:** Use a consistent print function across the code [#11](https://github.com/kerren/dockem/issues/11) ([8a7ab17](https://github.com/kerren/dockem/commit/8a7ab17b2f86610c1e75cd64855c605fdd8c864c))

## [2.1.0](https://github.com/kerren/dockem/compare/v2.0.0...v2.1.0) (2024-04-28)


### Features

* **version:** Add the version to the CLI and build it in using ldflags [#10](https://github.com/kerren/dockem/issues/10) ([769b04b](https://github.com/kerren/dockem/commit/769b04b2b51eb3f28cb4b7d59773ffb194fae266))

## [2.0.0](https://github.com/kerren/dockem/compare/v1.4.0...v2.0.0) (2024-04-28)


### ⚠ BREAKING CHANGES

* **build:** Remove the docker build flags as an option because we don't currently do anything with them #9

### Bug Fixes

* **build:** Remove the docker build flags as an option because we don't currently do anything with them [#9](https://github.com/kerren/dockem/issues/9) ([8b066f4](https://github.com/kerren/dockem/commit/8b066f4194a658948ac305f1459e6803583dcd63))

## [1.4.0](https://github.com/kerren/dockem/compare/v1.3.0...v1.4.0) (2024-04-28)


### Features

* **testing:** Add a test for the standard build where the hash has changed [#7](https://github.com/kerren/dockem/issues/7) ([4895c47](https://github.com/kerren/dockem/commit/4895c4715271c281a42252f6c023bc4524a67b29))
* **testing:** Add a test where the Dockerfile is outside of the build context [#8](https://github.com/kerren/dockem/issues/8) ([7a835ec](https://github.com/kerren/dockem/commit/7a835ec9ef01aa7fa3ca9b399875bb3cb4b87f0e))


### Bug Fixes

* **testing:** Correct the branch name on the test workflow to main ([fc1a60e](https://github.com/kerren/dockem/commit/fc1a60e896a4582140e0a27dab91ad76012790e3))

## [1.3.0](https://github.com/kerren/dockem/compare/v1.2.2...v1.3.0) (2024-04-28)


### Features

* **structure:** Restructured the folders in the repository to make the code more maintainable [#5](https://github.com/kerren/dockem/issues/5) ([23c5e37](https://github.com/kerren/dockem/commit/23c5e377ac0926f166ce689961661e19774946f6))
* **testing:** Add an end-to-end test for the standard build process where the hash exists [#6](https://github.com/kerren/dockem/issues/6) ([46f96a3](https://github.com/kerren/dockem/commit/46f96a347dfd7504a7d49b1ea1cb787df7122c8c))

### [1.2.2](https://github.com/kerren/dockem/compare/v1.2.1...v1.2.2) (2024-04-28)


### Bug Fixes

* **auth:** Use a registry default if one is not specified [#4](https://github.com/kerren/dockem/issues/4) ([db4a04c](https://github.com/kerren/dockem/commit/db4a04c5e5924a52f696e574fd6250fb1fa75a04))

### [1.2.1](https://github.com/kerren/dockem/compare/v1.2.0...v1.2.1) (2024-04-28)


### Bug Fixes

* **build:** Add the ability to specify a Dockerfile with a path outside of the build context [#2](https://github.com/kerren/dockem/issues/2) ([e88b551](https://github.com/kerren/dockem/commit/e88b5512f4e7a2c431a5e5274f292ace21f29fc9))

## [1.2.0](https://github.com/kerren/dockem/compare/v1.1.1...v1.2.0) (2024-04-28)


### Features

* **build:** Add the ability to calculate the relative paths between the Dockerfile and the build path. ([e56dbc5](https://github.com/kerren/dockem/commit/e56dbc50f1e7ab3450b977df8224dc857923cf39))

### [1.1.1](https://github.com/kerren/dockem/compare/v1.1.0...v1.1.1) (2024-04-27)

## [1.1.0](https://github.com/kerren/dockem/compare/v1.0.0...v1.1.0) (2024-04-27)


### Features

* **devops:** Add a build script to build the binaries for different platforms. ([bbdf4c0](https://github.com/kerren/dockem/commit/bbdf4c0529d8612901610fae4a7131f79f94591c))

## [1.0.0](https://github.com/kerren/dockem/compare/v0.0.1...v1.0.0) (2024-04-27)


### Features

* **devops:** Add the major release script ([79d824a](https://github.com/kerren/dockem/commit/79d824a2a38d86ef196d459785fd1ec21708e622))

### 0.0.1 (2024-04-27)


### Features

* **devops:** Add the initial release scripts ([f042b5f](https://github.com/kerren/dockem/commit/f042b5f23b592aa0b315bd581911f4c9b78e90e5))
