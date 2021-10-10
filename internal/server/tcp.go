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
	ctrl *HealthCheckController,
	hcServer bool,
) (*ServerTCP, error) {
	log.SetFlags(log.Lshortfile)

	server := ServerTCP{
		proto: ProtoTCP,
		name: name,
		port: port,
		hcControl: ctrl,
		hcServer: hcServer,
	}

	SendEvent("runtime", name, "Server TCP Created")
	return &server, nil
}

func (srv *ServerTCP) Start() {
	msg := fmt.Sprintf("Creating TCP server on port %d\n", srv.port)
	SendEvent("runtime", srv.name, msg)
	
	portStr := fmt.Sprintf(":%d", srv.port)
	ln, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatal(err)
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

		SendEvent("request", srv.name, msg)

		healthy := fmt.Sprintf("%s\n", srv.hcControl.GetHealthyStr())
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
	ctrl *HealthCheckController,
	hcServer bool,
	certKey string,
	certPem string,
) (*ServerTLS, error) {

	log.SetFlags(log.Lshortfile)

	srv := ServerTLS{
		proto: ProtoTLS,
		name: name,
		port: port,
		//events: events,
		hcControl: ctrl,
		hcServer: hcServer,
		certKey: certKey,
		certPem: certPem,
	}

	//*srv.events<-"Server Created"
	SendEvent("runtime", name, "Server TLS Created")
	return &srv, nil
}

func (srv *ServerTLS) Start() {

	cer, err := tls.LoadX509KeyPair(srv.certPem, srv.certKey)
	if err != nil {
		log.Fatal(err)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}}
	portStr := fmt.Sprintf(":%d", srv.port)

	ln, err := tls.Listen("tcp", portStr, tlsConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	srv.listener = ln

	msg := fmt.Sprintf("Starting TLS server on port %d\n", srv.port)
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

		SendEvent("request", srv.name, msg)

		healthy := fmt.Sprintf("%s\n", srv.hcControl.GetHealthyStr())
		n, err := conn.Write([]byte(healthy))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}