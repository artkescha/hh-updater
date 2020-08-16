package main

import (
	"flag"
	"github.com/artkescha/hh-updater/config"
	"github.com/artkescha/hh-updater/server"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

var (
	configFile = flag.String("config-file", "./config.yaml", "Configuration file")
	errChan    = make(chan error, 10)
	signalChan = make(chan os.Signal, 1)
)

func main() {
	flag.Parse()
	logrus.Info("Starting hh-updater...")

	config, err := config.ConfigFromFile(*configFile)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	server := server.NewServer(config)
	if err := server.Init(); err != nil {
		logrus.Fatal(err)
		return
	}

	logrus.Debugf("Configuration: %s", config.String())

	go func() {
		errChan <- server.Start()
	}()

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			if err != nil {
				logrus.Fatal(err)
			}
		case signal := <-signalChan:
			logrus.Infof("Captured %v. Exiting...", signal)
			if err := server.Stop(); err != nil {
				logrus.Fatal(err)
			}
			logrus.Info("Bye")
			os.Exit(0)
		}
	}

}
