package server

import (
	zlog "github.com/rs/zerolog/log"
)

var (
	AppName = "my-default-app"
)

func SendEvent(tp, name, msg string) {
	zlog.Info().
		Str("app-name", AppName).
		Str("type", tp).
		Str("resource", name).
		Msg(msg)
}
