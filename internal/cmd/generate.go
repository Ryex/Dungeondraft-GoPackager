package cmd

import (
	"errors"
	"path/filepath"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
)

type GenCmd struct {
	Pack       GenPackCmd `cmd:"" help:"Create a pack.json and populate it"`
	Thumbnails GenTumbCmd `cmd:"" aliases:"thumb" help:"Generate or regenerate thumbnails for the eventual packed resources"`
}

type GenPackCmd struct {
	InputPath string `arg:"" type:"existingdir" help:"the package folder path"`
	Overwrite bool   `short:"O" help:"overwrite existing pack.json"`

	ID      string `short:"I" help:"Unique ID for the pack, defaults to a randomly generated id"`
	Name    string `short:"N" help:"name of the package" required:""`
	Author  string `short:"A" help:"package author" required:""`
	Version string `short:"V" help:"package version"`

	AllowThirdParty *bool `short:"M" help:" set the 'allow_3rd_party_mapping_software_to_read' key. package will be incompatible with Dungeondraft v1.0.3.2" default:"true"`

	Keywords []string `short:"K" help:"comma separated keywords"`

	MinRedness    *float64 `short:"R" help:"enable custom colors and set the minimum redness value" default:"0.1"`
	MinSaturation *float64 `short:"S" help:"enable custom colors and set the minimum saturation value" default:"0"`
	RedTolerance  *float64 `short:"T" help:"enable custom colors and set the red tolerance value" default:"0.04"`
}

type GenTumbCmd struct {
	InputPath string `arg:"" type:"path" help:"the package folder path"`

	Progress bool `default:"true" negatable:"" help:"show progressbar"`
}

func (gpc *GenPackCmd) Run(ctx *Context) error {
	packDirPath, pathErr := filepath.Abs(gpc.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for pack folder"))
	}

	l := log.WithFields(log.Fields{
		"path": packDirPath,
	})

	l.Trace("Generateing pack.json")

	err := ddpackage.SavePackageJSON(l, ddpackage.SavePackageJSONOptions{
		Path:          packDirPath,
		ID:            gpc.ID,
		Name:          gpc.Name,
		Author:        gpc.Author,
		Version:       gpc.Version,
		Allow3rdParty: gpc.AllowThirdParty,
		Keywords:      gpc.Keywords,
		ColorOverides: structures.CustomColorOverrides{
			Enabled: gpc.MinRedness != nil || gpc.MinSaturation != nil || gpc.RedTolerance != nil,
			MinRedness: func() float64 {
				if gpc.MinRedness != nil {
					return *gpc.MinRedness
				} else {
					return 0.1
				}
			}(),
			MinSaturation: func() float64 {
				if gpc.MinSaturation != nil {
					return *gpc.MinSaturation
				} else {
					return 0
				}
			}(),
			RedTolerance: func() float64 {
				if gpc.RedTolerance != nil {
					return *gpc.RedTolerance
				} else {
					return 0.04
				}
			}(),
		},
	}, gpc.Overwrite)
	if err != nil {
		return errors.Join(err, errors.New("failed to generate pack.json"))
	}
	return nil
}

func (gtc *GenTumbCmd) Run(ctx *Context) error {
	log.Trace("Generating thumbnails")

	packDirPath, pathErr := filepath.Abs(gtc.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for pack folder"))
	}

	l := log.WithFields(log.Fields{
		"path": packDirPath,
	})

	pkg := ddpackage.NewPackage(l)

	err := pkg.LoadUnpackedFromFolder(packDirPath)
	if err != nil {
		l.WithError(err).Error("could not build Package")
		return err
	}

	var errs []error
	errs = pkg.BuildFileList()
	if len(errs) != 0 {
		for _, err := range errs {
			l.WithField("task", "build file list").Errorf("error: %s", err.Error())
		}
		return errors.New("Failed to build file list")
	}

	if gtc.Progress {
		total := int64(len(pkg.FileList().Filter(func(info *structures.FileInfo) bool {
			return info.IsTexture()
		})))
		bar := progressbar.Default(total, "Generating Thumbnails ...")
		errs = pkg.GenerateThumbnailsProgress(func(p float64) {
			bar.Set(int(p * float64(total)))
		})
	} else {
		errs = pkg.GenerateThumbnails()
	}
	if len(errs) != 0 {
		l.Error("error generating thumbnails")
		return errors.Join(errs...)
	}

	return nil
}
