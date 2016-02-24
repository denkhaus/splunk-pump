package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/codegangsta/cli"
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
		shutdown := make(chan int)

		//create a notification channel to shutdown
		sigChan := make(chan os.Signal, 1)
		logsPump := NewLogsPump(storagePath)
		logsPump.RegisterAdapter(NewSplunkAdapter, host)

		go func() {
			logger.Fatal(logsPump.Run())
			shutdown <- 1
		}()
	https: //github.com/xlab/closer/blob/master/cmd/example-error/main.go
		//register for interupt (Ctrl+C) and SIGTERM (docker)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("Shutting down...")
			logsPump.Close()
		}()

		<-shutdown

	}

	app.Run(os.Args)
}
