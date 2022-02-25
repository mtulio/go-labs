
deps:
	mkdir -p ./.local ./bin

generate-certs: deps
	openssl genrsa -out ./.local/server.key 2048
	openssl req -new -x509 -sha256 \
		-key ./.local/server.key \
		-out ./.local/server.crt -days 3650

build: deps
	go build -o ./bin/lab-app-server ./cmd/lab-app-server/

# build single service
build-single:
	go build -o ./bin/lab-bind-all ./cmd/lab-bind-all/

build-k8sapi:
	go build -o ./bin/lab-k8sapi-watcher ./cmd/lab-k8sapi-watcher/

build-lbwatcher:
	go build -o ./bin/lb-watcher ./cmd/lb-watcher/

# lbhc tools
build-lbhc-monitor:
	go build -o ./bin/lbhc-monitor ./cmd/lbhc-monitor/

build-lbhc-app:
	go build -o ./bin/lbhc-app ./cmd/lab-app-server/

build-all:
	$(MAKE) build
	$(MAKE) build-k8sapi

deploy-stack-aws-nlb:
	cd hack/deploy-stack && . .venv/bin/activate && cdk deploy -f
