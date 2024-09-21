package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	humanize "github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	treeprint "github.com/xlab/treeprint"

	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/ryex/dungeondraft-gopackager/pkg/unpack"
)

type ListCmd struct {
	InputPath string `arg:"" type:"path" help:"the .dungeondraft_pack file to unpack"`

	IgnoreJson bool `short:"J" help:"ignore and do not extract json files"`
}

func (ls *ListCmd) Run(ctx *Context) error {
	packFilePath, pathErr := filepath.Abs(ls.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for packfile"))
	}

	packFileName := filepath.Base(packFilePath)
	packName := strings.TrimSuffix(packFileName, filepath.Ext(packFileName))

	l := log.WithFields(log.Fields{
		"filename": packFileName,
	})

	unpacker := unpack.NewUnpacker(l, packName)

	unpacker.IgnoreJson = ls.IgnoreJson

	file, fileErr := os.Open(packFilePath)
	if fileErr != nil {
		log.WithField("path", packFilePath).WithError(fileErr).Error("could not open file for reading.")
		return fileErr
	}

	defer file.Close()

	fileList, err := unpacker.ReadPackageFilelist(file)
	if err != nil {
		log.WithError(err).Error("failed to read file list")
		return err
	}
	tree := treeprint.New()
	branchMap := make(map[string]treeprint.Tree)

	var nodeForPath func(path string, fileInfo *structures.FileInfo) treeprint.Tree
	nodeForPath = func(path string, fileInfo *structures.FileInfo) treeprint.Tree {
		meta := fmt.Sprintf("%s -- %s", fileInfo.Path, humanize.Bytes(uint64(fileInfo.Size)))
		useMeta := true
		if strings.HasSuffix(path, string(filepath.Separator)) {
			path = path[:len(path)-1]
			useMeta = false
		}
		dir, file := filepath.Split(path)
		if dir == "" {
			if branchMap[file] == nil {
				if useMeta {
					branchMap[file] = tree.AddMetaBranch(meta, file)
				} else {
					branchMap[file] = tree.AddBranch(file)
				}
			}
			return branchMap[file]
		} else {
			if branchMap[path] == nil {
				parent := nodeForPath(dir, fileInfo)
				if useMeta {
					branchMap[path] = parent.AddMetaBranch(meta, file)
				} else {
					branchMap[path] = parent.AddBranch(file)
				}
			}
			return branchMap[path]
		}
	}
	for _, packedFile := range fileList {
		path := unpacker.NormalizeResourcePath(packedFile)
		nodeForPath(path, &packedFile)
	}

	fmt.Println(tree.String())

	return nil
}
