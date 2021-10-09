
deps:
	mkdir -p ./.local ./bin

generate-certs: deps
	openssl genrsa -out ./.local/server.key 2048
	openssl req -new -x509 -sha256 \
		-key ./.local/server.key \
		-out ./.local/server.crt -days 3650

build: deps
	go build -o ./bin/lab-server ./cmd/lab-server
