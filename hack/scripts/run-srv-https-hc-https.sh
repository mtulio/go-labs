#!/bin/bash

#
# Run server with health check port running sepparated
# from service port, capture the traffic and exit.
#

PIDS=()
PID_SERVER=()
DT=$(date +%Y%m%d%H%M)
PORT_SVC=6443
PROTO_SVC=https
PORT_HC=6444
PROTO_HC=https
CERT_PATH="./server.crt"
KEY_PATH="./server.key"

#VARIANT="-tls-cbs"
VARIANT=""
APP="svc-${PROTO_SVC}-hc-${PROTO_HC}-${DT}${VARIANT}"
BINARY="lab-app-server${VARIANT}"

function ctrl_c_handler() {
  # echo "#> CTRL+C handler detected! Killing process and it's childdrens: ${RUN_PIDS[@]}"
  # for parent_pid in ${RUN_PIDS[@]}; do
  #   echo "Killing children's [${parent_pid}] process: $(pgrep -P ${parent_pid})"
  #   kill -15 $(pgrep -P ${parent_pid}) || (echo "#>> Error killing children."; true; )
  #   sleep 2;
  #   echo "Killing parent [${parent_pid}]"
  #   kill -15 ${parent_pid} || ( echo "#>> Error killing parent."; true; )
  # done
  kill -9 ${PID_SERVER[@]} ${PIDS[@]}
}
trap ctrl_c_handler SIGINT


start_server() {
  ./${BINARY} --app-name "${APP}" \
    --service-port "${PORT_SVC}" \
    --service-proto "${PORT_SVC}" \
    --health-check-port "${PORT_HC}" \
    --health-check-proto "${PROTO_HC}" \
    --cert-pem "$CERT_PATH" --cert-key "$KEY_PATH" \
    --log-path "./${APP}.log" \
    --debug \
    --debug-tls-keys-log-file "./${APP}-tlsKeys-srv.log" \
    |tee -a "./${APP}.log"
}

# capture the traffic for 2 minutes
start_capture_traffic() {
  local traffic_period_sec, capture_timeout
  traffic_period_sec=120
  capture_timeout=$((${traffic_period_sec} + 5))

  echo "#> Starting traffic capture for ${traffic_period_sec} seconds. Now=[$(date)]"
  sudo timeout ${capture_timeout} tcpdump -i any -nnn port ${PORT_HC} \
    -G ${traffic_period_sec} -W 1 -w ./${APP}.pcap
}

# Make client requests from custom HC agent
# Path will be /ping to compare traffic with NLB Agent /readyz
make_client_requests() {
  # TLSv1.2
  tls_profile=tls13i
  ./health-check-agent \
    --url "https://localhost:6444/ping" \
    --tls-profile ${tls_profile} \
    --watch-count 5 \
    --no-keep-alive \
    --debug-tls-keys-log-file "./${APP}-tlsKeys-cli.log" \
    | tee -a "${APP}-hc_cli.log"

  # TLSv1.3
  TLS_PROFILE=tls13i
  ./health-check-agent \
    --url "https://localhost:6444/ping" \
    --tls-profile ${tls_profile} \
    --watch-count 5 \
    --no-keep-alive \
    --debug-tls-keys-log-file "./${APP}-tlsKeys-cli.log" \
    | tee -a "${APP}-hc_cli.log"
}

start_server &
PID_SERVER+=($!)
sleep 60

ss -nltp |grep -E "(${PORT_SVC}|${PORT_HC})"
start_capture_traffic &
PIDS+=($!)
sleep 10

make_client_requests
sleep 30

echo "INFO: Waiting process to complete ... Now=[$(date)]"
wait "${PIDS[@]}"
echo "INFO: processes completed."

# Trigger signal handler to force to exit the app (running 2x)
kill -15 $(pidof ${BINARY}) && \
  kill -15 $(pidof ${BINARY})

echo "# Artifacts ready to be collected: "
ls -l ${APP}*

echo -e "\nscp ec2-user@$(curl -s https://mtulio.net/api/ip):~/${APP}* ."
