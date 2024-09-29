package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func AssertTrue(condition bool, msg string) {
	if condition {
		return
	}
	logrus.Fatalf("assertion failure: %s", msg)
}

func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

// FileExists tests if a file  exists and is not a Directory
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if errors.Is(err,  os.ErrNotExist) {
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

// InSlice tests inclusion of a string in a slice
func InSlice[T comparable](a T, list []T) bool {
	for i := 0; i < len(list); i++ {
		if list[i] == a {
			return true
		}
	}
	return false
}

func SplitOne(s string, sep string) (string, string) {
	x := strings.SplitN(s, sep, 1)
	return x[0], x[1]
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

	rest := n % int64(alignment)
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

func TruncatePathHumanFriendly(path string, maxLen int) string {
	path = filepath.Clean(path)
	repeat := false
	depth := 1
	for len(path) > maxLen && !repeat {
		dir, file := filepath.Split(path)
		top := dir
		for i := 0; i < depth && top != ""; i++ {
			top, _ = filepath.Split(top[:max(len(top)-1, 0)])
		}
		depth += 1
		var next string
		if top != "" {
			next = filepath.Join(top, "...", file)
		} else {
			next = path
		}
		if next == path {
			repeat = true
		}
		path = next
	}
	return path
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func Filter[T any](ts []T, P func(T) bool) []T {
	ret := []T{}
	for _, t := range ts {
		if P(t) {
			ret = append(ret, t)
		}
	}
	return ret
}

func ListDir(path string) (files []string, dirs []string, errs []error) {
	paths := []string{path}

	for len(paths) > 0 {
		current := paths[0]
		paths = paths[1:]

		d, err := os.ReadDir(current)
		if err != nil {
			errs = append(errs, err)
		}
		for _, entry := range d {
			curPath := filepath.Join(current, entry.Name())
			if entry.IsDir() {
				paths = append(paths, curPath)
				dirs = append(dirs, curPath)
			} else {
				files = append(files, curPath)
			}
		}
	}
	return
}

func PathIsSub(parent string, sub string) (bool, error) {
	up := ".." + string(os.PathSeparator)
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(sub))
	if err != nil {
		return false, err
	}
	if !strings.HasPrefix(rel, up) && rel != ".." {
		return true, nil
	}
	return false, nil
}
