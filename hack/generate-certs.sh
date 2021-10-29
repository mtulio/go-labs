#!/bin/bash

CERT_PATH="$(dirname $0)/../.local"
mkdir -p ${CERT_PATH}

CERT_KEY="${CERT_PATH}/server.key"
CERT_CRT="${CERT_PATH}/server.crt"

echo "Generating following cert files: "
echo " - key : ${CERT_KEY}"
echo " - cert: ${CERT_CRT}"

openssl genrsa -out ${CERT_KEY} 2048

openssl req -new -x509 -sha256 \
    -key ${CERT_KEY} \
    -out ${CERT_CRT} -days 3650
