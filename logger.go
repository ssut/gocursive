package main

import "github.com/Sirupsen/logrus"

var log = logrus.New()

func initLogger() {
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	log.Formatter = formatter
}
