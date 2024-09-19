package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/pkg/pack"
	log "github.com/sirupsen/logrus"
)

const usageText = `Desc:
	Packs the contents of a directory to a .dungeondraft_pack file, there must be a valid pack.json in the directory
Usage:
	dungeondraft-pack [args] <input folder> <dest folder>

	By default this program requires a pack.json to exist, it must either be created by dungeondraft or this program.

	To create the pack.json use the -name (-N), -author (-A), and -version (-V) to set it's fields.
	a new pack ID will be generated every time these options are passed.

	if a pack.json already exists this will fail unless you pass -editpack (-E).
	all values of the existing pack.json will be overwrites, including the author if it is left blank.

	If you only wish to generate the pack.json and not package the folder pass pass the -genpack (-G) flag.
	<dest folder> becomes optional and will be ignored in this case

	- if a package name, author, or version are specified; then package name and version can not be blank
	- passing in the name, author, and version will *ALWAYS* generate a new ID
	- by default the pack.json will not be over writted, pass -E | -editpack to do so
	- by default the pack in the dest folder will not be over written, pass -O | -overwrite to do so

Arguments:
`

func main() {
	flag.Usage = usage
	// args go here

	debugPtr := flag.Bool("debug", false, "output Debug level log messages?")
	flag.BoolVar(debugPtr, "vv", false, "alias of -debug")

	infoPtr := flag.Bool("info", false, "output Info level log messages?")
	flag.BoolVar(infoPtr, "v", false, "alias of -info")

	overwritePtr := flag.Bool("overwrite", false, "overwrite output files at dest")
	flag.BoolVar(overwritePtr, "O", false, "alias of -overwrite")

	packNamePtr := flag.String("name", "", "pack the package with the given name")
	flag.StringVar(packNamePtr, "N", "", "alias of -name")

	packAuthorPtr := flag.String("author", "", "pack the package with the given author")
	flag.StringVar(packAuthorPtr, "A", "", "alias of -author")

	packVersionPtr := flag.String("version", "", "pack the package with the given version")
	flag.StringVar(packVersionPtr, "V", "", "alias of -version")

	packEditPtr := flag.Bool("editpack", false, "overwrite the pack.json with the passed values")
	flag.BoolVar(packEditPtr, "E", false, "alias of -editpack")

	packGenPtr := flag.Bool("genpack", false, "write the pack.json and exit")
	flag.BoolVar(packGenPtr, "G", false, "alias of -genpack")

	flag.Parse()

	debug := *debugPtr
	info := *infoPtr
	overwrite := *overwritePtr

	packName := *packNamePtr
	packAuthor := *packAuthorPtr
	packVersion := *packVersionPtr
	packEdit := *packEditPtr

	packGen := *packGenPtr

	var inDir, outDir string
	if flag.NArg() < 1 {
		fmt.Println("Error: Must provide a pack folder")
		usage()
	} else if flag.NArg() < 2 {
		// windows using `\` as a path separator is bad , go treats it as an escape of a `"`
		if runtime.GOOS == "windows" && strings.Index(flag.Arg(0), `"`) >= 0 {
			splits := strings.SplitAfterN(flag.Arg(0), `"`, 2)
			inDir = strings.TrimSpace(strings.Trim(splits[0], `"`))
			outDir = strings.TrimSpace(strings.Trim(splits[1], `"`))
			// "\""
		} else if packGen {
			inDir = flag.Arg(0)
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
	if err != nil && !packGen {
		fmt.Println("could not get absolute path for dest folder", pathErr)
	}

	log.SetOutput(os.Stdout)

	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

	// Only log the warning severity or above.
	log.SetLevel(log.WarnLevel)
	if debug {
		log.SetLevel(log.DebugLevel)
	} else if info {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	l := log.WithFields(log.Fields{
		"path":           packDirPath,
		"outPackagePath": outDirPath,
	})
	var packer *pack.Packer
	if packName != "" || packAuthor != "" || packVersion != "" || packGen {
		if packName == "" || packVersion == "" {
			l.Fatal("if a package name, author, or version are specified, or genpack is set; then package name and version can not be blank ")
		}
		packer, err = pack.NewPackerFolder(l, packDirPath, packName, packAuthor, packVersion, packEdit)
		if packGen {
			l.Info("pack.json created")
			os.Exit(0)
		}
	} else {
		packer, err = pack.NewPackerFromFolder(l, packDirPath)
	}
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
