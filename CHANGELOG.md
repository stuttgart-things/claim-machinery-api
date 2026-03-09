## [0.15.3](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.15.2...v0.15.3) (2026-03-09)


### Bug Fixes

* write homerun notification logs to stdout instead of stderr ([a6453dc](https://github.com/stuttgart-things/claim-machinery-api/commit/a6453dcab1f366e7548418ccd65ef8bd7d59a213)), closes [#80](https://github.com/stuttgart-things/claim-machinery-api/issues/80)

## [0.15.2](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.15.1...v0.15.2) (2026-03-09)


### Bug Fixes

* suppress health check logging to reduce noise ([31a2816](https://github.com/stuttgart-things/claim-machinery-api/commit/31a2816c6340f3634dcd62cb0b62a2bd1df8bfd8))

## [0.15.1](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.15.0...v0.15.1) (2026-03-09)


### Bug Fixes

* **deps:** update module github.com/charmbracelet/huh to v1 ([0ccc389](https://github.com/stuttgart-things/claim-machinery-api/commit/0ccc38906ae76bc758d083cedf75af9a619f0720))

# [0.15.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.14.0...v0.15.0) (2026-03-09)


### Features

* send homerun2 notification on successful template order ([d5c8cf0](https://github.com/stuttgart-things/claim-machinery-api/commit/d5c8cf0fff5d06fcec87f1b7c9b30e30e82cef23)), closes [#77](https://github.com/stuttgart-things/claim-machinery-api/issues/77)

# [0.14.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.13.0...v0.14.0) (2026-03-09)


### Features

* return errors from KCL render functions and support remote profile hot reload ([cb06959](https://github.com/stuttgart-things/claim-machinery-api/commit/cb069596e6b56a00aa648fca4632da9900311d7b))

# [0.13.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.12.0...v0.13.0) (2026-03-06)


### Features

* add /release skill for triggering GitHub Actions release workflow ([609e8af](https://github.com/stuttgart-things/claim-machinery-api/commit/609e8afd8a94bc8ba5c0a422ca1937477dbb5fd9))

# [0.11.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.10.0...v0.11.0) (2026-03-04)


### Features

* add labul-vsphere profile with all 6 templates ([c95c322](https://github.com/stuttgart-things/claim-machinery-api/commit/c95c322b373cf8f8a981548c612d5f4868dda49e))

# [0.10.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.9.0...v0.10.0) (2026-03-04)


### Features

* support HTTP URL for TEMPLATE_PROFILE_PATH and configurable template profiles ([3069c42](https://github.com/stuttgart-things/claim-machinery-api/commit/3069c42df5fd45e4ce635b38438645deaff28af3))

# [0.9.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.8.0...v0.9.0) (2026-03-03)


### Features

* add trigger-release and trigger-pages tasks for remote workflow dispatch ([fc86931](https://github.com/stuttgart-things/claim-machinery-api/commit/fc86931057e82e800f0f6134cef153cf0208bd0b))

# [0.8.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.7.0...v0.8.0) (2026-03-03)


### Features

* auto-deploy pages on release with changelog and flux deploy reference ([d6ad54b](https://github.com/stuttgart-things/claim-machinery-api/commit/d6ad54bbbb5297ea083f9961838e261ec2c734fd))

# [0.7.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.6.0...v0.7.0) (2026-03-03)


### Features

* add GitHub Pages and GitLab Pages deployment ([042e218](https://github.com/stuttgart-things/claim-machinery-api/commit/042e218d982414a40a6173b1b6384b2782499d80))

# [0.6.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.5.6...v0.6.0) (2026-03-02)


### Features

* add deployment workflow with kustomize overlay example and deploy tasks ([6ffda57](https://github.com/stuttgart-things/claim-machinery-api/commit/6ffda574e9dbbb41c3ea388b81d02a060b2137fa))

## [0.5.6](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.5.5...v0.5.6) (2026-03-02)


### Bug Fixes

* add ScanImage function documentation ([5a3f03c](https://github.com/stuttgart-things/claim-machinery-api/commit/5a3f03cccc9f99b57cb5f8238873bbe1b5d133c0))

## [0.5.5](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.5.4...v0.5.5) (2026-03-02)


### Bug Fixes

* clarify BuildImage function documentation ([83ae7af](https://github.com/stuttgart-things/claim-machinery-api/commit/83ae7af331b1773b005959462822d110b3bf655d))

## [0.5.4](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.5.3...v0.5.4) (2026-03-02)


### Bug Fixes

* improve koBuildWithConfig documentation ([aa72fa1](https://github.com/stuttgart-things/claim-machinery-api/commit/aa72fa14483e0ec366e3110a7acbc4859f2922bb))

## [0.5.3](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.5.2...v0.5.3) (2026-03-02)


### Bug Fixes

* trigger release after build completes via workflow_run ([95684d9](https://github.com/stuttgart-things/claim-machinery-api/commit/95684d9caa267ed68550aa9bc0d034cae30036a5))

## [0.5.2](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.5.1...v0.5.2) (2026-03-02)


### Bug Fixes

* handle workflow_dispatch on main for ghcr.io push ([304e90b](https://github.com/stuttgart-things/claim-machinery-api/commit/304e90bcf002b2b861144ba3b00dbb2f993bfd67))

## [0.5.1](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.5.0...v0.5.1) (2026-03-02)


### Bug Fixes

* use ko --bare flag for clean image naming and add tag parameter ([8a3e925](https://github.com/stuttgart-things/claim-machinery-api/commit/8a3e925c2abb43d6cde622bedee44d7fcb5b7a7a))

# [0.5.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.4.1...v0.5.0) (2026-03-01)


### Bug Fixes

* pass GITHUB_TOKEN to ko for ghcr.io auth and disable SBOM push ([4009113](https://github.com/stuttgart-things/claim-machinery-api/commit/4009113733af60c2ec9d4255713415dab7f4e8fe))
* remove duplicate platform-engineering system declaration ([3aa919c](https://github.com/stuttgart-things/claim-machinery-api/commit/3aa919cdd716813758dccc91d0d45729ccfb0377))
* use server subcommand in Dagger BuildAndTest entrypoint ([7494f50](https://github.com/stuttgart-things/claim-machinery-api/commit/7494f5002fba5185b7fe7aecb05ab872eb4d7ac1))
* UX improvements for render command ([#36](https://github.com/stuttgart-things/claim-machinery-api/issues/36)) ([8ac4e6f](https://github.com/stuttgart-things/claim-machinery-api/commit/8ac4e6f97903aaf93a032c07b90ca24ec8518dda))


### Features

* add Gateway API HTTPRoute as alternative to Ingress ([576c0e1](https://github.com/stuttgart-things/claim-machinery-api/commit/576c0e15f21a18447fa73a3046d970e1a1ea8b50))
* consolidated Dagger-based release pipeline ([#67](https://github.com/stuttgart-things/claim-machinery-api/issues/67)) ([9ea1f4a](https://github.com/stuttgart-things/claim-machinery-api/commit/9ea1f4aa59cf7d308d859da7756167e4b7330e72)), closes [#52](https://github.com/stuttgart-things/claim-machinery-api/issues/52)

## [0.4.1](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.4.0...v0.4.1) (2026-02-04)


### Bug Fixes

* fix/release-config ([1c7a347](https://github.com/stuttgart-things/claim-machinery-api/commit/1c7a347ada17d88e4b1d71ad59a58c107158be59))

# [0.4.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.3.0...v0.4.0) (2026-02-04)


### Features

* add multiselect support for enum parameters ([#40](https://github.com/stuttgart-things/claim-machinery-api/issues/40)) ([23337f5](https://github.com/stuttgart-things/claim-machinery-api/commit/23337f517feac7bb9c425f2cd1e43958640d17a1))
* feat/add-api-test-funv ([2f871b5](https://github.com/stuttgart-things/claim-machinery-api/commit/2f871b5118570ff78b96541de3e4830035d3b566))
* feat/add-claim-field-doc ([d34bfc9](https://github.com/stuttgart-things/claim-machinery-api/commit/d34bfc9733a5a5632b174ad4a91777c5e66782c7))
* feat/add-pr-inlcude-task ([eba8970](https://github.com/stuttgart-things/claim-machinery-api/commit/eba8970c069b220b624ab2eef9b28a25c9cf888d))
* feat/disable-read-default-stddir ([f0ca1c3](https://github.com/stuttgart-things/claim-machinery-api/commit/f0ca1c3dd5e0894ae8159b2e8665d8b986021c4e))
* fix/42-test-failures ([5ba8689](https://github.com/stuttgart-things/claim-machinery-api/commit/5ba8689049ede945f89892ef23663dd022c48ce1))
* main ([86336b5](https://github.com/stuttgart-things/claim-machinery-api/commit/86336b5ba7f96d943d07f911a58116af994dde73))
* main ([7044f48](https://github.com/stuttgart-things/claim-machinery-api/commit/7044f48fe2ea090aaf73c1ce15f3d0decacf1da1))
* main ([3f9f17b](https://github.com/stuttgart-things/claim-machinery-api/commit/3f9f17b7fca9fe4e33a21a9cf763bc3ebcd2b034))

# [0.3.0](https://github.com/stuttgart-things/claim-machinery-api/compare/v0.2.0...v0.3.0) (2026-02-01)


### Features

* feat/add-dagger-service ([951a3bd](https://github.com/stuttgart-things/claim-machinery-api/commit/951a3bdd357efe911612038c3e83e73f936612d8))

# [0.2.0](https://github.com/stuttgart-things/claim-machinery-api/compare/vv0.1.3...v0.2.0) (2026-01-31)


### Features

* feat/add-logo ([d3271a9](https://github.com/stuttgart-things/claim-machinery-api/commit/d3271a9fe197e53bd69a9a481928c1b7851c9bb9))
