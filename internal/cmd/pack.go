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

	err = pkg.BuildFileList()
	if err != nil {
		l.WithError(err).Error("could not build file list")
		return err
	}

	bar := progressbar.Default(100, "Packing ...")
	err = pkg.PackPackage(outDirPath, ddpackage.PackOptions{Overwrite: pc.Overwrite}, func(p float64) {
		bar.Set(int(p * 100))
	})
	if err != nil {
		l.WithError(err).Error("packing failure")
		return err
	}
	return nil
}
