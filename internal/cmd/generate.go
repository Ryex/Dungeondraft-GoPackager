package cmd

import (
	"errors"
	"path/filepath"

	log "github.com/sirupsen/logrus"

)

type GenCmd struct {
	Pack       GenPackCmd `cmd:"" help:"Create a pack.json and populate it"`
	Thumbnails GenTumbCmd `cmd:"" aliases:"thumb" help:"Generate or regenerate thumbnails for the eventual packed resources"`
}

type GenPackCmd struct {
	InputPath string `arg:"" type:"existingdir" help:"the package folder path"`
	Overwrite bool   `short:"O" help:"overwrite existing pack.json"`

	Name   string `short:"N" help:"name of the package" required:""`
	Author string `short:"A" help:"package version" required:""`

	AllowThirdParty *bool `short:"M" help:" set the 'allow_3rd_party_mapping_software_to_read' key. package will be incompatible with Dungeondraft v1.0.3.2" default:"true"`

	Keywords []string `short:"K" help:"comma separated keywords"`

	MinRedness    *float64 `short:"R" help:"enable custom colors and set the minimum redness value" default:"0.1"`
	MinSaturation *float64 `short:"S" help:"enable custom colors and set the minimum saturation value" default:"0"`
	RedTolerance  *float64 `short:"T" help:"enable custom colors and set the red tolerance value" default:"0.04"`
}

type GenTumbCmd struct {
	InputPath string `arg:"" type:"path" help:"the package folder path"`
	Overwrite bool   `short:"O" help:"overwrite output files at destination"`
}

func (cmd *GenPackCmd) Run(ctx *Context) error {

	packDirPath, pathErr := filepath.Abs(cmd.InputPath)
	if pathErr != nil {
		return errors.Join(pathErr, errors.New("could not get absolute path for pack folder"))
	}

	l := log.WithFields(log.Fields{
		"path": packDirPath,
	})

	l.Trace("Generateing pack.json", cmd)

	return nil
}

func (cmd *GenTumbCmd) Run(ctx *Context) error {
	log.Trace("Generating thumbnails")
	return nil
}
