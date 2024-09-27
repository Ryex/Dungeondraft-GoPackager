package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	log "github.com/sirupsen/logrus"
)

type Context struct {
	Pkg       *ddpackage.Package
	InputPath string
	Log       log.FieldLogger
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
	if utils.DirExists(ctx.InputPath) {
		err := ctx.Pkg.LoadUnpackedFromFolder(ctx.InputPath)
		if err != nil {
			ctx.Log.WithError(err).Error("failed to load package")
			return err
		}
	} else {
		err := ctx.Pkg.LoadFromPackedPath(ctx.InputPath)
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
