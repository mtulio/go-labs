#!/bin/bash

#
# Run server with health check port running sepparated
# from service port, capture the traffic and exit.
#


DT=$(date +%Y%m%d%H%M)
PORT_SVC=6443
PROTO_SVC=https
PORT_HC=6444
PROTO_HC=https
CERT_PATH="./server.crt"
KEY_PATH="./server.key"

VARIANT="tls-cbs"
APP="svc-${PROTO_SVC}-hc-${PROTO_HC}-${DT}-${VARIANT}"
BINARY="lab-app-server-${VARIANT}"

./${BINARY} --app-name "${APP}" \
  --service-port "${PORT_SVC}" \
  --service-proto "${PORT_SVC}" \
  --health-check-port "${PORT_HC}" \
  --health-check-proto "${PROTO_HC}" \
  --cert-pem "$CERT_PATH" --cert-key "$KEY_PATH" \
  --log-path "./${APP}.log" |tee -a "./${APP}.log" &


sleep 30

ss -nltp |grep -E "(${PORT_SVC}|${PORT_HC})"

# Then capture the traffic for 2 minutes
sudo tcpdump -i any -nnn port ${PORT_HC} -G 120 -W 1 -w ${APP}.pcap

sleep 30

# Trigger signal handler to force to exit the app (running 2x)
kill -15 $(pidof ${BINARY}) && \
  kill -15 $(pidof ${BINARY})

echo "FINISHED"
