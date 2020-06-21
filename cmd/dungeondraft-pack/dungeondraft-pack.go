package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.com/ryexandrite/dungeondraft-gopackager/pkg/pack"
)

const usageText = `Desc:
	Packs the contesnts of a directory to a .dungeondraft_pack file, there must be a valid pack.json in the direcotry
Usage:
	dungeondraft-pack [args] <input folder> <dest folder>
Arguments:
`

func main() {
	flag.Usage = usage
	// args go here

	debugPtr := flag.Bool("debug", false, "output debug info level log messages?")
	flag.BoolVar(debugPtr, "v", false, "alias of -debug")

	overwritePtr := flag.Bool("overwrite", false, "overwrite output files at dest")
	flag.BoolVar(overwritePtr, "O", false, "alias of -overwrite")

	flag.Parse()

	debug := *debugPtr
	overwrite := *overwritePtr

	var inDir, outDir string
	if flag.NArg() < 1 {
		fmt.Println("Error: Must provide a pack folder")
		usage()
	} else if flag.NArg() < 2 {
		// windows useing `\` as a path seperator is bad , go treats it as an excape of a `"`
		if runtime.GOOS == "windows" && strings.Index(flag.Arg(0), `"`) >= 0 {
			splits := strings.SplitAfterN(flag.Arg(0), `"`, 2)
			inDir = strings.TrimSpace(strings.Trim(splits[0], `"`))
			outDir = strings.TrimSpace(strings.Trim(splits[1], `"`))
			// "\""
		} else {
			fmt.Println("Error: Must provide a output folder")
			usage()
		}
	} else {
		inDir = flag.Arg(0)
		outDir = flag.Arg(1)
	}

	packDirPath, pathErr := filepath.Abs(inDir)
	if pathErr != nil {
		fmt.Println("could not get absolute path for pack folder", pathErr)
	}

	outDirPath, err := filepath.Abs(outDir)
	if err != nil {
		fmt.Println("could not get absolute path for dest folder", pathErr)
	}

	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.WarnLevel)
	if debug {
		log.SetLevel(log.InfoLevel)
	}

	l := log.WithFields(log.Fields{
		"path":           packDirPath,
		"outPackagePath": outDirPath,
	})

	packer, err := pack.NewPackerFromFolder(l, packDirPath)
	if err != nil {
		l.Fatal("could not build Packer")
	}

	packer.Overwrite = overwrite

	err = packer.BuildFileList()
	if err != nil {
		l.Fatal("could not build file list")
	}

	err = packer.PackPackage(outDirPath)
	if err != nil {
		l.Fatal("packing failure")
	}
}

func usage() {
	fmt.Print(usageText)
	flag.PrintDefaults()
	os.Exit(2)
}
