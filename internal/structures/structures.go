package structures

type PackageHeadersBytes struct {
	H1        uint32
	H2        uint32
	H3        uint32
	H4        uint32
	H5        uint32
	H7        [16]uint32
	FileCount uint32
}

type FileInfoBytes struct {
	Offset uint64
	Size   uint64
	Md5    [16]byte
}

type FileInfo struct {
	Path   string
	Offset int64
	Size   int64
	Md5    string
}
