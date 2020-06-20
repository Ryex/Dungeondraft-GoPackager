package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

// DirExists tests if a Directyory exists
func DirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// RipTexture detects and pulls image data from texture bytes
func RipTexture(data []byte) (fileExt string, fileData []byte, err error) {

	// webp
	start := bytes.Index(data, []byte{0x52, 0x49, 0x46, 0x46})
	if start >= 0 {
		var size int32
		err = binary.Read(bytes.NewBuffer(data[start+4:start+8]), binary.LittleEndian, size)
		if err != nil {
			return
		}
		fileExt = ".webp"
		fileData = data[start : start+8+int(size)]
		return
	}

	// png
	start = bytes.Index(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	if start >= 0 {
		end := bytes.Index(data, []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82})
		if end < 0 {
			err = errors.New("found PNG start but could not find PNG end tag")
			return
		}
		fileExt = ".png"
		fileData = data[start : end+8]
		return
	}

	// jpg
	start = bytes.Index(data, []byte{0xFF, 0xD8, 0xFF})
	if start >= 0 {
		end := bytes.Index(data, []byte{0xFF, 0xD9})
		if end < 0 {
			err = errors.New("found JPG start but could not find JPG end tag")
			return
		}
		fileExt = ".jpg"
		fileData = data[start:end]
		return
	}

	err = errors.New("no valid image data found")
	return
}
