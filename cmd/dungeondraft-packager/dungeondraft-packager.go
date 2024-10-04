package main

import (
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/ryex/dungeondraft-gopackager/internal/gui"
	"github.com/snowzach/rotatefilehook"

	log "github.com/sirupsen/logrus"
)

var CLI struct {
	LogLevel string `enum:"debug,info,warn,error" default:"warn"`
	LogFile  string `short:"L" type:"path" default:"./logs/packager.log"`
}

func main() {
	kong.Parse(&CLI,
		kong.Configuration(kong.JSON, "./dd-gopackager.json", "~/.config/dd-gopackager.json"),
		kong.Name("dungeondraft-packager"),
		kong.Description(
			"Pack, Unpack, Edit, and Prepare resources for .dungeondraft_pack files\n\n"+
				"log file and level can also be configured from a file.\n"+
				"the first of the folowing will be loaded:\n"+
				"\t./dd-gopackager.json\n"+
				"\t~/.config/dd-gopackager.json",
		),
		kong.UsageOnError(),
		kong.ConfigureHelp(
			kong.HelpOptions{
				Compact: true,
				Summary: true,
				Tree:    true,
			}),
		// vars
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

	rotateFileHook, err := rotatefilehook.NewRotateFileHook(rotatefilehook.RotateFileConfig{
		Filename:   CLI.LogFile,
		MaxSize:    50, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Level:      logLevel,
		Formatter: &log.JSONFormatter{
			TimestampFormat: time.RFC822,
		},
	})
	if err != nil {
		log.WithError(err).Error("Failed to init log file rotate hook")
	} else {
		log.AddHook(rotateFileHook)
	}
	log.Infof("Log level %s", logLevel.String())
	log.Infof("Logging to %s", CLI.LogFile)

	app := gui.NewApp()
	app.Main()
}
