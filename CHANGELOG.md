# Changelog

All notable changes to this project will be documented in this file. See [commit-and-tag-version](https://github.com/absolute-version/commit-and-tag-version) for commit guidelines.

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
