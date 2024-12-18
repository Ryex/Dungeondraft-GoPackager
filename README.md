<div align="center">

# Dungeondraft-GoPackager

  <img src="cmd/dungeondraft-packager/Icon.png" width="200"> 

## Dungeondraft Packer, Unpacker, and Editor

</div>

GUI and Command-line utilities to pack, unpack, and edit custom assets, and tags for [Dungeondraft](https://dungeondraft.net/)

Contribute translations on [Crowdin](https://crowdin.com/project/dundeondraft-gopackager)

### Features

- Pack custom asset packs
  - Set or generate a package ID
- Unpack packaged asset packs (useful to combine multiple packs or edit tags)
- View assets and tags of packed or unpacked packages
- Edit tags of individual assets in an unpacked package
- Edit tag sets
- Edit metadata for Walls and Tilesets (custom color, tileset name, etc.)
- Generate tags from the folder structure of the package
- Generate thumbnails for assets in an unpacked package
  - ~20 seconds for 42K thumbnails!
- GUI or CLI interface. Automate your workflow!


### Installation

You can either install the precompiled binaries available on the [release page](https://github.com/Ryex/Dungeondraft-GoPackager/releases)

Binaries are available for all major platforms:

- [Windows](https://github.com/Ryex/Dungeondraft-GoPackager/releases/latest/download/Dungeondraft-GoPackager-Windows.zip)
- [MacOs](https://github.com/Ryex/Dungeondraft-GoPackager/releases/latest/download/Dungeondraft-GoPackager-macOS.zip)
- [Linux](https://github.com/Ryex/Dungeondraft-GoPackager/releases/latest/download/Dungeondraft-GoPackager-Linux.tar.gz)

Or, if you have [Go](https://go.dev/) installed you can `go install` them yourself

```shell
go install github.com/ryex/dungeondraft-gopackager/cmd/dungeondraft-packager
go install github.com/ryex/dungeondraft-gopackager/cmd/dungeondraft-packager-cli/
```

### GUI
Version 2.0 of Dungeondraft-GoPackager comes with a nice GUI!

Simply enter a path to, or browse for, your packed `.dungeondraft_pack` or unpacked package folder to get started!


<!-- TODO: ADD Nice GUI images -->


### CLI Usage

#### Show Help
```
dungeondraft-packager[-cli][.exe] -h
```

#### Unpack Assets
```
dungeondraft-packager-cli[.exe] unpack <input-path> <destination-path> [flags]
```
The assets contained in the `.dungeondraft_pack`  file will be written to a folder the same name as the package under the dest folder.

#### Pack Assets
```
dungeondraft-packager-cli[.exe] pack <input-path> <destination-path> [flags]
```
The assets in the input folder (provided there is a valid `pack.json`) will be written to a `<packname>.dungeondraft_pack` file in the destination directory.

#### New pack.json
```
dungeondraft-packager-cli[.exe] generate (gen) pack --name=STRING --author=STRING <input-path> [flags]
```
A valid `pack.json` with a new id and the provided values will be created in the input directory (-O overwrites an existing `pack.json`).


### If You Have Issues

If you have issues like the packager not picking up files, try passing in the `--log-level=info` or `--log-level=debug` flags to get info and debug output. Then, makes sure there isn't a structural problem with your package folder.

If you can't find the problem file an issue with the `--log-level=debug` debug output.
