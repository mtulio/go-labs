package server;

import (
	"log"
	"fmt"
	"net/http"
	"encoding/json"
)

type ServerHTTP struct {
	proto Protocol
	name     string
	port     uint64
	listener *http.ServeMux
	hcServer  bool
	hcControl *HealthCheckController
}

func NewHTTPServer(
	name string,
	port uint64,
	ctrl *HealthCheckController,
	hcServer bool,
) (*ServerHTTP, error) {
	log.SetFlags(log.Lshortfile)

	server := ServerHTTP{
		proto: ProtoHTTP,
		name: name,
		port: port,
		hcControl: ctrl,
		hcServer: hcServer,
	}

	server.listener = http.NewServeMux()
	server.listener.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
		code := 200
		respBody := fmt.Sprintf("%s", server.hcControl.GetHealthyStr())
		w.Header().Set("Content-Type", "text/plain")		
		if !server.hcControl.GetHealthy() {
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
				Body     string `json:"body"`
				Code int `json:"code"`
			}
			pigeon := &EventRequest{
				Body:     respBody,
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
			SendEvent("request", name, string(data))
		}()

		w.Write([]byte(respBody))
	})

	SendEvent("runtime", name, "Server HTTP Created")
	return &server, nil
}

func (srv *ServerHTTP) Start() {
	msg := fmt.Sprintf("Creating HTTP server on port %d\n", srv.port)
	SendEvent("runtime", srv.name, msg)

	port := fmt.Sprintf(":%d", srv.port)
	log.Fatal(http.ListenAndServe(port, srv.listener))
}

type ServerHTTPS struct {
	proto Protocol
	name     string
	port     uint64
	listener *http.ServeMux
	hcServer  bool
	hcControl *HealthCheckController
	certKey string
	certPem string
}

func NewHTTPSServer(
	name string,
	port uint64,
	ctrl *HealthCheckController,
	hcServer bool,
	certPem string,
	certKey string,
) (*ServerHTTPS, error) {
	log.SetFlags(log.Lshortfile)

	server := ServerHTTPS{
		proto: ProtoHTTPS,
		name: name,
		port: port,
		hcControl: ctrl,
		hcServer: hcServer,
		certPem: certPem,
		certKey: certKey,
	}

	server.listener = http.NewServeMux()
	server.listener.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")		
		w.Write([]byte(fmt.Sprintf("%s", server.hcControl.GetHealthyStr())))

		if !server.hcControl.GetHealthy() {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	SendEvent("runtime", name, "Server HTTPS Created")
	return &server, nil
}

// func (srv *ServerHTTPS) Start() {
// 	msg := fmt.Sprintf("Creating HTTPS server on port %d\n", srv.port)
// 	SendEvent("runtime", srv.name, msg)

// 	http.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "text/plain")		
// 		w.Write([]byte(fmt.Sprintf("%s", srv.hcControl.GetHealthyStr())))
// 		if !srv.hcControl.GetHealthy() {
// 			w.WriteHeader(http.StatusInternalServerError)
// 		}
// 	})

// 	port := fmt.Sprintf(":%d", srv.port)
// 	err := http.ListenAndServeTLS(port, srv.certPem, srv.certKey, nil)
// 	if err != nil {
// 		log.Fatal("ListenAndServe HTTPS: ", err)
// 	}
// }

func (srv *ServerHTTPS) Start() {
	msg := fmt.Sprintf("Creating HTTPS server on port %d\n", srv.port)
	SendEvent("runtime", srv.name, msg)

	port := fmt.Sprintf(":%d", srv.port)
	log.Fatal(http.ListenAndServeTLS(port, srv.certPem, srv.certKey, srv.listener))
}