package ddpackage

import (
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"
)

func GenPackID() string {
	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))

	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, 8)

	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

type NewPackageJsonOptions struct {
	Path          string
	Name          string
	Author        string
	Version       string
	Keywords      []string
	Allow3rdParty *bool
	ColorOverides structures.CustomColorOverrides
}

// NewPackerFromFolder builds a new Packer from a folder with a valid pack.json
func NewPackageJson(log logrus.FieldLogger, options NewPackageJsonOptions, overwrite bool) (err error) {
	folderPath, err := filepath.Abs(options.Path)
	if err != nil {
		return
	}

	dirExists := utils.DirExists(folderPath)
	if !dirExists {
		err = errors.New("directory does not exists")
		log.WithError(err).WithField("path", folderPath).Error("can't package a non existent folder")
		return
	}

	packJSONPath := filepath.Join(folderPath, `pack.json`)

	packExists := utils.FileExists(packJSONPath)
	if packExists {
		if !overwrite {
			err = errors.New("a pack.json already exists and overwrite is not enabled")
			log.WithError(err).WithField("path", folderPath).Error("a pack.json already exists")
			return
		} else {
			log.WithField("path", folderPath).Warn("Overwriting pack.json")
		}
	}

	if options.Name == "" {
		err = errors.New("name field can not be empty")
		log.WithError(err).Error("invalid pack info")
		return
	}

	if options.Version == "" {
		err = errors.New("version field can not be empty")
		log.WithError(err).Error("invalid pack info")
		return
	}

	pack := structures.PackageInfo{
		Name:           options.Name,
		ID:             GenPackID(),
		Author:         options.Author,
		Version:        options.Version,
		Keywords:       options.Keywords,
		KeywordsRaw:    strings.Join(options.Keywords, ","),
		Allow3rdParty:  options.Allow3rdParty,
		ColorOverrides: options.ColorOverides,
	}

	packJSONBytes, err := json.MarshalIndent(&pack, "", "\t")
	if err != nil {
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("can't create pack.json")
		return
	}

	err = os.WriteFile(packJSONPath, packJSONBytes, 0644)
	if err != nil {
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("can't write pack.json")
		return
	}
	return
}
