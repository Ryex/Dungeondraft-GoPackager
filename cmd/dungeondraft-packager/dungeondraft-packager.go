package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/ryex/dungeondraft-gopackager/internal/gui"
	"github.com/snowzach/rotatefilehook"

	log "github.com/sirupsen/logrus"
)

var CLI struct {
	LogLevel string ` enum:"debug,info,warn,error" default:"info"`
	LogFile  string `short:"L" type:"path" default:"./logs/packager.log"`
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
	f, err := os.OpenFile(CLI.LogFile, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening log file: %v", err)
	} else {
		defer f.Close()
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	}

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

	app := gui.NewApp()
	app.Main()
}
