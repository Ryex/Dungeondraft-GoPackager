# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

<!-- insertion marker -->
## [v2.0.3](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v2.0.3) - 2024-11-28

<small>[Compare with v2.0.2](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v2.0.2...v2.0.3)</small>

### Bug Fixes

- don't generate global tag set by default ([340926a](https://github.com/Ryex/Dungeondraft-GoPackager/commit/340926a631025f0e0e69e933b74471bee6b0d83f) by Rachel Powers).
- prevent tileset data save on preview, default color white ([fe90372](https://github.com/Ryex/Dungeondraft-GoPackager/commit/fe9037261b5fe6384d05bf78dd571b641cc6a776) by Rachel Powers).

## [v2.0.2](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v2.0.2) - 2024-11-08

<small>[Compare with v2.0.1](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v2.0.1...v2.0.2)</small>

### Bug Fixes

- prevent crash generating large amounts of thumbnails ([92742d3](https://github.com/Ryex/Dungeondraft-GoPackager/commit/92742d3185e72104954947b4da33f5917dbcd519) by Rachel Powers).

## [v2.0.1](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v2.0.1) - 2024-10-24

<small>[Compare with v2.0.0](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v2.0.0...v2.0.1)</small>

### Bug Fixes

- sort object *inside* tags too ([65df801](https://github.com/Ryex/Dungeondraft-GoPackager/commit/65df801f0a1fdf97ecdd7dd3e5ab7b584011ef46) by Rachel Powers).

## [v2.0.0](https://github.com/Ryex/Dungeondraft-GoPackager/releases/tag/v2.0.0) - 2024-10-23

<small>[Compare with v2.0.0+pre2](https://github.com/Ryex/Dungeondraft-GoPackager/compare/v2.0.0+pre2...v2.0.0)</small>

### Features

- edit cli command ([ccbb511](https://github.com/Ryex/Dungeondraft-GoPackager/commit/ccbb5115f2d2499e42b0e792d0c32fb02fc473f6) by Rachel Powers).
- rebuild example path when switching between split and delimited mode for tag generation ([1dd0e4c](https://github.com/Ryex/Dungeondraft-GoPackager/commit/1dd0e4c851aae9fa77c483c0e1d111df67112dcf) by Rachel Powers).
- make "show thumbnails" toggle remembered ([e2dff05](https://github.com/Ryex/Dungeondraft-GoPackager/commit/e2dff05e43d887a6be4afa7b0d21132c38678f44) by Rachel Powers).
- drag and drop files ([4ab454f](https://github.com/Ryex/Dungeondraft-GoPackager/commit/4ab454f8da52c80191a45353a44ec3ba8b969f05) by Rachel Powers).
- default loglevel is now "warn" ([0bca081](https://github.com/Ryex/Dungeondraft-GoPackager/commit/0bca081b7b730303eeb348897c614bb3047ebaf2) by Rachel Powers).
- toggle between thumbnails and source images ([b125198](https://github.com/Ryex/Dungeondraft-GoPackager/commit/b1251983dba672f0d79fd3e24248eb526771c7bf) by Rachel Powers).
- don't package thumbnails for which there is no matching texture ([fc0e551](https://github.com/Ryex/Dungeondraft-GoPackager/commit/fc0e551f8709653f517cd34e59aeaa6688ddd910) by Rachel Powers).
- show resources by tag ([36b800a](https://github.com/Ryex/Dungeondraft-GoPackager/commit/36b800ac187b5cccd87678785f60bea9540d7687) by Rachel Powers).
- ui "toggle" that may get used elsewhere ([1fd35e2](https://github.com/Ryex/Dungeondraft-GoPackager/commit/1fd35e23547d32c53be8fdaf6f2fddc7e4e40f57) by Rachel Powers).
- add by-tag resource listing to cli ([9a10975](https://github.com/Ryex/Dungeondraft-GoPackager/commit/9a109750760264e820e41a1d92e98f8be60732e6) by Rachel Powers).
- improve progress feedback from cli ([e31fea7](https://github.com/Ryex/Dungeondraft-GoPackager/commit/e31fea72feb0ba29ee7a028546b3a40f6feb4a9d) by Rachel Powers).
- cli list files in packed order ([5203473](https://github.com/Ryex/Dungeondraft-GoPackager/commit/5203473094b7e9bf2e0ccff4ebdfb401cab66143) by Rachel Powers).
- Write order sorting for file list ([64c64b3](https://github.com/Ryex/Dungeondraft-GoPackager/commit/64c64b37bb0f23609ae7bf6341bed3cb1ed20169) by Rachel Powers).
- move to a 'just in time' method of writing file size when packaging ([9a5e88e](https://github.com/Ryex/Dungeondraft-GoPackager/commit/9a5e88ecb0c561726ca93b09b0c02de0d390b837) by Rachel Powers).

### Bug Fixes

- sort package tags alphabeticaly ([f586574](https://github.com/Ryex/Dungeondraft-GoPackager/commit/f58657473deb20c73d4873b45d0065a58b9aeed3) by Rachel Powers).
- use bakground for object tag list ([8e428bf](https://github.com/Ryex/Dungeondraft-GoPackager/commit/8e428bfe27fdd07838b2a1bcb42b6d67111408f7) by Rachel Powers).
- spinners shouldn't have percision errors, version spinner should behave like a version ([5ef1be5](https://github.com/Ryex/Dungeondraft-GoPackager/commit/5ef1be56fb1cead9ff9ec2bae1498e9ef62c1e2d) by Rachel Powers).
- enable reading json that may have trailing commas (don't save it though) ([4113c6d](https://github.com/Ryex/Dungeondraft-GoPackager/commit/4113c6d05249d341c9bd4847b3c426c56e58203b) by Rachel Powers).
- remember last selected resource tab, wallends don't have their own metadata ([3ba7ad0](https://github.com/Ryex/Dungeondraft-GoPackager/commit/3ba7ad0398f32d85ee9468c3dfed5449d567514e) by Rachel Powers).
- update example path when changing seperators ([ab12a0b](https://github.com/Ryex/Dungeondraft-GoPackager/commit/ab12a0bcd39392d13db69f5cd3c8624e12476130) by Rachel Powers).
- don't try to pack or unpack if output path is empty ([ef4243a](https://github.com/Ryex/Dungeondraft-GoPackager/commit/ef4243a500e8cb95a5148f27df035842b19b5789) by Rachel Powers).
- remove webp info from readme, useing our own static binding now ([16a5b6e](https://github.com/Ryex/Dungeondraft-GoPackager/commit/16a5b6e348ecdf46a8eef342f4a8042a8d135202) by Rachel Powers).
- add packname as parent folder during extraction ([e0d2eeb](https://github.com/Ryex/Dungeondraft-GoPackager/commit/e0d2eebd07cf8aefa0f1462587833777be320d8e) by Rachel Powers).
- change default color to transparent ([1476df8](https://github.com/Ryex/Dungeondraft-GoPackager/commit/1476df8237809df4b2ea7d695965287aeab0ef9f) by Rachel Powers).
- ignore thumbnail toggle for  non texture ([ca497c7](https://github.com/Ryex/Dungeondraft-GoPackager/commit/ca497c762aeb459002a390854738ce5149703948) by Rachel Powers).
- hide tag generation dialog after tags have finshed generating ([23452c1](https://github.com/Ryex/Dungeondraft-GoPackager/commit/23452c1980b4809188423b6aed07dc4db7b713de) by Rachel Powers).
- swap position of generate tags and edit tag sets buttons ([d9e0fa4](https://github.com/Ryex/Dungeondraft-GoPackager/commit/d9e0fa4c2b0ceaaf2a51c2aeba45d8c2ac2c3588) by Rachel Powers).
- update cli GenerateTumbnails call for multi error return ([84099c0](https://github.com/Ryex/Dungeondraft-GoPackager/commit/84099c08bd64a38f7572fbe0ec8cc86dabc72775) by Rachel Powers).
- ui cleanup ([331c600](https://github.com/Ryex/Dungeondraft-GoPackager/commit/331c6009c6c79633731ac12712caa8c690a4b51d) by Rachel Powers).
- change image background raster to not use a pure white ([99994a3](https://github.com/Ryex/Dungeondraft-GoPackager/commit/99994a3471bd291ad1979a97facb3b9d268626e4) by Rachel Powers).
- correct resource path for tags ([ef4882c](https://github.com/Ryex/Dungeondraft-GoPackager/commit/ef4882cc3e2243c2afa3d0ea630ece189a88d79e) by Rachel Powers).
- thumbnail generation now confirms to dd style even for paths and walls ([9f4d5e2](https://github.com/Ryex/Dungeondraft-GoPackager/commit/9f4d5e261eee2c269f5d262562455dd8ed5a1a4e) by Rachel Powers).
- *finaly* fix webp support and thumbnail generation, for real this time ([2cdfd15](https://github.com/Ryex/Dungeondraft-GoPackager/commit/2cdfd1506dda4bb026c28325788dcc80ebf4077e) by Rachel Powers).
- add input background to example tag and sets lists ([5a96ee6](https://github.com/Ryex/Dungeondraft-GoPackager/commit/5a96ee6d260692b07c60dc6e3ba785001c8ffdc4) by Rachel Powers).
- fix tag by single seperator ([3d3abf2](https://github.com/Ryex/Dungeondraft-GoPackager/commit/3d3abf269daf1e069eccd97ac841f33161c3c1ad) by Rachel Powers).
- turn off default entry validation for bound entries ([8f54fb0](https://github.com/Ryex/Dungeondraft-GoPackager/commit/8f54fb0e343840c6f5508f4e85e74576b622d8e1) by Rachel Powers).
- use correct relative path when generating tags ([83b34ee](https://github.com/Ryex/Dungeondraft-GoPackager/commit/83b34eebf3759a431a0f5f1cb8676857dd273bc6) by Rachel Powers).
- sort tags alphabeticaly for display ([3b67c47](https://github.com/Ryex/Dungeondraft-GoPackager/commit/3b67c47b971498e4597cfd3148cea5416c2894ab) by Rachel Powers).
- use opened folder name not opened folder parent for pach json name default ([255cfeb](https://github.com/Ryex/Dungeondraft-GoPackager/commit/255cfeb178a9154678ed47448a54cfec2b1351f8) by Rachel Powers).
- improve thumbnail generation ([2856be3](https://github.com/Ryex/Dungeondraft-GoPackager/commit/2856be332835af57d6fc806876b068f868cbd41d) by Rachel Powers).
- save generated tags ([106f0ac](https://github.com/Ryex/Dungeondraft-GoPackager/commit/106f0acde7753ff908af6a9445f756a1e6e98333) by Rachel Powers).
- premptivly filter thumbnail fs events when generating thumbnails ([e7fb988](https://github.com/Ryex/Dungeondraft-GoPackager/commit/e7fb988267c67c3d98a25a1cdfd4f9d8da8ba526) by Rachel Powers).
- change default tag delimiter ([9160de9](https://github.com/Ryex/Dungeondraft-GoPackager/commit/9160de97c390859d9ad2eb2189b314d8bd3b34fc) by Rachel Powers).
- update logo colors to differeneate ([0929f90](https://github.com/Ryex/Dungeondraft-GoPackager/commit/0929f90c3f19a6407e8194ab2fc7c053b6518c97) by Rachel Powers).

### Code Refactoring

- cleanup bindings code ([4bb99c5](https://github.com/Ryex/Dungeondraft-GoPackager/commit/4bb99c5c56da309032969e97d5609e4ac22ae89b) by Rachel Powers).

### Performance Improvements

- vastly speed up loading of resource folder ([9df726b](https://github.com/Ryex/Dungeondraft-GoPackager/commit/9df726b545760a235bc68c946066f304d3bf5475) by Rachel Powers).

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
