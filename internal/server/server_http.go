package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
		respBody := fmt.Sprintf("Available routes: \n/healthy\n/ping")
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
			respBody := fmt.Sprintf("%s", srv.config.hc.GetHealthyStr())
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
				// event := `{
				// 	"response": {
				// 		"body": respBody,
				// 		"code": code
				// 	},
				// 	"request": r
				// }`
				if srv.config.debug {
					srv.config.event.Send("request", srv.config.name, string(data))
				}
				srv.config.metric.Inc("requests_hc")

			}()

			w.Write([]byte(respBody))

			hConn := r.Header.Get("Connection")
			if strings.ToLower(hConn) != "close" {
				return
			}

			// close connection
			//contentLength := strconv.Itoa(len(respBody))
			//w.Header().Set("Content-Length", contentLength)
			//w.Header().Set("X-Content-Length", contentLength)
			w.Header().Del("Transfer-Encoding")

			// Check that the rw can be hijacked.
			hj, ok := w.(http.Hijacker)
			//fmt.Println("finish 00")
			// The rw can't be hijacked, return early.
			if !ok {
				srv.config.event.Send("request", srv.config.name, string("can't hijack rw"))
				//fmt.Println("can't hijack rw")
				return
			}
			//fmt.Println("finish 01")
			// Hijack the rw.
			conn, _, err := hj.Hijack()
			if err != nil {
				srv.config.event.Send("request", srv.config.name, string("hijack handle error conn"))
				//fmt.Println("can't hijack rw")
				return
			}
			//fmt.Println("finish 02")
			// Close the hijacked raw tcp connection.
			if err := conn.Close(); err != nil {
				srv.config.event.Send("request", srv.config.name, string("hijack handle error Close"))
				//fmt.Println("can't hijack rw")
				return
			}
			//fmt.Println("finish 03")
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
