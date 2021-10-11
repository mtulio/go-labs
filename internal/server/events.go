package server

import (
	logr "github.com/sirupsen/logrus"
)

var (
	AppName = "my-default-app"
)

func init() {
	logr.SetFormatter(&logr.JSONFormatter{})
}

func SendEvent(tp, name, msg string) {
	logr.WithFields(logr.Fields{
		"app-name": AppName,
		"type":     tp,
		"resource": name,
	}).Info(msg)
}
