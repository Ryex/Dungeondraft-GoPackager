package cmd

import (
  "os"
	"errors"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/ryex/dungeondraft-gopackager/pkg/unpack"
)

type UnpackCmd struct {
	InputPath       string `arg:"" type:"path" help:"the .dungeondraft_pack file to unpack"`
	DestinationPath string `arg:"" type:"path" help:"the destination folder path to place the unpacked files"`

	Overwrite   bool `short:"O" help:"overwrite output files at destination"`
	RipTextures bool `short:"R" help:"convert .tex files in the package to normal image formats (probably never needed)" `
	IgnoreJson  bool `short:"J" help:"ignore and do not extract json files"`
}

func (uc *UnpackCmd) Run(ctx *Context) error {
	packFilePath, pathErr := filepath.Abs(uc.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for packfile"))
	}

	packFileName := filepath.Base(packFilePath)
	packName := strings.TrimSuffix(packFileName, filepath.Ext(packFileName))

	outDirPath, pathErr := filepath.Abs(uc.DestinationPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for dest folder"))
	}

	l := log.WithFields(log.Fields{
		"filename": packFileName,
		"outPath":  outDirPath,
	})

	unpacker := unpack.NewUnpacker(l, packName)

	unpacker.Overwrite = uc.Overwrite
	unpacker.RipTextures = uc.RipTextures
	unpacker.IgnoreJson = uc.IgnoreJson

	file, fileErr := os.Open(packFilePath)
	if fileErr != nil {
		log.WithField("path", packFilePath).WithError(fileErr).Error("could not open file for reading.")
		return fileErr
	}

	defer file.Close()

	err := unpacker.ExtractPackage(file, outDirPath)
	if err != nil {
		l.WithError(err).Error("failed to extract package")
		return err
	}
	return nil
}
