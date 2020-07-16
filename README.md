# Dungeondraft-GoPackage

## Dungeondraft Packer and Unpacker

Command-line utilities to pack and unpack custom assets for Dungeondraft

The pack tool by default does not on its own generate an ID for your package.

So, it must have been pack at least once by Dungeondraft itself and have a valid `pack.json`
OR
A `pack.json` must be created by using the flags to pass in th pack name and version with an optional author.

Windows executable are provided so you don't have to build it yourself.

### Usage:
```dungeondraft-unpack.exe -h```
will show the help

```dungeondraft-unpack.exe [args] <.dungeondraft_pack file> <dest folder>```
The assets contained in the `.dungeondraft_pack`  file will be written to a folder the same name as the package under the dest folder.

```dungeondraft-pack [args] <input folder> <dest folder>```
The assets in the input folder (provided there is a valid `pack.json`) will be written to a `<packname>.dungeondraft_pack` file in the destination directory.

```dungeondraft-pack [args] -G [-E] -N <packname> -V <version> [-A <author>] <input folder>```
A valid `pack.json` with a new id and the provided values will be created in the input directory (-E overwrites an existing `pack.json`).
The packer will then exit.

```dungeondraft-pack [args] [-E] -N <packname> -V <version> [-A <author>] <input folder> <dest folder>```
A valid `pack.json` with a new id and the provided values will be created in the input directory (-E overwrites an existing `pack.json`).
Then the assets in the input folder will be written to a `<packname>.dungeondraft_pack` file in the destination directory.

### If You Have Issues

If you have issues like the packager not picking up files, try passing in teh `-v` and `-vv` flags to get info and debug output. to makes sure there isn't a structural problem with your package folder.

If you can't fid the problem file an issue with the `-vv` debug output.
