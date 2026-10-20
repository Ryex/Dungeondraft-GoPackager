# Sets up the MSYS2 UCRT64 gcc toolchain for cgo.
#
# Unlike setcc.ps1, which uses the zig cc compiler, this works with the race
# detector: the zig toolchain cannot link the thread sanitizer on Windows.
#
#   . .\setcc-gcc.ps1
#   go build ./...

$ucrt = "$env:USERPROFILE\scoop\apps\msys2\current\ucrt64\bin"

if (-Not (Test-Path "$ucrt/gcc.exe")) {
    Write-Error "The MSYS2 UCRT64 toolchain is missing. Install it with:"
    Write-Error "  & `"$env:USERPROFILE\scoop\apps\msys2\current\usr\bin\pacman.exe`" -S --needed --noconfirm mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-pkgconf"
    return
}

# The DLLs of the toolchain have to be on the PATH, both to link the binaries
# and to run the tests.
if (-Not ($env:PATH -split ";" -contains $ucrt)) {
    $env:PATH = "$ucrt;$env:PATH"
}

$env:CC = "gcc"
$env:CXX = "g++"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = 1

# CGO_CFLAGS is deliberately not set: -fno-sanitize-trap=undefined is a
# clang/zig specific flag that gcc neither needs nor understands.
