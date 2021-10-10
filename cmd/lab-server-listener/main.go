package main;

import (
	"log"
	//"fmt"
	"github.com/mtulio/go-lab-api/internal/server"
)

func main() {
	readyToShutdown := make(chan struct{})
	// events := make(chan string)
	
	// go func() {
	// 	select {
	// 	case msg := <-events:
	// 		fmt.Println("received: ", msg)
	// 	}
	// }()
	
	lnc := server.ListenerOptions{
		ServiceProto: server.ProtoTCP,
		ServicePort: 30300,
		HCProto: server.ProtoTCP,
		HCPort: 30301,
	}
	//fmt.Println(lnc)
	ln, err := server.NewListener(&lnc)
	if err != nil {
		log.Fatal("ERROR starting the listener")
	}
	//fmt.Println(">>>>>")
	// go func() { events<-"xxxxxxxx" }()
	//fmt.Println(err)
	ln.Start()
	//fmt.Println(ln)

	// read and print channel of requests
	<-readyToShutdown
}