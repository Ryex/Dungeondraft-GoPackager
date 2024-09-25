package ddpackage

import "errors"

var (
	EmptyFileListError     = errors.New("empty file list")
	MissingPackJsonError   = errors.New("missing pack.json")
	PackJsonReadError      = errors.New("pack.json read error")
	InvalidPackJsonError   = errors.New("invalid pack.json")
	UnsetPackIdError       = errors.New("pack id not set")
	UnsetUnpackedPathError = errors.New("pack unpacked path not set")
	TagsReadError          = errors.New("tags read error")
	TagsParseError         = errors.New("tag file parse error")
	MetadataReadError      = errors.New("metadata read error")
	WallParseError         = errors.New("wall file parse error")
	TilesetParseError      = errors.New("tileset file parse error")
)
