# lab-unhealthy-requests

Steps to reproduce the lab to validate if a unhealthy node is receiving requests.


## Setup

### Requirements

- VPC
- 2x Subnets (at least) in in different Azs
- igw attached to VPC, and it's default route to subnets
- IAM instance role with policy to allow describe-target-healthy
- NLB in two different subnets/AZs with cross-zone enabled
 - Listener TCP 6443 attached to target group: target type IP, Service port 6443/tcp, health check HTTPS 6444 path /readyz, interval 10s, timeout 10s.
- SG allowing IN 6443-6444/TCP from NLB's subnets
- 2x EC2s in different subnets

### Environment

- Server*: run the app (service HTTPS 6443, health check HTTPS 6444).
- ServerA: Will be the "master-app", app that will be forced to fail the health check, and also will generate traffic to public load balance endpoint. All the logs and metrics will be collected here.
- ServerB: Will run the app, only (without traffic generation)


## Play

- Copy the most recent binaries to the servers

```
for IP in ${IPS_SRV}; do scp bin/lab-* ec2-user@$IP:~/  ; done
```

- generate self-signed cers

```
for IP in ${IPS_SRV}; do ssh ec2-user@$IP "openssl genrsa -out server.key 
2048 ; openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650 -subj '/C=US/ST=MA/L=Springfield/O=Simpsons/CN=localhost'"; done
```

### ServerB

- Run the app on ServerB (ssh and run in background)

``` shell
export APP=t11-svc-tcp-hc-https
export TG_NAME="lab-a-appsv-P1IAY1I6W51O"
export AWS_REGION=us-east-1
export TG_ARN=$(aws elbv2 describe-target-groups --region us-east-1 --query "TargetGroups[?TargetGroupName == \`${TG_NAME}\`].TargetGroupArn" --output text)

./lab-app-server --app-name ${APP} \
  --service-port 6443 --service-proto https \
  --health-check-port 6444 --health-check-proto https \
  --cert-pem ./server.crt --cert-key ./server.key \
  --log-path ./${APP}.log \
  --watch-target-group-arn ${TG_ARN} &
```

> Make sure the command above does not failed in any step (mainly when getting TG ARN, otherwise the IAM Instance role need to be fixed)

- Test if the app is reachable on local node:

``` shell
# Health check endpoint should exists only in HC Server
$ curl -ksw "%{http_code}" https://localhost:6444/readyz -o /dev/null

$ curl -ksw "%{http_code}" https://localhost:6443/ping -o /dev/null
```

> Answer expected: HTTP code `200` for both endpoiints

- Check if the IP of ServerB is healthy on the target group

- Observe the metrics on the log (`${APP}.log`), should be looks like:

```
"app_termination\":false,
"app_healthy\":true,
"tg_healthy\":false,
"tg_health_count\":1,
"tg_unhealth_count\":1,
"reqc_service\":13,
"reqc_hc\":0,
"reqc_client\":40,
"reqc_client_2xx\":40,
"reqc_client_4xx\":0,
"reqc_client_5xx\":0

}","resource":"metrics-push","time":"2021-10-15T14:14:41Z","type":"metrics"}
```

### ServerA 

- get the NLB's DNS

- Test if the app is reachable over the NLB

``` shell
$ export NLB_DNS="mrb-app-accc54e9561c5081.elb.us-east-1.amazonaws.com"

$ curl -ks https://${NLB_DNS}:6443/ping
pong

$ curl -ksw "%{http_code}" https://${NLB_DNS}:6443/ping -o /dev/null
200
```

- Capture the packages (new window)

``` shell
sudo tcpdump -i eth0 tcp port 6443 or tcp port 6444 -w ${APP}.pcap
```

- Run the app on ServerA with request generator

> Make sure all variables is correct

``` shell
export APP=t11-svc-tcp-hc-https
export TG_NAME="lab-a-appsv-P1IAY1I6W51O"
export AWS_REGION=us-east-1
export NLB_DNS="lab-a-LB8A1-T7Z5E46Y4CZR-d5d2049ddb7ffda2.elb.us-east-1.amazonaws.com"
export GEN_REQ_INTERVAL_MS=100

export TG_ARN=$(aws elbv2 describe-target-groups --region us-east-1 --query "TargetGroups[?TargetGroupName == \`${TG_NAME}\`].TargetGroupArn" --output text)


./lab-app-server --app-name ${APP} \
    --service-port 6443 --service-proto https \
    --health-check-port 6444 --health-check-proto https \
    --cert-pem ./server.crt --cert-key ./server.key \
    --log-path ./${APP}.log \
    --watch-target-group-arn ${TG_ARN} \
    --gen-requests-to-url "https://${NLB_DNS}:6443/ping" \
    --gen-requests-slow-start 90 \
    --gen-requests-interval ${GEN_REQ_INTERVAL_MS} \
    --termination-timeout 240 &
```

- Wait the requests generator start and the node start receiving the requests. Metrics: 
`reqc_client` > 0
`reqc_service` > 0

- Send the termination signal to force the HC to fail and the target will transition to unhealthy state. After app back to the service, wait some time to force to finishand stop the requests generator

``` shell
# Send sig term, the app will fail the HC and clean after 240s
sleep 60; kill -TERM $(pidof lab-app-server) ; sleep 360;

# Send 2x sig term, the app will be finished
sleep 60; kill -TERM $(pidof lab-app-server); kill -TERM $(pidof lab-app-server); 

``` 

- Collect the logs

``` shell
for IP in $IPS_SRV; do scp ec2-user@${IP}:~/${APP}.log examples/${APP}-${IP}.log ; done
```

- Rename the ServerA to extract the metrics from it. Ex suffix -MASTER

``` shell
mv examples/${APP}-54.234.62.55.log examples/${APP}-MASTER.log
```

- extract insights from logs: metrics in tsv

``` shell
#> ServerA log
LOG_FILE=examples/${APP}-MASTER.log

echo -e "TIMESTAMP \t APP_TERM \t APP_HEALTHY \t TG_HEALTHY \t TG_HEA_CNT \t TG_UNH_CNT \t REQ_HC_CNT \t REQ_SVC_CNT \t REQ_CLI_CNT \t REQ_CLI_2xx" > ${LOG_FILE}.csv

jq -r '. |select ( .type == "metrics" ) | .msg ' ${LOG_FILE} \
  | jq -r ' [ .time , .app_termination, .app_healthy, .tg_healthy, .tg_health_count , .tg_unhealth_count , .reqc_hc, .reqc_service, .reqc_client, .reqc_client_2xx ] | @tsv ' \
  >> ${LOG_FILE}.csv
```

- Explore the metrics as tables on ${LOG_FILE}.csv
