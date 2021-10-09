package server;


type Protocol uint8

const (
	ProtoTCP Protocol = iota
	ProtoTLS
	ProtoHTTP
	ProtoHTTPS
)

type Server interface {
	Start()
	//ShutdownHealthy() error
	//GetType() string
	//GetState() bool
	//SetState(value bool) error
}