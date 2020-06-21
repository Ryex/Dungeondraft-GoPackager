# Dungeondraft-GoPackage

## Dungeondraft Packer and Unpacker

Command-line utilities to pack and unpack custom assets for Dungeondraft

The pack tool does not on its own generate an ID for your package so it must have been pack at least once by Dungeondraft itself and have a valid `pack.json`

Windows executables are provided so you don't have to build it yourself.

### Usage:
```dungeondraft-unpack.exe [args] <.dungeondraft_pack file> <dest folder>```
The assets contained in the `.dungeondraft_pack`  file will be written to a folder the same name as the package under the dest folder

```dungeondraft-pack [args] <input folder> <dest folder>```
The assets in the input folder (provided there is a valid pack.json) will be written to a `<packname>.dungeondraft_pack` file in the destination directory
