package main

import (
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/ryex/dungeondraft-gopackager/internal/cmd"

	log "github.com/sirupsen/logrus"
)

var CLI struct {
	LogLevel string `enum:"debug,info,warn,error" default:"warn"`

	Pack     cmd.PackCmd   `cmd:"" help:"Packs the contents of a directory to a .dungeondraft_pack file, there must be a valid pack.json in the directory"`
	Unpack   cmd.UnpackCmd `cmd:"" help:"Extracts the contesnts of a .dungeondraft_pack file"`
	Generate cmd.GenCmd    `cmd:"" aliases:"gen" help:"Generate pack data and thumbtails"`
	List     cmd.ListCmd   `cmd:"" aliases:"ls" help:"list resources in a .dungeondraft_pack file"`
	Edit     cmd.EditCmd   `cmd:"" help:"Edit pack info, tags, and tag sets"`
}

func main() {
	ctx := &cmd.Context{}
	cliCtx := kong.Parse(&CLI,
		kong.Name("dungeondraft-packager-cli"),
		kong.Description("Pack, Unpack, Edit, and Prepare resources for .dungeondraft_pack files"),
		kong.UsageOnError(),
		kong.ConfigureHelp(
			kong.HelpOptions{
				Compact: false,
				Summary: true,
				Tree:    false,
			}),
		kong.Bind(ctx),
	)

	var logLevel log.Level
	switch CLI.LogLevel {
	case "debug":
		logLevel = log.DebugLevel
	case "info":
		logLevel = log.InfoLevel
	case "warn":
		logLevel = log.WarnLevel
	case "error":
		logLevel = log.ErrorLevel
	default:
		logLevel = log.InfoLevel
	}

	log.SetLevel(logLevel)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC822,
	})

	err := cliCtx.Run(ctx)
	cliCtx.FatalIfErrorf(err)
}
