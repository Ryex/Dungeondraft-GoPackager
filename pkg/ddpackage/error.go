package ddpackage

import "errors"

var (
	ErrUnsupportedGodot   = errors.New("unsupported godot package version")
	ErrEmptyFileList      = errors.New("empty file list")
	ErrMissingPackJSON    = errors.New("missing pack.json")
	ErrPackJSONRead       = errors.New("pack.json read error")
	ErrPackJSONParse      = errors.New("failed to parse pack json")
	ErrInvalidPackage     = errors.New("not a valid package")
	ErrInvalidPackJSON    = errors.New("invalid pack.json")
	ErrUnsetPackID        = errors.New("pack id not set")
	ErrUnsetUnpackedPath  = errors.New("pack unpacked path not set")
	ErrTagsRead           = errors.New("tags read error")
	ErrTagsWrite          = errors.New("tags write error")
	ErrTagsParse          = errors.New("tag file parse error")
	ErrMetadataRead       = errors.New("metadata read error")
	ErrWallParse          = errors.New("wall file parse error")
	ErrWallSave           = errors.New("wall file save error")
	ErrTilesetParse       = errors.New("tileset file parse error")
	ErrTilesetSave        = errors.New("tileset file save error")
	ErrPackageNotLoaded   = errors.New("package not loaded")
	ErrResourceNotFound   = errors.New("resource not found")
	ErrPackageNotUnpacked = errors.New("package not loaded in unpacked mode")
	ErrPackageNotPacked   = errors.New("package not loaded in packed mode")
	ErrReadUnpacked       = errors.New("unpacked resource read error")
	ErrReadPacked         = errors.New("packed resource read error")
	ErrJSONStandardize    = errors.New("error standardizing json, while trailing commas are supported the file must otherwise be valid json")
)
