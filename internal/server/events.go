

package server;

import (
	//"fmt"
	zlog "github.com/rs/zerolog/log"
)

func SendEvent(tp, name, msg string) {
	//fmt.Println(msg)
	//zlog.Print(msg)
	zlog.Info().
		Str("type", tp).
		Str("type", name).
		Msg(msg)
}