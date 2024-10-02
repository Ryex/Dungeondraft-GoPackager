<div align="center">

# Dungeondraft-GoPackager

  <img src="cmd/dungeondraft-packager/Icon.png" width="200"> 

## Dungeondraft Packer, Unpacker, and Editor

</div>



GUI and Command-line utilities to pack, unpack, and edit custom assets, and tags for [Dungeondraft](https://dungeondraft.net/)


### Instation

you can either install the precompiled binaries available on the [release page](https://github.com/Ryex/Dungeondraft-GoPackager/releases)

Binaries are available for all major platforms:

- [Windows](https://github.com/Ryex/Dungeondraft-GoPackager/releases/download/Dungeondraft-GoPackager-Windows.zip)
- [MacOs](https://github.com/Ryex/Dungeondraft-GoPackager/releases/download/Dungeondraft-GoPackager-macOS.zip)
- [Linux](https://github.com/Ryex/Dungeondraft-GoPackager/releases/download/Dungeondraft-GoPackager-Linux.tgz)

Or, if you have [Go](https://go.dev/) installed you can `go install` them yourself

```shell
go install github.com/ryex/dungeondraft-gopackager/cmd/dungeondraft-packager
go install github.com/ryex/dungeondraft-gopackager/cmd/dungeondraft-packager-cli/
```

and the binaries will be complied and installed to your `$GOBIN` path

#### NOTE

This program depends on a binding to libwebp (to go only webp decoder doesn't lke some images)
if your building from source you'll need to install it
##### MacOS:
```bash
brew install webp
```
##### Linux:
```bash
sudo apt-get update
sudo apt-get install libwebp-dev
```
##### Windows


You'll new to use msys2 and install a toochain, go and one of the `mingw-w64-<system>-x86_64-libwebp` packages to successfully compile


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

If you have issues like the packager not picking up files, try passing in the `-vvv` flags to get info and debug output. Then, makes sure there isn't a structural problem with your package folder.

If you can't find the problem file an issue with the `-vvv` debug output.
