package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/sirupsen/logrus"
	"github.com/xlab/closer"
)

var (
	logger *logrus.Logger
)

const (
	storagePath = "/opt/splunkpump.db"
)

func init() {
	logger = logrus.New()
	logger.Level = logrus.DebugLevel
	logger.Out = os.Stdout
}

func main() {

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
			logger.Info("startup ---------------------------------------------")
			logsPump.RegisterAdapter(NewSplunkAdapter, host)
			return logsPump.Run()
		}, true)
	}

	app.Run(os.Args)
}
