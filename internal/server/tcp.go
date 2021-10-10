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
	//events *chan string,
	ctrl *HealthCheckController,
) (*ServerTCP, error) {
	log.SetFlags(log.Lshortfile)

	server := ServerTCP{
		proto: ProtoTCP,
		name: name,
		port: port,
		//events: events,
		hcControl: ctrl,
	}

	//go func() { *events<-"NewTCPServer1" }()
	//go func() { *events<-"NewTCPServer2" }()
	SendEvent("runtime", name, "Server TCP Created")
	//*events<-"NewTCPServer 2"
	return &server, nil
}

func (srv *ServerTCP) Start() {
	msg := fmt.Sprintf("Creating TCP server on port %d\n", srv.port)
	SendEvent("runtime", srv.name, msg)
	//fmt.Printf(msg)
	//go func() { *srv.events<-msg; }()
	//*srv.events<-msg;
	
	portStr := fmt.Sprintf(":%d", srv.port)
	ln, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatal(err)
		//return nil, err 	
	}
	defer ln.Close()

	srv.listener = ln

	msg = fmt.Sprintf("Starting TCP server on port %d\n", srv.port)
	SendEvent("runtime", srv.name, msg)
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go srv.connHandler(conn)
	}
}

func (srv *ServerTCP) connHandler(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		//println(msg)
		SendEvent("request", srv.name, msg)

		healthy := fmt.Sprintf("%v\n", srv.hcControl.Healthy)
		n, err := conn.Write([]byte(healthy))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}

type ServerTLS struct {
	proto Protocol
	name string
	port uint64
	listener net.Listener
	//events *chan string
	hcServer  bool
	hcControl *HealthCheckController
	certKey string
	certPem string
}

func NewTLSServer(
	name string,
	port uint64,
	//events *chan string,
	ctrl *HealthCheckController,
	certKey string,
	certPem string,
) (*ServerTLS, error) {

	log.SetFlags(log.Lshortfile)

	srv := ServerTLS{
		proto: ProtoTLS,
		name: name,
		port: port,
		//events: events,
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
	//*srv.events<-"Server Created"
	SendEvent("runtime", name, "Server TLS Created")
	return &srv, nil
}

func (srv *ServerTLS) Start() {
	msg := fmt.Sprintf("Starting TLS server on port %d\n", srv.port)
	//fmt.Printf(msg)
	//*srv.events<-msg
	SendEvent("runtime", srv.name, msg)
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go srv.connHandler(conn)
	}
}

func (srv *ServerTLS) connHandler(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		//println(msg)
		SendEvent("request", srv.name, msg)

		healthy := fmt.Sprintf("%v\n", srv.hcControl.Healthy)
		n, err := conn.Write([]byte(healthy))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}