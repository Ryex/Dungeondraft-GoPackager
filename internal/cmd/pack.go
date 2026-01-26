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

	err := ctx.LoadPkg(packDirPath)
	if err != nil {
		l.WithError(err).Error("could not load unpacked Package")
		return err
	}
	defer ctx.Pkg.Close()

	if pc.Progress {
		total := int64(ctx.Pkg.FileList().Size())
		bar := progressbar.Default(total, "Packing ...")
		err = ctx.Pkg.PackPackageProgress(outDirPath, ddpackage.PackOptions{Overwrite: pc.Overwrite}, func(p float64) {
			bar.Set(int(p * float64(total)))
		})
	} else {
		err = ctx.Pkg.PackPackage(outDirPath, ddpackage.PackOptions{Overwrite: pc.Overwrite})
	}
	if err != nil {
		l.WithError(err).Error("packing failure")
		return err
	}
	return nil
}
