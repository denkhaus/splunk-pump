package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/codegangsta/cli"
)

var (
	logger = logrus.New()
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	app := cli.NewApp()
	app.Name = "splunk-pump"
	app.Usage = "A docker log pump to splunk"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host, d",
			Usage:  "splunk host to push messages to",
			EnvVar: "SPLUNK_HOST",
		},
	}

	app.Action = func(ctx *cli.Context) {
		host := ctx.GlobalString("host")
		if host == "" {
			cli.ShowAppHelp(ctx)
			return
		}

		logsPump := NewLogsPump()
		logsPump.RegisterAdapter(NewSplunkAdapter, host)
		logger.Fatal(logsPump.Run())
	}

	app.Run(os.Args)
}
