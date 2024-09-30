# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

<!-- insertion marker -->
## [v2.0.0+pre2](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v2.0.0+pre2) - 2024-09-30

<small>[Compare with v2.0.0+pre-1](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v2.0.0+pre-1...v2.0.0+pre2)</small>

### Features

- generate tags dialog ([aa162a2](https://github.com/Ryex/Dungeondraft-GoPackager/commit/aa162a247ff2047c9ee20f67820a3c3ff1586bea) by Rachel Powers).
- add about and credits, add buttons to top right ([4a2bd2f](https://github.com/Ryex/Dungeondraft-GoPackager/commit/4a2bd2ffc9bf19e3ba5dcf40efe5bf60507a5d37) by Rachel Powers).
- vastly improved logo ([a9a99e2](https://github.com/Ryex/Dungeondraft-GoPackager/commit/a9a99e2fd0c8de9b3214ef9f64562b1b79c3dfe3) by Rachel Powers).
- tag set editor ([322f57d](https://github.com/Ryex/Dungeondraft-GoPackager/commit/322f57d8d72f0ddf1360fe0af5b4c2ca010c9cdd) by Rachel Powers).
- configre gui logging via file ([507a73d](https://github.com/Ryex/Dungeondraft-GoPackager/commit/507a73d8e164b99cb600c11521e1af82d982bee6) by Rachel Powers).
- support updating package list after build ([4ae2473](https://github.com/Ryex/Dungeondraft-GoPackager/commit/4ae2473df4dd888c953990b2bcfd995aa1b34dbf) by Rachel Powers).
- editable assest metadata ([97747d9](https://github.com/Ryex/Dungeondraft-GoPackager/commit/97747d9f45c63015946901df57b7aee4b41497a1) by Rachel Powers).
- edit resource tags ([cc92de1](https://github.com/Ryex/Dungeondraft-GoPackager/commit/cc92de1a797245340acd525dae4aef7877257912) by Rachel Powers).
- edit package settings, create new pack.json ([024a1ce](https://github.com/Ryex/Dungeondraft-GoPackager/commit/024a1ce1cac64aff3485396970c450da264f51e0) by Rachel Powers).
- filter resource list with globs ([3d991f9](https://github.com/Ryex/Dungeondraft-GoPackager/commit/3d991f9bbe9e3d67b4070381252a98ee80f7931c) by Rachel Powers).
- `list tags` cli command ([3346bd8](https://github.com/Ryex/Dungeondraft-GoPackager/commit/3346bd85072673ce5de89d8fdb88388b42a51931) by Rachel Powers).
- read and write tags and metadata ([4774eef](https://github.com/Ryex/Dungeondraft-GoPackager/commit/4774eefccf7b9f0497bea749dc53a20c8f9d9d39) by Rachel Powers).

### Bug Fixes

- tags are only for objects ([a683150](https://github.com/Ryex/Dungeondraft-GoPackager/commit/a683150bdecb90fd9dfcd2096844778b4016448f) by Rachel Powers).
- clean up some ui padding ([c1aed6f](https://github.com/Ryex/Dungeondraft-GoPackager/commit/c1aed6f8bba14d9f71771907be2b35d8c67d5499) by Rachel Powers).
- update files in index when edited. ([705bdc2](https://github.com/Ryex/Dungeondraft-GoPackager/commit/705bdc2350ae86a30d3d607be8b4dc529cbc5417) by Rachel Powers).
- add missing translations, show id in pack json dialog ([f22a60f](https://github.com/Ryex/Dungeondraft-GoPackager/commit/f22a60fc6704c19de0d14f9eb9a15b606745e0e6) by Rachel Powers).
- improve logging, log to file by default ([83ed7ee](https://github.com/Ryex/Dungeondraft-GoPackager/commit/83ed7ee2c6d1dab9690f12a0496f3440e734ea0a) by Rachel Powers).
- allow () in glob filter ([5954c19](https://github.com/Ryex/Dungeondraft-GoPackager/commit/5954c1993b1cdeb936e0f4f39381a548e9494271) by Rachel Powers).
- pack info dialog checkmarks set on load ([610f224](https://github.com/Ryex/Dungeondraft-GoPackager/commit/610f2245b4a2889b29416c0f9847aced2882a2ad) by Rachel Powers).
- fix broken webp support ([b8205d2](https://github.com/Ryex/Dungeondraft-GoPackager/commit/b8205d2c14f5bf6a28777f39ce8e69cfdb072353) by Rachel Powers).

## [v2.0.0+pre-1](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v2.0.0+pre-1) - 2024-09-25

<small>[Compare with v1.1.0](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v1.1.0...v2.0.0+pre-1)</small>

### Features

- finialize packaging gui ([27210cb](https://github.com/Ryex/Dungeondraft-GoPackager/commit/27210cb576d49edeaf3793e71bca537ab3f73069) by Rachel Powers).
- initial gui ([2f60012](https://github.com/Ryex/Dungeondraft-GoPackager/commit/2f600127f1c6918a5ef4f43554cb14d2d72d1369) by Rachel Powers).
- start of gui ([75fdbd0](https://github.com/Ryex/Dungeondraft-GoPackager/commit/75fdbd07b88885548c9a56974a522c071bb6aa38) by Rachel Powers).
- track image format ([8699a4a](https://github.com/Ryex/Dungeondraft-GoPackager/commit/8699a4a5321e790be87a0d713f2f9572043a873d) by Rachel Powers).
- impl generate thumbnails ([9d62b6f](https://github.com/Ryex/Dungeondraft-GoPackager/commit/9d62b6f631016660fed7587b20d219ba7ede02b5) by Rachel Powers).

### Code Refactoring

- merge packer and unpacker structs ([0e9fdfa](https://github.com/Ryex/Dungeondraft-GoPackager/commit/0e9fdfadca20dcdcb33a83d44de50b72d76dc5e9) by Rachel Powers).
- better FileInfo init ([dc92fff](https://github.com/Ryex/Dungeondraft-GoPackager/commit/dc92fffeed05a5092b24710d90b66d363b731217) by Rachel Powers).
- rename binary ([745b024](https://github.com/Ryex/Dungeondraft-GoPackager/commit/745b024d592c011e6f706a4e22deeeef88fa84dc) by Rachel Powers).
- merge command binaries, use kong for cli parsing, seperate generate command, list cmd ([fc9b8ca](https://github.com/Ryex/Dungeondraft-GoPackager/commit/fc9b8ca7ac15eaed1f291d8ff9fb0b48ca689901) by Rachel Powers).
- update for useing with latest dungeondraft (GoDot 3.4.2) ([872512f](https://github.com/Ryex/Dungeondraft-GoPackager/commit/872512faec06d52be95b93ecef39c0a46cfe391e) by Rachel Powers).

## [v1.1.0](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v1.1.0) - 2020-07-16

<small>[Compare with v1.0.2](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v1.0.2...v1.1.0)</small>

## [v1.0.2](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v1.0.2) - 2020-07-16

<small>[Compare with v1.0.1](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v1.0.1...v1.0.2)</small>

## [v1.0.1](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v1.0.1) - 2020-07-07

<small>[Compare with v1.0.0](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v1.0.0...v1.0.1)</small>

## [v1.0.0](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v1.0.0) - 2020-06-21

<small>[Compare with v0.1.0](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v0.1.0...v1.0.0)</small>

## [v0.1.0](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v0.1.0) - 2020-06-20

<small>[Compare with first commit](https://github.com/Ryex/Dungeondraft-GoPackager/compare/b36d63374d2e7f5ca3f5553c37d12561dcc3956b...v0.1.0)</small>

## [2.0.0+pre-1](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/2.0.0+pre-1) - 2024-09-24

<small>[Compare with v1.1.0](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v1.1.0...2.0.0+pre-1)</small>

### Features

- finialize packaging gui ([27210cb](https://github.com/Ryex/Dungeondraft-GoPackager/commit/27210cb576d49edeaf3793e71bca537ab3f73069) by Rachel Powers).
- initial gui ([2f60012](https://github.com/Ryex/Dungeondraft-GoPackager/commit/2f600127f1c6918a5ef4f43554cb14d2d72d1369) by Rachel Powers).
- start of gui ([75fdbd0](https://github.com/Ryex/Dungeondraft-GoPackager/commit/75fdbd07b88885548c9a56974a522c071bb6aa38) by Rachel Powers).
- track image format ([8699a4a](https://github.com/Ryex/Dungeondraft-GoPackager/commit/8699a4a5321e790be87a0d713f2f9572043a873d) by Rachel Powers).
- impl generate thumbnails ([9d62b6f](https://github.com/Ryex/Dungeondraft-GoPackager/commit/9d62b6f631016660fed7587b20d219ba7ede02b5) by Rachel Powers).

### Code Refactoring

- merge packer and unpacker structs ([0e9fdfa](https://github.com/Ryex/Dungeondraft-GoPackager/commit/0e9fdfadca20dcdcb33a83d44de50b72d76dc5e9) by Rachel Powers).
- better FileInfo init ([dc92fff](https://github.com/Ryex/Dungeondraft-GoPackager/commit/dc92fffeed05a5092b24710d90b66d363b731217) by Rachel Powers).
- rename binary ([745b024](https://github.com/Ryex/Dungeondraft-GoPackager/commit/745b024d592c011e6f706a4e22deeeef88fa84dc) by Rachel Powers).
- merge command binaries, use kong for cli parsing, seperate generate command, list cmd ([fc9b8ca](https://github.com/Ryex/Dungeondraft-GoPackager/commit/fc9b8ca7ac15eaed1f291d8ff9fb0b48ca689901) by Rachel Powers).
- update for useing with latest dungeondraft (GoDot 3.4.2) ([872512f](https://github.com/Ryex/Dungeondraft-GoPackager/commit/872512faec06d52be95b93ecef39c0a46cfe391e) by Rachel Powers).



## [v1.1.0](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v1.1.0) - 2020-07-16

<small>[Compare with v1.0.2](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v1.0.2...v1.1.0)</small>

## [v1.0.2](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v1.0.2) - 2020-07-16

<small>[Compare with v1.0.1](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v1.0.1...v1.0.2)</small>

## [v1.0.1](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v1.0.1) - 2020-07-07

<small>[Compare with v1.0.0](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v1.0.0...v1.0.1)</small>

## [v1.0.0](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v1.0.0) - 2020-06-21

<small>[Compare with v0.1.0](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v0.1.0...v1.0.0)</small>

## [v0.1.0](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v0.1.0) - 2020-06-20

<small>[Compare with first commit](https://github.com/Ryex/Dungeondraft-GoPackager/compare/b36d63374d2e7f5ca3f5553c37d12561dcc3956b...v0.1.0)</small>
