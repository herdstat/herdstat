# Changelog

## [0.10.0](https://github.com/herdstat/herdstat/compare/v0.9.2...v0.10.0) (2023-04-02)


### Features

* support for version command ([#54](https://github.com/herdstat/herdstat/issues/54)) ([9d49af4](https://github.com/herdstat/herdstat/commit/9d49af45c0fb8a8395b9b22d6acf49db9e24d176))

## [0.9.2](https://github.com/herdstat/herdstat/compare/v0.9.1...v0.9.2) (2023-03-25)


### Bug Fixes

* remove one day off for commits ([ae09201](https://github.com/herdstat/herdstat/commit/ae09201666800ee302348d4fff64b702dc598e07))

## [0.9.1](https://github.com/herdstat/herdstat/compare/v0.9.0...v0.9.1) (2023-03-20)


### Bug Fixes

* log analysis period on debug level ([ae0d16b](https://github.com/herdstat/herdstat/commit/ae0d16b44d88ba42186eb4dcf1f9179e26f770d9))
* use commit time instead of author time when analyzing commits ([454109d](https://github.com/herdstat/herdstat/commit/454109d25a2967e5845cef262b6a324553a3ee73))

## [0.9.0](https://github.com/herdstat/herdstat/compare/v0.8.0...v0.9.0) (2023-03-11)


### Features

* allow for excluding commits via filters ([311f1e0](https://github.com/herdstat/herdstat/commit/311f1e0d04e89b2514cd5e241a42643bce497455))

## [0.8.0](https://github.com/herdstat/herdstat/compare/v0.7.0...v0.8.0) (2023-03-04)


### Features

* remove dependency on mergestat and use go-git instead ([#37](https://github.com/herdstat/herdstat/issues/37)) ([4b697fd](https://github.com/herdstat/herdstat/commit/4b697fdbd4fbdad59bb7503101422485c3390a9b))

## [0.7.0](https://github.com/herdstat/herdstat/compare/v0.6.0...v0.7.0) (2023-02-05)


### Features

* include opened issues and PRs as contributions ([44afbbd](https://github.com/herdstat/herdstat/commit/44afbbd8a019613054ab645ff30b9237d0dda109)), closes [#5](https://github.com/herdstat/herdstat/issues/5)
* parameterizable 'quantization' of color spectrum ([8d8c143](https://github.com/herdstat/herdstat/commit/8d8c1433045554756e206c586e78ec32d20d1291))

## [0.6.0](https://github.com/herdstat/herdstat/compare/v0.5.2...v0.6.0) (2023-01-29)


### Features

* add support for light/dark mode ([f95c19e](https://github.com/herdstat/herdstat/commit/f95c19e849724dd158d530d2bfffbb8b90130c80))

## [0.5.2](https://github.com/herdstat/herdstat/compare/v0.5.1...v0.5.2) (2023-01-26)


### Bug Fixes

* fix typo in color configuration key ([63fc9fb](https://github.com/herdstat/herdstat/commit/63fc9fb2c530f1398c6616c9539b53a09f936661))

## [0.5.1](https://github.com/herdstat/herdstat/compare/v0.5.0...v0.5.1) (2023-01-23)


### Bug Fixes

* fix flaky test ([8d591df](https://github.com/herdstat/herdstat/commit/8d591dfe5ef6676d1d5eacee471086eee6137cde))

## [0.5.0](https://github.com/herdstat/herdstat/compare/v0.4.1...v0.5.0) (2023-01-23)


### Features

* support custom primary color for contribution graph ([7636799](https://github.com/herdstat/herdstat/commit/7636799045723f5aee5dbe5e4e8d7eb7af1905c8))

## [0.4.1](https://github.com/herdstat/herdstat/compare/v0.4.0...v0.4.1) (2023-01-10)


### Bug Fixes

* add non-breaking space to contribution count labels ([f4f5fef](https://github.com/herdstat/herdstat/commit/f4f5fefc0082faaed280e9c85cb178e159289903))

## [0.4.0](https://github.com/herdstat/herdstat/compare/v0.3.1...v0.4.0) (2023-01-10)


### Features

* add label with overall number of contributions ([6c2a152](https://github.com/herdstat/herdstat/commit/6c2a152636727f1afa7aee0d0801b58d95916ea0))

## [0.3.1](https://github.com/herdstat/herdstat/compare/v0.3.0...v0.3.1) (2023-01-10)


### Bug Fixes

* use PAT for release please action to enable triggering dependent ([1b4a4d3](https://github.com/herdstat/herdstat/commit/1b4a4d3b252f94f1b534c35e5c3b1c957d5675f2))

## [0.3.0](https://github.com/herdstat/herdstat/compare/v0.2.0...v0.3.0) (2023-01-09)


### Features

* introduce parameter to set GitHub API token ([9557269](https://github.com/herdstat/herdstat/commit/9557269d10eda07efbef353e2f7c68520f761ae7))

## [0.2.0](https://github.com/herdstat/herdstat/compare/v0.1.0...v0.2.0) (2023-01-06)


### ⚠ BREAKING CHANGES

* load mergestat shared library from standard locations
* move repository flag to root command
* refactor viper-related code to make it work
* redefined configuration file structure

### Features

* addition of shorthands for all parameters ([43e5c30](https://github.com/herdstat/herdstat/commit/43e5c30ba5235deaad6cabb8f21b1e71d19acfdb))
* move repository flag to root command ([cc523a3](https://github.com/herdstat/herdstat/commit/cc523a3a0b626b8a778937c115b3ff718de9cda3))
* redefined configuration file structure ([b7183ad](https://github.com/herdstat/herdstat/commit/b7183adcf842232081814cebf8940d54d93e5273))
* support configurable visualization period ([d7d8bcc](https://github.com/herdstat/herdstat/commit/d7d8bcc8d38be1fad5764f79b02d8e2f4526fbb0))


### Bug Fixes

* load mergestat shared library from standard locations ([e81e88a](https://github.com/herdstat/herdstat/commit/e81e88a157db2c1d1b0492175651f791a81677f8))
* refactor viper-related code to make it work ([b7183ad](https://github.com/herdstat/herdstat/commit/b7183adcf842232081814cebf8940d54d93e5273))
* Remove debug options for regular execution ([81240c7](https://github.com/herdstat/herdstat/commit/81240c78d1aeb878bb9aa2ad1d181891d0903339))
* Remove errorneous quotation marks from debug command ([a788ee8](https://github.com/herdstat/herdstat/commit/a788ee818bd25a22a169f75cb56c261ae38bbee8))
* update commands in README ([795a511](https://github.com/herdstat/herdstat/commit/795a51172d079340d013bb0059ab8c8eb368b1a7))
* vertically align weekday axes labels ([74d81c7](https://github.com/herdstat/herdstat/commit/74d81c7f87e10b013e237b64cf8e91bf2cb3a94f))

## 0.1.0 (2023-01-02)


### Miscellaneous Chores

* release 0.1.0 ([c33ac33](https://github.com/herdstat/herdstat/commit/c33ac33d3c12b8f1b6e49fce206f4f2ed5e6078b))
