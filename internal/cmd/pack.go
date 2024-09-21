package cmd
import (
	"errors"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/ryex/dungeondraft-gopackager/pkg/pack"
)

type PackCmd struct {
	InputPath       string `arg:"" type:"path" help:"the package folder path"`
	DestinationPath string `arg:"" type:"path" help:"the destination folder path to place the packaged .dungeondraft_pack"`

	Overwrite bool `short:"O" help:"overwrite output files at destination"`
}

func (p *PackCmd) Run(ctx *Context) error {
	packDirPath, pathErr := filepath.Abs(p.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for pack folder"))
	}

	outDirPath, pathErr := filepath.Abs(p.DestinationPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for dest folder"))
	}

	l := log.WithFields(log.Fields{
		"path":           packDirPath,
		"outPackagePath": outDirPath,
	})

	packer, err := pack.NewPackerFromFolder(l, packDirPath)
	if err != nil {
		l.WithError(err).Error("could not build Packer")
		return err
	}

	packer.Overwrite = p.Overwrite

	err = packer.BuildFileList()
	if err != nil {
		l.WithError(err).Error("could not build file list")
		return err
	}

	err = packer.PackPackage(outDirPath)
	if err != nil {
		l.WithError(err).Error("packing failure")
		return err
	}
	return nil
}
