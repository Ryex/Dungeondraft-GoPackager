package cmd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

type UnpackCmd struct {
	InputPath       string `arg:"" type:"path" help:"the .dungeondraft_pack file to unpack"`
	DestinationPath string `arg:"" type:"path" help:"the destination folder path to place the unpacked files"`

	Overwrite   bool `short:"O" help:"overwrite output files at destination"`
	RipTextures bool `short:"R" help:"convert .tex files in the package to normal image formats (probably never needed)" `
	Thumbnails  bool `short:"T" help:"don't ignore resource thumbnails"`
	Progress    bool `default:"true" negatable:"" help:"show progressbar"`
}

func (uc *UnpackCmd) Run(ctx *Context) error {
	packFilePath, pathErr := filepath.Abs(uc.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for packfile"))
	}

	packFileName := filepath.Base(packFilePath)

	outDirPath, pathErr := filepath.Abs(uc.DestinationPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for dest folder"))
	}

	l := log.WithFields(log.Fields{
		"filename": packFileName,
		"outPath":  outDirPath,
	})

	pkg := ddpackage.NewPackage(l)

	file, fileErr := os.Open(packFilePath)
	if fileErr != nil {
		log.WithField("path", packFilePath).WithError(fileErr).Error("could not open file for reading.")
		return fileErr
	}

	defer file.Close()

	options := ddpackage.UnpackOptions{
		Overwrite:   uc.Overwrite,
		RipTextures: uc.RipTextures,
		Thumbnails:  uc.Thumbnails,
	}
	var err error
	if uc.Progress {
		total := int64(len(pkg.FileList()))
		bar := progressbar.Default(total, "Unpacking ...")
		err = pkg.ExtractPackageProgress(outDirPath, options, func(p float64) {
			bar.Set(int(p * float64(total)))
		})
	} else {
		err = pkg.ExtractPackage(outDirPath, options)
	}
	if err != nil {
		l.WithError(err).Error("failed to extract package")
		return err
	}
	return nil
}
