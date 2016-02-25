package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/sirupsen/logrus"
	"github.com/xlab/closer"
)

var (
	logger = logrus.New()
)

const (
	storagePath = "/opt/splunkpump.db"
)

func main() {
	logger.Level = logrus.DebugLevel

	app := cli.NewApp()
	app.Name = "splunk-pump"
	app.Usage = "A docker log pump to splunk"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host, d",
			Usage:  "splunk host to push messages to",
			EnvVar: "SP_SPLUNK_HOST",
		},
	}

	app.Action = func(ctx *cli.Context) {
		host := ctx.GlobalString("host")
		if host == "" {
			cli.ShowAppHelp(ctx)
			return
		}

		logsPump := NewLogsPump(storagePath)

		closer.Bind(func() {
			logsPump.Shutdown()
			logger.Info("terminated")
		})

		closer.Checked(func() error {
			logsPump.RegisterAdapter(NewSplunkAdapter, host)
			return logsPump.Run()
		}, true)
	}

	app.Run(os.Args)
}
