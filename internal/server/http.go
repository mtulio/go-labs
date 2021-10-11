package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type ServerHTTP struct {
	proto     Protocol
	name      string
	port      uint64
	listener  *http.ServeMux
	hcServer  bool
	hcControl *HealthCheckController
	Event     *EventHandler
}

func NewHTTPServer(
	name string,
	port uint64,
	ctrl *HealthCheckController,
	hcServer bool,
	ev *EventHandler,
) (*ServerHTTP, error) {
	log.SetFlags(log.Lshortfile)

	srv := ServerHTTP{
		proto:     ProtoHTTP,
		name:      name,
		port:      port,
		hcControl: ctrl,
		hcServer:  hcServer,
		Event:     ev,
	}

	srv.listener = http.NewServeMux()
	srv.listener.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
		code := 200
		respBody := fmt.Sprintf("%s", srv.hcControl.GetHealthyStr())
		w.Header().Set("Content-Type", "text/plain")
		if !srv.hcControl.GetHealthy() {
			code = 500
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Println(code)
		go func() {
			// type EventRequest struct {
			// 	Response map[string]string `json:response`
			// 	Request interface{} `json:response`
			// }
			// event := EventRequest{
			// 	Response: {
			// 		"body": respBody,
			// 		"code": code,
			// 	},
			// }
			type EventRequest struct {
				Body string `json:"body"`
				Code int    `json:"code"`
			}
			pigeon := &EventRequest{
				Body: respBody,
				Code: code,
			}
			data, _ := json.Marshal(pigeon)
			// event := `{
			// 	"response": {
			// 		"body": respBody,
			// 		"code": code
			// 	},
			// 	"request": r
			// }`
			srv.Event.Send("request", name, string(data))
		}()

		w.Write([]byte(respBody))
	})

	srv.Event.Send("runtime", name, "Server HTTP Created")
	return &srv, nil
}

func (srv *ServerHTTP) Start() {
	msg := fmt.Sprintf("Creating HTTP server on port %d\n", srv.port)
	srv.Event.Send("runtime", srv.name, msg)

	port := fmt.Sprintf(":%d", srv.port)
	log.Fatal(http.ListenAndServe(port, srv.listener))
}

type ServerHTTPS struct {
	proto     Protocol
	name      string
	port      uint64
	listener  *http.ServeMux
	hcServer  bool
	hcControl *HealthCheckController
	certKey   string
	certPem   string
	Event     *EventHandler
}

func NewHTTPSServer(
	name string,
	port uint64,
	ctrl *HealthCheckController,
	hcServer bool,
	ev *EventHandler,
	certPem string,
	certKey string,
) (*ServerHTTPS, error) {
	log.SetFlags(log.Lshortfile)

	srv := ServerHTTPS{
		proto:     ProtoHTTPS,
		name:      name,
		port:      port,
		hcControl: ctrl,
		hcServer:  hcServer,
		certPem:   certPem,
		certKey:   certKey,
		Event:     ev,
	}

	srv.listener = http.NewServeMux()
	srv.listener.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fmt.Sprintf("%s", srv.hcControl.GetHealthyStr())))

		if !srv.hcControl.GetHealthy() {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	srv.Event.Send("runtime", name, "Server HTTPS Created")
	return &srv, nil
}

func (srv *ServerHTTPS) Start() {
	msg := fmt.Sprintf("Creating HTTPS server on port %d\n", srv.port)
	srv.Event.Send("runtime", srv.name, msg)

	port := fmt.Sprintf(":%d", srv.port)
	log.Fatal(http.ListenAndServeTLS(port, srv.certPem, srv.certKey, srv.listener))
}
