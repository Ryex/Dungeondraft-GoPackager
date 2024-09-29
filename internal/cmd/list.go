package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	treeprint "github.com/xlab/treeprint"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

type ListCmd struct {
	Files ListFilesCmd `cmd:"" help:"lists files in the pack"`
	Tags  ListTagsCmd  `cmd:"" help:"lists all tags that match the provided resource patterns, with no patterns lists all tags"`
}

type ListFilesCmd struct {
	All        bool   `short:"A" help:"List all file in the package, overrides the individual type options"`
	Textures   bool   `short:"X" default:"true" negatable:"" help:"list texture files. default is true but negatable with --no-textures"`
	Thumbnails bool   `short:"T" default:"false" negatable:"" help:"list thumbnail files"`
	Data       bool   `short:"D" default:"false" negatable:"" help:"list Data files (tags, and wall/terrain metadata )"`
	InputPath  string `arg:"" type:"path" help:"the .dungeondraft_pack file or resource directory to work with"`
}

func (lsf *ListFilesCmd) Run(ctx *Context) error {
	err := ctx.LoadPkg(lsf.InputPath)
	if err != nil {
		return err
	}
	lsf.printTree(ctx.Log, ctx.Pkg)
	return nil
}

func (lsf *ListFilesCmd) printTree(l logrus.FieldLogger, pkg *ddpackage.Package) {
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
	fileList := pkg.FileList()
	for i, fi := range fileList {
		if fi.IsMetadata() && !lsf.All {
			continue
		}
		if fi.IsTexture() && (!lsf.Textures && !lsf.All) {
			continue
		}
		if fi.IsThumbnail() && (!lsf.Thumbnails && !lsf.All) {
			continue
		}
		if fi.IsData() && (!lsf.Data && !lsf.All) {
			continue
		}
		path := pkg.NormalizeResourcePath(fi.ResPath)
		l.WithField("res", fi.ResPath).
			WithField("size", fi.Size).
			WithField("index", i).
			Trace("building tree node")
		nodeForPath(path, fi)
	}

	fmt.Println(tree.String())
}

type ListTagsCmd struct {
	InputPath    string   `arg:"" type:"path" help:"the .dungeondraft_pack file or resource directory to work with"`
	GlobPatterns []string `arg:"" optional:"" help:"glob patterns to match against paths relative to package root (paths should not stor with a dot (./) and must use slash separation even on windows (a/b))"`
}

func (lst *ListTagsCmd) Run(ctx *Context) error {
	err := ctx.LoadPkg(lst.InputPath)
	if err != nil {
		return err
	}
	err = ctx.LoadTags()
	if err != nil {
		return err
	}
	return lst.printTags(ctx.Log, ctx.Pkg)
}

func (ls *ListTagsCmd) printTags(l logrus.FieldLogger, pkg *ddpackage.Package) error {
	var tags []string
	if len(ls.GlobPatterns) < 1 {
		tags = pkg.Tags().AllTags()
	} else {
		files, err := pkg.FileList().Glob(func(fi *structures.FileInfo) bool { return fi.IsTexture() }, ls.GlobPatterns...)
		if err != nil {
			l.WithError(err).Error("failed to glob file list")
			return err
		}
		tags = pkg.Tags().TagsFor(files.RelPaths()...).AsSlice()
	}
	for _, tag := range tags {
		fmt.Fprintln(os.Stdout, tag)
	}
	return nil
}
