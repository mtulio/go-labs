package main

import (
	// "fmt"
	// "io"
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	pathCert          = "./.local/server.crt"
	pathCertKey       = "./.local/server.key"
	listenerTCPPort   = 31044
	listenerHTTPPort  = 31080
	listenerHTTPSPort = 31443
	listenerTLSPort   = 31444
	hcTCPPort         = 32044
	hcHTTPPort        = 32080
	hcHTTPSPort       = 32443
	hcTLSPort         = 32444
	hcStatus          = true
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	msg := "This is an example server.\n"
	w.Write([]byte(msg))
	fmt.Println(req)
	// fmt.Fprintf(w, "This is an example server.\n")
	// io.WriteString(w, "This is an example server.\n")
}

func HealthyHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	msg := fmt.Sprintf("ok: %v", hcStatus)
	w.Write([]byte(msg))
	fmt.Println(req)
	// fmt.Fprintf(w, "This is an example server.\n")
	// io.WriteString(w, "This is an example server.\n")
}

func startHTTPSServer() {
	port := fmt.Sprintf(":%d", listenerHTTPSPort)
	fmt.Printf("Starting HTTPS server on port %s\n", port)
	err := http.ListenAndServeTLS(port, pathCert, pathCertKey, nil)
	if err != nil {
		log.Fatal("ListenAndServe HTTPS: ", err)
	}

}

func startHTTPServer() {
	port := fmt.Sprintf(":%d", listenerHTTPPort)
	fmt.Printf("Starting HTTP server on port %s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe HTTP: ", err)
	}

}

func startHTTPSServerHC() {
	port := fmt.Sprintf(":%d", hcHTTPSPort)
	fmt.Printf("Starting HC HTTPS server on port %s\n", port)
	err := http.ListenAndServeTLS(port, pathCert, pathCertKey, nil)
	if err != nil {
		log.Fatal("ListenAndServe HTTPS: ", err)
	}
}

func startHTTPServerHC() {
	port := fmt.Sprintf(":%d", hcHTTPPort)
	fmt.Printf("Starting HC HTTP server on port %s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe HTTP: ", err)
	}
}

func startTLSServer() {
	log.SetFlags(log.Lshortfile)

	cer, err := tls.LoadX509KeyPair(pathCert, pathCertKey)
	if err != nil {
		log.Println(err)
		return
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	port := fmt.Sprintf(":%d", listenerTLSPort)
	ln, err := tls.Listen("tcp", port, config)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	fmt.Printf("Starting TLS server on port %s\n", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go tcpConnHandler(conn)
	}
}

func startTLSServerHC() {
	log.SetFlags(log.Lshortfile)

	cer, err := tls.LoadX509KeyPair(pathCert, pathCertKey)
	if err != nil {
		log.Println(err)
		return
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	port := fmt.Sprintf(":%d", hcTLSPort)
	ln, err := tls.Listen("tcp", port, config)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	fmt.Printf("Starting HC TLS server on port %s\n", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go tcpConnHandler(conn)
	}
}

func startTCPServer() {
	log.SetFlags(log.Lshortfile)

	port := fmt.Sprintf(":%d", listenerTCPPort)
	ln, err := net.Listen("tcp", port)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	fmt.Printf("Starting TCP server on port %s\n", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go tcpConnHandler(conn)
	}
}

func startTCPServerHC() {
	log.SetFlags(log.Lshortfile)

	port := fmt.Sprintf(":%d", hcTCPPort)
	ln, err := net.Listen("tcp", port)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	fmt.Printf("Starting HC TCP server on port %s\n", port)
	for {
		conn, err := ln.Accept()
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

		n, err := conn.Write([]byte("world\n"))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}

func main() {
	http.HandleFunc("/hello", HelloServer)
	http.HandleFunc("/healthyz", HealthyHandler)
	http.HandleFunc("/readyz", HealthyHandler)

	// Handle sigterm and await termChan signal
	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGTERM)

	go func() {
		<-termChan // Blocks here until interrupted
		log.Print("SIGTERM received. Do nothing yet... TODO fail HC\n")
		hcStatus = false
	}()

	go startTLSServerHC()
	go startTCPServerHC()
	go startHTTPServerHC()
	go startHTTPSServerHC()

	go startTLSServer()
	go startTCPServer()
	go startHTTPServer()
	startHTTPSServer()

	// read and print channel of requests

}
