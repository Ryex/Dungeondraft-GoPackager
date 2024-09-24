package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	humanize "github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	treeprint "github.com/xlab/treeprint"

	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
)

type ListCmd struct {
	InputPath string `arg:"" type:"path" help:"the .dungeondraft_pack file to unpack"`
}

func (ls *ListCmd) Run(ctx *Context) error {
	packFilePath, pathErr := filepath.Abs(ls.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for packfile"))
	}

	packFileName := filepath.Base(packFilePath)

	l := log.WithFields(log.Fields{
		"filename": packFileName,
	})

	pkg := ddpackage.NewPackage(l)

	file, err := pkg.LoadFromPackedPath(packFilePath)
	if err != nil {
		l.WithError(err).Error("failed to load package")
		return err
	}

	defer file.Close()

	tree := treeprint.New()
	branchMap := make(map[string]treeprint.Tree)

	var nodeForPath func(path string, fileInfo *structures.FileInfo) treeprint.Tree
	nodeForPath = func(path string, fileInfo *structures.FileInfo) treeprint.Tree {
		meta := fmt.Sprintf("%s -- %s", fileInfo.ResPath, humanize.Bytes(uint64(fileInfo.Size)))
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
	for i := 0; i < len(pkg.FileList); i++ {
		packedFile := &pkg.FileList[i]
		path := pkg.NormalizeResourcePath(packedFile.ResPath)
		l.WithField("res", packedFile.ResPath).
			WithField("size", packedFile.Size).
			WithField("index", i).
			Trace("building tree node")
		nodeForPath(path, packedFile)
	}

	fmt.Println(tree.String())

	return nil
}