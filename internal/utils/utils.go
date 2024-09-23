package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

func AssertTrue(condition bool, msg string) {
	if condition {
		return
	}
	logrus.Fatalf("assertion failure: %s", msg)
}

// FileExists tests if a file  exists and is not a Directory
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists tests if a Directyory exists and is a Directory
func DirExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
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

// StringInSlice tests inclusion of a string in a slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// CheckErrorRead checks and logs a read error
func CheckErrorRead(log logrus.FieldLogger, err error, n int, expected int) bool {
	if err != nil {
		log.WithError(err).Error("read error")
		return false
	} else if n < expected {
		log.WithField("readBytes", n).
			WithField("expectedBytes", expected).
			Error("wrong number of bytes read")
		return false
	}
	return true
}

// CheckErrorWrite checks and logs a read error
func CheckErrorWrite(log logrus.FieldLogger, err error) bool {
	if err != nil {
		log.WithError(err).Error("write error")
		return false
	}
	return true
}

// CheckErrorSeek checks and looks a seek error
func CheckErrorSeek(log logrus.FieldLogger, err error) bool {
	if err != nil {
		log.WithError(err).Error("seek failure")
		return false
	}
	return true
}

func Tell(r io.Seeker) (int64, error) {
	curPos, err := r.Seek(0, io.SeekCurrent) // tell
	return curPos, err
}

func Align(n int64, alignment int) int64 {
	if alignment == 0 {
		return n
	}

	var rest int64 = n % int64(alignment)
	if rest == 0 {
		return n
	} else {
		return n + (int64(alignment) - rest)
	}
}

func Pad(out io.Writer, bytes int64) error {
	for i := int64(0); i < bytes; i++ {
		var b byte = 0
		err := binary.Write(out, binary.LittleEndian, b)
		if err != nil {
			return err
		}
	}
	return nil
}
