package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	treeprint "github.com/xlab/treeprint"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

type ListCmd struct {
	Files ListFilesCmd `cmd:""`
	Tags  ListTagsCmd  `cmd:""`
}

type ListFilesCmd struct {
	InputPath string `arg:"" type:"path" help:"the .dungeondraft_pack file or resource directory to work with"`
}

func (ls *ListFilesCmd) Run(ctx *Context) error {
	packPath, pathErr := filepath.Abs(ls.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for packfile"))
	}

	if utils.DirExists(packPath) {
		return ls.loadUnpacked(packPath)
	}
	return ls.loadPacked(packPath)
}

func (ls *ListFilesCmd) loadPacked(path string) error {
	l := log.WithFields(log.Fields{
		"filename": path,
	})
	pkg := ddpackage.NewPackage(l)

	file, err := pkg.LoadFromPackedPath(path)
	if err != nil {
		l.WithError(err).Error("failed to load package")
		return err
	}

	defer file.Close()

	ls.printTree(l, pkg)
	return nil
}

func (ls *ListFilesCmd) loadUnpacked(path string) error {
	l := log.WithFields(log.Fields{
		"filename": path,
	})
	pkg := ddpackage.NewPackage(l)

	err := pkg.LoadUnpackedFromFolder(path)
	if err != nil {
		l.WithError(err).Error("failed to load package")
		return err
	}

	ls.printTree(l, pkg)
	return nil
}

func (ls *ListFilesCmd) printTree(l logrus.FieldLogger, pkg *ddpackage.Package) {
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
}

type ListTagsCmd struct {
	InputPath string `arg:"" type:"path" help:"the .dungeondraft_pack file or resource directory to work with"`
}

func (ls *ListTagsCmd) Run(ctx *Context) error {
	packPath, pathErr := filepath.Abs(ls.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for packfile"))
	}

	if utils.DirExists(packPath) {
		return ls.loadUnpacked(packPath)
	}
	return ls.loadPacked(packPath)
}

func (ls *ListTagsCmd) loadPacked(path string) error {
	l := log.WithFields(log.Fields{
		"filename": path,
	})
	pkg := ddpackage.NewPackage(l)

	file, err := pkg.LoadFromPackedPath(path)
	if err != nil {
		l.WithError(err).Error("failed to load package")
		return err
	}
	defer file.Close()
	err = pkg.LoadPackedTags(file)
	if err != nil {
		l.WithError(err).Error("failed to load tags")
		return err
	}

	ls.printTree(l, pkg)
	return nil
}

func (ls *ListTagsCmd) loadUnpacked(path string) error {
	l := log.WithFields(log.Fields{
		"filename": path,
	})
	pkg := ddpackage.NewPackage(l)

	err := pkg.LoadUnpackedFromFolder(path)
	if err != nil {
		l.WithError(err).Error("failed to load package")
		return err
	}
	err = pkg.ReadUnpackedTags()
	if err != nil {
		l.WithError(err).Error("failed to load tags")
		return err
	}

	ls.printTree(l, pkg)
	return nil
}

func (ls *ListTagsCmd) printTree(l logrus.FieldLogger, pkg *ddpackage.Package) {
	tree := treeprint.New()
	branchMap := make(map[string]treeprint.Tree)

	var nodeForPath func(path string, fileInfo *structures.FileInfo) treeprint.Tree
	nodeForPath = func(path string, fileInfo *structures.FileInfo) treeprint.Tree {
		var tags []string
		for tag := range pkg.Tags.Tags {
			set := pkg.Tags.Tags[tag]
			if set.Has(fileInfo.RelPath) {
				tags = append(tags, strings.Join([]string{"\"", tag, "\""}, ""))
			}
		}
		meta := strings.Join(tags, ", ")
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
		info := &pkg.FileList[i]
		if !info.IsTexture() {
			continue
		}
		path := pkg.NormalizeResourcePath(info.ResPath)
		l.WithField("res", info.ResPath).
			WithField("size", info.Size).
			WithField("index", i).
			Trace("building tree node")
		nodeForPath(path, info)
	}

	fmt.Println(tree.String())
}
