package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/ryex/dungeondraft-gopackager/internal/gui"

	log "github.com/sirupsen/logrus"
)

var CLI struct {
	LogLevel log.Level `short:"v" type:"counter" help:"log level, 0 = Error, 1 = Warn (-v), 2 = Info (-vv), 3 = Debug (-vvv), 4 = Trace (-vvvv)"`
}

func main() {
	kong.Parse(&CLI,
		kong.Name("dungeondraft-packager"),
		kong.Description("Pack, Unpack, Edit, and Prepare resources for .dungeondraft_pack files"),
		kong.UsageOnError(),
		kong.ConfigureHelp(
			kong.HelpOptions{
				Compact: true,
				Summary: true,
			}),
		// vars
	)

	level := CLI.LogLevel + 2
	log.SetLevel(level)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

	app := gui.NewApp()
	app.Main()
}
