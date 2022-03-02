package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

type ServerHTTP struct {
	listener *http.ServeMux
	config   *ServerConfig
}

func NewHTTPServer(cfg *ServerConfig) (*ServerHTTP, error) {
	log.SetFlags(log.Lshortfile)

	srv := ServerHTTP{
		config: cfg,
	}

	srv.listener = http.NewServeMux()

	srv.listener.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		respBody := fmt.Sprintf("pong")
		w.Header().Set("Content-Type", "text/plain")

		go func() {
			type EventRequest struct {
				Body string `json:"body"`
				Code int    `json:"code"`
			}
			req := &EventRequest{
				Body: respBody,
				Code: 200,
			}
			data, _ := json.Marshal(req)
			if srv.config.debug {
				srv.config.event.Send("request", srv.config.name, string(data))
			}
			if srv.config.hcServer {
				srv.config.metric.Inc("requests_hc")
			} else {
				srv.config.metric.Inc("requests_service")
			}
		}()

		w.Write([]byte(respBody))
	})

	srv.listener.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		respBody := fmt.Sprintf("Available routes: \n/ping\n/%s", cfg.hcPath)
		w.Header().Set("Content-Type", "text/plain")

		go func() {

			if srv.config.hcServer {
				srv.config.metric.Inc("requests_hc")
			} else {
				srv.config.metric.Inc("requests_service")
			}

		}()

		w.Write([]byte(respBody))
	})

	// register Health-checkk endpoint only in Health check server

	if cfg.hcServer {
		if cfg.hcPath == "" {
			log.Fatal("Health-check path was not properly defined for Health Check server")
		}
		srv.listener.HandleFunc(cfg.hcPath, func(w http.ResponseWriter, r *http.Request) {
			code := 200
			respBody := srv.config.hc.GetHealthyStr()
			w.Header().Set("Content-Type", "text/plain")
			if !srv.config.hc.GetHealthy() {
				code = 500
				w.WriteHeader(http.StatusInternalServerError)
			}
			go func() {
				type EventRequest struct {
					Body string `json:"body"`
					Code int    `json:"code"`
				}
				req := &EventRequest{
					Body: respBody,
					Code: code,
				}
				data, _ := json.Marshal(req)
				if srv.config.debug {
					srv.config.event.Send("request", srv.config.name, string(data))
				}
				srv.config.metric.Inc("requests_hc")
			}()

			w.Write([]byte(respBody))
		})
	}

	srv.config.event.Send("runtime", srv.config.name, "Server HTTP Created")
	return &srv, nil
}

func (srv *ServerHTTP) Start() {
	protoName := "HTTP"
	if srv.config.proto == ProtoHTTPS {
		protoName = "HTTPS"
	}
	msg := fmt.Sprintf("Creating %s server on port %d\n", protoName, srv.config.port)
	srv.config.event.Send("runtime", srv.config.name, msg)

	port := fmt.Sprintf(":%d", srv.config.port)
	if srv.config.proto == ProtoHTTPS {
		// log.Fatal(http.ListenAndServeTLS(
		// 	port, srv.config.certPem,
		// 	srv.config.certKey, srv.listener),
		// )
		// log.Fatal(server.ListenAndServeTLS(srv.config.certPem, srv.config.certKey))
		cert, err := tls.LoadX509KeyPair(
			srv.config.certPem, srv.config.certKey,
		)
		if err != nil {
			log.Fatal(err)
		}
		server := &http.Server{
			Addr:    port,
			Handler: srv.listener,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				// CipherSuites: []uint16{
				// 	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				// 	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				// },
				MinVersion: tls.VersionTLS12,
				VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
					fmt.Println("> tlsCallback VerifyPeerCertificate: ")
					fmt.Printf(">> rawCerts: %+v\n", rawCerts)
					fmt.Printf(">> rawCerts: %+v\n", verifiedChains)
					return nil
				},
				VerifyConnection: func(s tls.ConnectionState) error {
					fmt.Println("> tlsCallback VerifyConnection: ")
					fmt.Printf(">> ConnectionState: %+v\n", s)
					return nil
				},
				GetConfigForClient: func(ci *tls.ClientHelloInfo) (*tls.Config, error) {
					fmt.Println("> tlsCallback GetConfigForClient: ")
					fmt.Printf(">> ClientHelloInfo: %+v\n", ci)
					return nil, nil
				},
				GetClientCertificate: func(ci *tls.CertificateRequestInfo) (*tls.Certificate, error) {
					fmt.Println("> tlsCallback GetClientCertificate: ")
					fmt.Printf(">> CertificateRequestInfo: %+v\n", ci)
					return nil, nil
				},
				GetCertificate: func(ci *tls.ClientHelloInfo) (*tls.Certificate, error) {
					fmt.Println("> tlsCallback GetCertificate: ")
					fmt.Printf(">> ClientHelloInfo: %+v\n", ci)
					return nil, nil
				},
			},
		}
		if srv.config.debug {
			tlsKeysFD, err := os.OpenFile(srv.config.listener.options.DebugTLSKeysLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				panic(err)
			}
			defer tlsKeysFD.Close()
			server.TLSConfig.KeyLogWriter = tlsKeysFD
		}

		addr := server.Addr
		if addr == "" {
			addr = ":https"
		}
		fmt.Println(port, addr)
		lnCfg := &net.ListenConfig{
			Control:   TCPControl,
			KeepAlive: -1,
		}
		ln, err := lnCfg.Listen(context.Background(), "tcp", addr)
		if err != nil {
			log.Fatal(err)
		}

		defer ln.Close()

		log.Fatal(server.ServeTLS(ln, "", ""))

	}
	log.Fatal(http.ListenAndServe(port, srv.listener))
}

// StartController will do nothing in HTTP/S servers (only TCP).
func (srv *ServerHTTP) StartController() {
}
