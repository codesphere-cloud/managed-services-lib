# Changelog

## [0.7.0](https://github.com/codesphere-cloud/managed-services-lib/compare/v0.6.0...v0.7.0) (2026-08-21)


### ⚠ BREAKING CHANGES

* explicit provider contract ([#15](https://github.com/codesphere-cloud/managed-services-lib/issues/15))

### refac

* explicit provider contract ([#15](https://github.com/codesphere-cloud/managed-services-lib/issues/15)) ([31fe171](https://github.com/codesphere-cloud/managed-services-lib/commit/31fe17195dc7f33b4266681ff50dd2a4518b64ad))


### Bug Fixes

* **deps:** update module github.com/onsi/ginkgo/v2 to v2.32.1 ([#20](https://github.com/codesphere-cloud/managed-services-lib/issues/20)) ([ebdaaa2](https://github.com/codesphere-cloud/managed-services-lib/commit/ebdaaa22241a86448de0591736f39680e57e0582))
* **deps:** update module github.com/stretchr/testify to v1.12.1 ([#25](https://github.com/codesphere-cloud/managed-services-lib/issues/25)) ([632d92b](https://github.com/codesphere-cloud/managed-services-lib/commit/632d92bea38208654855639b76f60ea2bc15d9ed))
* retry jobs in separate pods to retain logs ([#13](https://github.com/codesphere-cloud/managed-services-lib/issues/13)) ([4bc7bc4](https://github.com/codesphere-cloud/managed-services-lib/commit/4bc7bc40864daaa6187adc05120ed64a9c72b792))

## [0.6.0](https://github.com/codesphere-cloud/managed-services-lib/compare/v0.5.0...v0.6.0) (2026-08-04)


### Features

* expose container resources on service jobs ([#11](https://github.com/codesphere-cloud/managed-services-lib/issues/11)) ([fbf7c18](https://github.com/codesphere-cloud/managed-services-lib/commit/fbf7c1883cde6d04b5f73338405879f2804571ab))

## [0.5.0](https://github.com/codesphere-cloud/managed-services-lib/compare/v0.4.0...v0.5.0) (2026-07-22)


### Features

* add image pull secrets support to service job ([d0cc9da](https://github.com/codesphere-cloud/managed-services-lib/commit/d0cc9da7f1bbfdfd52cb75ae508c220c5056d065))


### Bug Fixes

* vulnerability scan findings ([af5e754](https://github.com/codesphere-cloud/managed-services-lib/commit/af5e754c7ba933ded89532d321c6b662752b9a03))

## [0.4.0](https://github.com/codesphere-cloud/managed-services-lib/compare/v0.3.1...v0.4.0) (2026-07-21)


### ⚠ BREAKING CHANGES

* make backups a generic, opt-in provider capability ([#9](https://github.com/codesphere-cloud/managed-services-lib/issues/9))

### Features

* add helpers for k8s jobs and wrappers for backup operations ([#5](https://github.com/codesphere-cloud/managed-services-lib/issues/5)) ([c7ec776](https://github.com/codesphere-cloud/managed-services-lib/commit/c7ec77627d0ee61b0e613dbac559669729d62310))
* make backups a generic, opt-in provider capability ([#9](https://github.com/codesphere-cloud/managed-services-lib/issues/9)) ([81f16e6](https://github.com/codesphere-cloud/managed-services-lib/commit/81f16e6c370a236f662765e52c405c1ca41cbaa1))

## [0.3.1](https://github.com/codesphere-cloud/managed-services-lib/compare/v0.3.0...v0.3.1) (2026-07-15)


### Bug Fixes

* RegisterRoutes provider type parameters not inferrable ([22d8202](https://github.com/codesphere-cloud/managed-services-lib/commit/22d82021c2cfc31a0f2bfd95f5e0029588b43526))

## [0.3.0](https://github.com/codesphere-cloud/managed-services-lib/compare/v0.2.0...v0.3.0) (2026-07-15)


### Features

* allow CreateParams to be any type ([39b447f](https://github.com/codesphere-cloud/managed-services-lib/commit/39b447fba68e4f412eafa2efac25b9473d03c609))

## [0.2.0](https://github.com/codesphere-cloud/managed-services-lib/compare/v0.1.0...v0.2.0) (2026-07-11)


### ⚠ BREAKING CHANGES

* flatten pkg package and prepare for open source ([#3](https://github.com/codesphere-cloud/managed-services-lib/issues/3))

### refac

* flatten pkg package and prepare for open source ([#3](https://github.com/codesphere-cloud/managed-services-lib/issues/3)) ([b5ec8b9](https://github.com/codesphere-cloud/managed-services-lib/commit/b5ec8b9cc1bbfa09603204102a9bb1e9f88fed55))

## 0.1.0 (2026-07-10)


### Features

* initial commit ([a46d2a4](https://github.com/codesphere-cloud/managed-services-lib/commit/a46d2a4b452140f87fe9c57dbf986b070ee0c203))
