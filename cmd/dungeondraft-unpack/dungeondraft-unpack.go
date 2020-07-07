package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/pkg/unpack"
	log "github.com/sirupsen/logrus"
)

const usageText = `Desc:
	Extracts the contesnts of a .dungeondraft_pack file
Usage:
	dungeondraft-unpack [args] <.dungeondraft_pack file> <dest folder>
Arguments:
`

func main() {
	flag.Usage = usage
	// args go here

	debugPtr := flag.Bool("debug", false, "output debug info level log messages?")
	flag.BoolVar(debugPtr, "v", false, "alias of -debug")

	overwritePtr := flag.Bool("overwrite", false, "overwrite outputfiles at dest")
	flag.BoolVar(overwritePtr, "O", false, "alias of -overwrite")

	ripPtr := flag.Bool("riptex", false, "convert .tex files int he package to normal image formats (probably never needed)")
	flag.BoolVar(ripPtr, "R", false, "alias of -riptex")

	flag.Parse()

	debug := *debugPtr
	overwrite := *overwritePtr
	ripTex := *ripPtr

	if flag.NArg() < 1 {
		fmt.Println("Error: Must provide a pack file")
		usage()
	} else if flag.NArg() < 2 {
		fmt.Println("Error: Must provide a output folder")
		usage()
	}

	packFilePath, pathErr := filepath.Abs(flag.Arg(0))
	if pathErr != nil {
		fmt.Println("could not get absolute path for packfile", pathErr)
	}

	packFileName := filepath.Base(packFilePath)
	packName := strings.TrimSuffix(packFileName, filepath.Ext(packFileName))

	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

	// Only log the warning severity or above.
	log.SetLevel(log.WarnLevel)
	if debug {
		log.SetLevel(log.InfoLevel)
	}

	outDirPath, err := filepath.Abs(flag.Arg(1))
	if err != nil {
		return
	}

	logger := log.WithFields(log.Fields{
		"filename": packFileName,
		"outPath":  outDirPath,
	})

	unpacker := unpack.NewUnpacker(logger, packName)

	unpacker.Overwrite = overwrite
	unpacker.RipTextures = ripTex

	file, fileErr := os.Open(packFilePath)
	if fileErr != nil {
		log.WithField("path", packFilePath).WithError(fileErr).Fatal("could not open file for reading.")
	}

	defer file.Close()

	err = unpacker.ExtractPackage(file, outDirPath)
	if err != nil {
		logger.WithError(err).Fatal("failed to extract package")
	}
}

func usage() {
	fmt.Print(usageText)
	flag.PrintDefaults()
	os.Exit(2)
}
