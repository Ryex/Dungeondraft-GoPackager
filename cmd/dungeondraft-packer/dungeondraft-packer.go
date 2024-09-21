package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/ryex/dungeondraft-gopackager/internal/cmd"

	log "github.com/sirupsen/logrus"
)

var CLI struct {
	LogLevel log.Level `short:"v" type:"counter" help:"log level, 0 = Error, 1 = Warn (-v), 2 = Info (-vv), 3 = Debug (-vvv), 4 = Trace (-vvvv)"`

	Pack     cmd.PackCmd   `cmd:"" help:"Packs the contents of a directory to a .dungeondraft_pack file, there must be a valid pack.json in the directory"`
	Unpack   cmd.UnpackCmd `cmd:"" help:"Extracts the contesnts of a .dungeondraft_pack file"`
	Generate cmd.GenCmd    `cmd:"" aliases:"gen" help:"Generate pack data and thumbtails"`
	List     cmd.ListCmd   `cmd:"" aliases:"ls" help:"list resources in a .dungeondraft_pack file"`
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("dungeondraft-packer"),
		kong.Description("Pack, Unpack, Edit, and Prepare resources for .dungeondrat_pack files"),
		kong.UsageOnError(),
		kong.ConfigureHelp(
			kong.HelpOptions{
				Compact: true,
				Summary: false,
			}),
		// vars
	)

	level := CLI.LogLevel + 2
	log.SetLevel(level)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

	err := ctx.Run(&cmd.Context{})
	ctx.FatalIfErrorf(err)
}
