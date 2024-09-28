package ddpackage

import (
	"encoding/json"
	"errors"
	"fmt"
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
	seededRand := rand.New(
		rand.NewSource(time.Now().UnixNano()))

	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, 8)

	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

type SavePackageJSONOptions struct {
	Path          string
	Name          string
	ID            *string
	Author        string
	Version       string
	Keywords      []string
	Allow3rdParty *bool
	ColorOverides structures.CustomColorOverrides
}

func SavePackageJSONOptionsFromPkg(pkg *Package) SavePackageJSONOptions {
	id := pkg.id
	allowthirdParty := *pkg.info.Allow3rdParty
	keywords := make([]string, len(pkg.info.Keywords))
	copy(keywords, pkg.info.Keywords)
	options := SavePackageJSONOptions{
		Path:          pkg.unpackedPath,
		Name:          pkg.name,
		ID:            &id,
		Author:        pkg.info.Author,
		Version:       pkg.info.Version,
		Keywords:      keywords,
		Allow3rdParty: &allowthirdParty,
		ColorOverides: pkg.info.ColorOverrides,
	}
	return options
}

// NewPackerFromFolder builds a new Packer from a folder with a valid pack.json
func SavePackageJSON(log logrus.FieldLogger, options SavePackageJSONOptions, overwrite bool) (err error) {
	folderPath, err := filepath.Abs(options.Path)
	if err != nil {
		return
	}

	if dirExists := utils.DirExists(folderPath); !dirExists {
		err = os.MkdirAll(folderPath, 0o777)
		if err != nil {
			return errors.Join(err, fmt.Errorf("failed to make directory %s", folderPath))
		}
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

	if options.ID == nil {
		id := GenPackID()
		options.ID = &id
	}

	pack := structures.PackageInfo{
		Name:           options.Name,
		ID:             *options.ID,
		Author:         options.Author,
		Version:        options.Version,
		Keywords:       options.Keywords,
		KeywordsRaw:    strings.Join(options.Keywords, ","),
		Allow3rdParty:  options.Allow3rdParty,
		ColorOverrides: options.ColorOverides,
	}

	packJSONBytes, err := json.MarshalIndent(&pack, "", "  ")
	if err != nil {
		log.WithError(err).
			WithField("path", folderPath).
			WithField("packJSONPath", packJSONPath).
			Error("can't create pack.json")
		return
	}

	err = os.WriteFile(packJSONPath, packJSONBytes, 0o644)
	if err != nil {
		log.WithError(err).WithField("path", folderPath).WithField("packJSONPath", packJSONPath).Error("can't write pack.json")
		return
	}
	return
}
