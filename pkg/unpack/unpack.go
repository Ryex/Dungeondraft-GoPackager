package unpack

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/ryexandrite/dungeondraft-gopackager/internal/structures"
	"gitlab.com/ryexandrite/dungeondraft-gopackager/internal/utils"
)

// Unpacker dungeondraft_pack files in the pck arcive format
type Unpacker struct {
	log         logrus.FieldLogger
	name        string
	RipTextures bool
}

// NewUnpacker builds a new Unpacker
func NewUnpacker(log logrus.FieldLogger, name string) *Unpacker {
	return &Unpacker{
		log:  log,
		name: name,
	}
}

func checkErrorRead(log logrus.FieldLogger, err error, n int, expected int) bool {
	if err != nil {
		log.Error(err)
		return false
	} else if n < expected {
		log.Errorf("Read Failed: attempted to read [%v] bytes, only read [%v]", expected, n)
		return false
	} else {
		return true
	}
}

func checkErrorSeek(log logrus.FieldLogger, err error) bool {
	if err != nil {
		log.Error(err)
		return false
	}
	return true

}

// ExtractPackage extracts the package contents to the filesystem
func (u *Unpacker) ExtractPackage(r io.ReadSeeker, outDir string) (err error) {

	fileList, err := u.ReadPackageFilelist(r)
	if err != nil {
		return
	}

	err = u.ExtractFilelist(r, fileList, outDir)
	return
}

// ExtractFilelist takes a slice of FileInfo and extracts the files from the package at the reader
func (u *Unpacker) ExtractFilelist(r io.ReadSeeker, fileList []structures.FileInfo, outDir string) (err error) {

	outDirPath, err := filepath.Abs(outDir)
	if err != nil {
		return
	}

	dirExists, err := utils.DirExists(outDirPath)
	if err != nil {
		return
	}

	if !dirExists {
		err = os.MkdirAll(outDirPath, 0777)
		if err != nil {
			return
		}
	}

	valid, err := u.isValidPackage(r)

	if !valid {
		err = errors.New("not a valid package")
		return
	}

	for _, packedFile := range fileList {
		path := filepath.Join(outDirPath, filepath.Dir(packedFile.Path))
		fileNameFull := filepath.Base(packedFile.Path)
		fileExt := filepath.Ext(fileNameFull)
		fileName := strings.TrimSuffix(fileNameFull, fileExt)

		l := u.log.WithField("packedFile", packedFile.Path).WithField("offset", packedFile.Offset)

		err = os.MkdirAll(path, 0777)
		if err != nil {
			l.WithField("unpackedFile", path).WithError(err).
				Error("can not make target directory")
			return
		}

		_, err = r.Seek(packedFile.Offset, io.SeekStart)
		if err != nil {
			l.WithError(err).
				Error("can not seek to offset for file data")
		}

		fileData := make([]byte, packedFile.Size)

		var n int
		n, err = r.Read(fileData)
		if !checkErrorRead(l, err, n, int(packedFile.Size)) {
			return
		}

		// if the md5 wasn't blank we would do it here

		if fileExt == ".tex" && u.RipTextures {
			var ext string
			var data []byte
			ext, data, err = utils.RipTexture(fileData)
			if err == nil {
				fileExt = ext
				fileData = data
				fileNameFull = fileName + ext
			}
		}

		filePath := filepath.Join(path, strings.TrimRight(fileNameFull, string([]byte{0})))
		l = l.WithField("unpackedFile", filePath)

		l.Info("writing file")

		var p *os.File
		p, err = os.Create(filePath)
		if err != nil {
			l.WithError(err).Error("can not open file for writing")
			return
		}
		p.Write(fileData)
		err = p.Close()
		if err != nil {
			l.WithError(err).Error("failed to close file")
			return
		}

	}

	return
}

// ReadPackageFilelist Takes an io.reader and attempts to extract a list of files stored in the package
func (u *Unpacker) ReadPackageFilelist(r io.ReadSeeker) (fileList []structures.FileInfo, err error) {

	valid, err := u.isValidPackage(r)

	if !valid {
		err = errors.New("not a valid package")
		return
	}
	// reader is in position, start extracting

	fileList, err = u.getFileList(r)

	u.log.WithField("fileList", fileList).Info("file list")

	return
}

func (u *Unpacker) isValidPackage(r io.ReadSeeker) (bool, error) {
	magic := []byte{0x47, 0x44, 0x50, 0x43} // GDPC
	magicBuf := make([]byte, 4)

	_, err := r.Seek(0, io.SeekStart) // back to start
	if !checkErrorSeek(u.log, err) {
		return false, err
	}

	// find our magic to figure out where to start reading
	n, err := r.Read(magicBuf) // attempt to read the GDPC Magic from the start of the file
	if !checkErrorRead(u.log, err, n, 4) {
		return false, err
	}

	if bytes.Equal(magicBuf, magic) {
		u.log.Infof("%v looks like a pck archive", u.name)
		_, err = r.Seek(0, io.SeekStart) // back to start
		if !checkErrorSeek(u.log, err) {
			return false, err
		}
	} else {
		u.log.Infof("Failed to read GDPC pck Magic: read [%v] expected [%v]", magicBuf, magic)

		_, err = r.Seek(-4, io.SeekEnd) // 4 bytes from end
		if !checkErrorSeek(u.log, err) {
			return false, err
		}

		n, err = r.Read(magicBuf) // attempt to read the GDPC Magic from the end of the file
		if !checkErrorRead(u.log, err, n, 4) {
			return false, err
		}

		if !bytes.Equal(magicBuf, magic) {
			u.log.Infof("%v looks like a self-contained exe", u.name)

			_, err = r.Seek(-12, io.SeekEnd) // 12 bytes from end
			if !checkErrorSeek(u.log, err) {
				return false, err
			}

			var mainOffset int64
			err = binary.Read(r, binary.LittleEndian, &mainOffset)

			if err != nil {
				u.log.WithError(err).Error("Could not read main offset of data in self-contained exe")
				return false, err
			}

			var curPos int64
			curPos, err = r.Seek(0, io.SeekCurrent) // tell
			if !checkErrorSeek(u.log, err) {
				return false, err
			}

			_, err = r.Seek(curPos-mainOffset-8, io.SeekStart)

		} else {
			u.log.Errorf("Failed to read GDPC self-contained exe Magic: read [%v] expected [%v]", magicBuf, magic)
			return false, nil
		}
	}

	return true, nil
}

func (u *Unpacker) getFileList(r io.ReadSeeker) (fileList []structures.FileInfo, err error) {
	var headers structures.PackageHeadersBytes

	err = binary.Read(r, binary.LittleEndian, &headers)
	if err != nil {
		u.log.WithError(err).Error("Could not read package headers")
		return
	}

	u.log.Infof("%v info: %v", u.name, headers)

	fileCount := headers.FileCount

	for fileNum := uint32(1); fileNum <= fileCount; fileNum++ {
		var filePathLength int32
		err = binary.Read(r, binary.LittleEndian, &filePathLength)
		if err != nil {
			u.log.WithError(err).WithField("FileNum", fileNum).Errorf("Could not read file path length for file [%v]", fileNum)
			return
		}

		var infoBytes structures.FileInfoBytes
		pathBytes := make([]byte, filePathLength)
		err = binary.Read(r, binary.LittleEndian, &pathBytes)
		if err != nil {
			u.log.WithError(err).WithField("FileNum", fileNum).Errorf("Could not read file path for file [%v]", fileNum)
			return
		}

		err = binary.Read(r, binary.LittleEndian, &infoBytes)
		if err != nil {
			u.log.WithError(err).WithField("FileNum", fileNum).Errorf("Could not read file info for file [%v]", fileNum)
			return
		}

		info := structures.FileInfo{
			Path:   strings.Replace(string(pathBytes), "res://", u.name+"/", 1),
			Offset: int64(infoBytes.Offset),
			Size:   int64(infoBytes.Size),
			Md5:    hex.EncodeToString(infoBytes.Md5[:]),
		}

		u.log.
			//WithField("infoBytes", infoBytes).
			//WithField("FileNum", fileCount).
			//WithField("FileCount", fileNum).
			WithField("info", info).
			Infof("File [%v/%v]", fileNum, fileCount)
		fileList = append(fileList, info)

	}

	return
}
