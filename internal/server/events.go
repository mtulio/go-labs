package server

import (
	"os"

	"github.com/sirupsen/logrus"
)

type EventHandler struct {
	Type    string
	AppName string
	LogPath string
	logr    *logrus.Logger
}

func NewEventHandler(app, logPath string) *EventHandler {
	ev := EventHandler{
		AppName: app,
		LogPath: logPath,
	}
	var logr = logrus.New()
	logr.SetFormatter(&logrus.JSONFormatter{})

	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logr.Out = file
		} else {
			logr.Info("Failed to log to file, using default stderr")
		}
	}

	ev.logr = logr
	return &ev
}

func (ev *EventHandler) SendEvent(tp, name, msg string) {
	ev.logr.WithFields(logrus.Fields{
		"app":      ev.AppName,
		"type":     tp,
		"resource": name,
	}).Info(msg)
}

func (ev *EventHandler) Send(tp, name, msg string) {
	ev.logr.WithFields(logrus.Fields{
		"app-name": ev.AppName,
		"type":     tp,
		"resource": name,
	}).Info(msg)
}
