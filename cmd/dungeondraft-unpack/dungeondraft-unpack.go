package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.com/ryexandrite/dungeondraft-gopackager/pkg/unpack"
)

const usageText = `Extracts the contesnts of a .dungeondraft_pack file
Usage:
	dungeondraft-unpack [args] <.dungeondraft_pack file> <output folder>
Arguments:
`

func main() {
	flag.Usage = usage
	// args go here

	debugPtr := flag.Bool("debug", false, "output debug info level log messages?")
	flag.BoolVar(debugPtr, "v", false, "alias of -debug")

	flag.Parse()

	debug := *debugPtr

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

	// Only log the warning severity or above.
	log.SetLevel(log.WarnLevel)
	if debug {
		log.SetLevel(log.InfoLevel)
	}

	logger := log.WithFields(log.Fields{
		"filename": packFileName,
	})

	unpacker := unpack.NewUnpacker(logger, packName)

	file, fileErr := os.Open(packFilePath)
	if fileErr != nil {
		log.WithField("path", packFilePath).WithError(fileErr).Fatal("Could not open file for reading.")
	}

	defer file.Close()

	unpacker.ExtractPackage(file, flag.Arg(1))
}

func usage() {
	fmt.Print(usageText)
	flag.PrintDefaults()
	os.Exit(2)
}
