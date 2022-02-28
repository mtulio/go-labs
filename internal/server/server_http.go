package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
		w.Header().Set("Content-Type", "text/plain")

		respBody := fmt.Sprintf("pong")
		m := srv.config.metric
		path := "/ping"

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
				m.Inc("requests_hc")
				m.PromReqInc("200", GetServerTypeToStr(ServerTypeHC), path)
			} else {
				m.Inc("requests_service")
				m.PromReqInc("200", GetServerTypeToStr(ServerTypeSvc), path)
			}
		}()

		w.Write([]byte(respBody))
	})

	srv.listener.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		m := srv.config.metric
		path := "/"
		respBody := fmt.Sprintf("Available routes: \n/ping\n/%s", cfg.hcPath)
		w.Header().Set("Content-Type", "text/plain")

		go func() {

			if srv.config.hcServer {
				m.Inc("requests_hc")
				m.PromReqInc("200", GetServerTypeToStr(ServerTypeHC), path)
			} else {
				m.Inc("requests_service")
				m.PromReqInc("200", GetServerTypeToStr(ServerTypeSvc), path)
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
			m := srv.config.metric
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
				m.PromReqInc("200", GetServerTypeToStr(ServerTypeHC), cfg.hcPath)
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
		log.Fatal(http.ListenAndServeTLS(
			port, srv.config.certPem,
			srv.config.certKey, srv.listener),
		)
	}
	log.Fatal(http.ListenAndServe(port, srv.listener))
}

// StartController will do nothing in HTTP/S servers (only TCP).
func (srv *ServerHTTP) StartController() {
}
