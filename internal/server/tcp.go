package server;

import (
	"net"
	"log"
	"fmt"
	"crypto/tls"
	"bufio"
)

type ServerTCP struct {
	proto Protocol
	name     string
	port     uint64
	listener net.Listener
	events   *chan string
	hcServer  bool
	hcControl *HealthCheckController
}

func NewTCPServer(
	name string,
	port uint64,
	events *chan string,
	ctrl *HealthCheckController,
) (*ServerTCP, error) {
	log.SetFlags(log.Lshortfile)

	server := ServerTCP{
		proto: ProtoTCP,
		name: name,
		port: port,
		events: events,
	}

	portStr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Println(err)
		return nil, err 	
	}
	defer ln.Close()

	server.listener = ln
	go func() { *events<-"NewTCPServer1" }()
	go func() { *events<-"NewTCPServer2" }()
	//*events<-"NewTCPServer 2"
	return &server, nil
}

func (srv *ServerTCP) Start() {
	msg := fmt.Sprintf("Starting TCP server on port %d\n", srv.port)
	//fmt.Printf(msg)
	//go func() { *srv.events<-msg; }()
	*srv.events<-msg;
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go tcpConnHandler(conn)
	}
}

type ServerTLS struct {
	proto Protocol
	name string
	port uint64
	listener net.Listener
	events *chan string
	hcServer  bool
	hcControl *HealthCheckController
	certKey string
	certPem string
}

func NewTLSServer(
	name string,
	port uint64,
	events *chan string,
	ctrl *HealthCheckController,
	certKey string,
	certPem string,
) (*ServerTLS, error) {

	log.SetFlags(log.Lshortfile)

	srv := ServerTLS{
		proto: ProtoTLS,
		name: name,
		port: port,
		events: events,
		certKey: certKey,
		certPem: certPem,
	}

	cer, err := tls.LoadX509KeyPair(srv.certPem, srv.certKey)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	portStr := fmt.Sprintf(":%d", port)
	ln, err := tls.Listen("tcp", portStr, config)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer ln.Close()

	srv.listener = ln
	*srv.events<-"Server Created"
	return &srv, nil
}

func (srv *ServerTLS) Start() {
	msg := fmt.Sprintf("Starting TLS server on port %d\n", srv.port)
	//fmt.Printf(msg)
	*srv.events<-msg
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go tcpConnHandler(conn)
	}
}

func tcpConnHandler(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		println(msg)

		n, err := conn.Write([]byte("ok\n"))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}