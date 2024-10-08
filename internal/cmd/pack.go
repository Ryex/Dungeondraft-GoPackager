package cmd

import (
	"errors"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

type PackCmd struct {
	InputPath       string `arg:"" type:"path" help:"the package folder path"`
	DestinationPath string `arg:"" type:"path" help:"the destination folder path to place the packaged .dungeondraft_pack"`

	Overwrite  bool `short:"O" help:"overwrite output files at destination"`
	Thumbnails bool `short:"T" help:"generate thumbnails"`
	Progress   bool `default:"true" negatable:"" help:"show progressbar"`
}

func (pc *PackCmd) Run(ctx *Context) error {
	packDirPath, pathErr := filepath.Abs(pc.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for pack folder"))
	}

	outDirPath, pathErr := filepath.Abs(pc.DestinationPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for dest folder"))
	}

	l := log.WithFields(log.Fields{
		"path":           packDirPath,
		"outPackagePath": outDirPath,
	})

	pkg := ddpackage.NewPackage(l)

	err := pkg.LoadUnpackedFromFolder(packDirPath)
	if err != nil {
		l.WithError(err).Error("could not load unpacked Package")
		return err
	}

	errs := pkg.BuildFileList()
	if len(errs) != 0 {
		for _, err := range errs {
			l.WithField("task", "build file list").Errorf("err: %s", err.Error())
		}
		return errors.New("Failed to build file list")
	}

	if pc.Progress {
		total := int64(len(pkg.FileList()))
		bar := progressbar.Default(total, "Packing ...")
		err = pkg.PackPackageProgress(outDirPath, ddpackage.PackOptions{Overwrite: pc.Overwrite}, func(p float64) {
			bar.Set(int(p * float64(total)))
		})
	} else {
		err = pkg.PackPackage(outDirPath, ddpackage.PackOptions{Overwrite: pc.Overwrite})
	}
	if err != nil {
		l.WithError(err).Error("packing failure")
		return err
	}
	return nil
}
