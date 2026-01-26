// Package cmd
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
)

type Context struct {
	Pkg       *ddpackage.Package
	InputPath string
	Log       *log.Entry
}

func (ctx *Context) LoadPkg(path string) error {
	packPath, pathErr := filepath.Abs(path)
	if pathErr != nil {
		return errors.Join(pathErr, fmt.Errorf("could not get absolute path for %s", path))
	}
	ctx.InputPath = packPath
	log.Info("using input path ", ctx.InputPath)
	ctx.Log = log.WithFields(log.Fields{
		"inputPath": ctx.InputPath,
	})

	ctx.Pkg = ddpackage.NewPackage(ctx.Log)

	bar := progressbar.NewOptions64(
		-1,
		progressbar.OptionSetDescription("Loading ..."),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(10),
		progressbar.OptionShowTotalBytes(true),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetMaxDetailRow(1),
	)
	if utils.DirExists(ctx.InputPath) {
		err := ctx.Pkg.LoadUnpackedFromFolder(ctx.InputPath)
		if err != nil {
			ctx.Log.WithError(err).Error("failed to load package")
			return err
		}
		errs := ctx.Pkg.BuildFileListProgress(func(p float64, curPath string, max int64) {
			bar.ChangeMax64(max)
			bar.Set64(int64(p * float64(max)))
			bar.AddDetail(utils.TruncatePathHumanFriendly(curPath, 60))
		})
		bar.Finish()
		if len(errs) != 0 {
			for _, err := range errs {
				ctx.Log.WithField("task", "building file list").Errorf("error : %s", err.Error())
			}
			return errors.Join(errs...)
		}
	} else {
		err := ctx.Pkg.LoadFromPackedPath(ctx.InputPath, func(p float64, curRes string, max int64) {
			bar.ChangeMax64(max)
			bar.Set64(int64(p * float64(max)))
			bar.AddDetail(utils.TruncatePathHumanFriendly(curRes, 60))
		})
		bar.Finish()
		if err != nil {
			ctx.Log.WithError(err).Error("failed to load package")
			return err
		}
	}
	return nil
}

func (ctx *Context) LoadTags() error {
	err := ctx.Pkg.LoadTags()
	if err != nil {
		ctx.Log.WithError(err).Error("failed to load tags")
		return err
	}
	return nil
}


func (ctx *Context) LoadMetadata() error {
	err := ctx.Pkg.LoadResourceMetadata()
	if err != nil {
		ctx.Log.WithError(err).Error("failed to load metadata")
		return err
	}
	return nil
}
