package ddpackage

import "errors"

var (
	ErrUnsupportedGodot   = errors.New("unsupported godot package version")
	ErrEmptyFileList      = errors.New("empty file list")
	ErrMissingPackJSON    = errors.New("missing pack.json")
	ErrPackJSONRead       = errors.New("pack.json read error")
	ErrInvalidPackage     = errors.New("not a valid package")
	ErrInvalidPackJSON    = errors.New("invalid pack.json")
	ErrUnsetPackID        = errors.New("pack id not set")
	ErrUnsetUnpackedPath  = errors.New("pack unpacked path not set")
	ErrTagsRead           = errors.New("tags read error")
	ErrTagsWrite          = errors.New("tags write error")
	ErrTagsParse          = errors.New("tag file parse error")
	ErrMetadataRead       = errors.New("metadata read error")
	ErrWallParse          = errors.New("wall file parse error")
	ErrTilesetParse       = errors.New("tileset file parse error")
	ErrPackageNotLoaded   = errors.New("package not loaded")
	ErrResourceNotFound   = errors.New("resource not found")
	ErrPackageNotUnpacked = errors.New("package not loaded in unpacked mode")
	ErrPackageNotPacked   = errors.New("package not loaded in packed mode")
	ErrReadUnpacked       = errors.New("unpacked resource read error")
	ErrReadPacked         = errors.New("packed resource read error")
)
