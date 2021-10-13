# service TCP HealthCheck HTTP


- Start the app
```
APP=mrb-svc-tcp-hc-http

TG_ARN=$(aws elbv2 describe-target-groups --query "TargetGroups[?TargetGroupName == \`${APP}\`].TargetGroupArn" --output text)

./lab-server-listener --app-name ${APP} \
  --service-port 6443 --service-proto tcp \
  --health-check-port 6444 --health-check-proto http \
  --log-path ./${APP}.log \
  --watch-target-group-arn ${TG_ARN}
```

- Read the log


```
jq -r '. |select ( .type == "metrics" ) | .msg ' examples/mrb-svc-tcp-hc-http.log  |jq -r ' [ { "time": .time , "App": .app_healthy, "TG": .tg_healthy } ]  '

```

Create a csv

```
echo -e "TIMESTAMP \t APP_HEALTHY \t TG_HEALTHY \t REQUEST_HC_COUNT" > mrb-svc-tcp-hc-http.csv
jq -r '. |select ( .type == "metrics" ) | .msg ' examples/mrb-svc-tcp-hc-http.log  |jq -r ' [ .time , .app_healthy, .tg_healthy, .reqc_hc ] | @tsv ' >> mrb-svc-tcp-hc-http.csv
```


## HTTPS

- Setup the server

copy bin
```
IP_SRV="54.83.236.78 34.234.170.218 54.224.194.13"
for IP in $IP_SRV; do scp  bin/lab-* ec2-user@$IP:~/; done
```

generate certs

```
for IP in $IP_SRV; do ssh ec2-user@$IP "openssl genrsa -out server.key 
2048"; done
for IP in $IP_SRV; do ssh ec2-user@$IP "openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650 -subj '/C=US/ST=MA/L=Springfield/O=Simpsons/CN=localhost'"; done
```


- Start the app (on each server / or only one)
```
export AWS_REGION=us-east-1
export APP=mrb-tcp-https

TG_ARN=$(aws elbv2 describe-target-groups --region us-east-1 --query "TargetGroups[?TargetGroupName == \`${APP}\`].TargetGroupArn" --output text)

./lab-app-server --app-name ${APP} \
  --service-port 6444 --service-proto tcp \
  --health-check-port 6443 --health-check-proto https \
  --cert-pem ./server.crt --cert-key ./server.key \
  --log-path ./${APP}.log \
  --watch-target-group-arn ${TG_ARN}
```

- Force to fail health check

```
kill -TERM $(pidof lab-app-server)
```

- Wait 120sec to unhealth be cleared

- Collect the logs

```
for IP in $IP_SRV; do scp ec2-user@${IP}:~/mrb-tcp-https.log examples/mrb-tcp-https-${IP}.log; done
```

Create a CSV

```
echo -e "TIMESTAMP \t APP_TERMINATION \t APP_HEALTHY \t TG_HEALTHY \t TG_HEA_CNT \t TG_UNH_CNT \t REQ_HC_CNT" > examples/mrb-svc-tcp-hc-https.csv
jq -r '. |select ( .type == "metrics" ) | .msg ' examples/mrb-tcp-https-54.83.236.78-TERM.log |jq -r ' [ .time , .app_termination, .app_healthy, .tg_healthy, .tg_health_count , .tg_unhealth_count , .reqc_hc ] | @tsv ' >> examples/mrb-svc-tcp-hc-https.csv
```

Create a csv for tests with 3 nodes

```
echo -e "TIMESTAMP \t APP_HEALTHY \t TG_HEALTHY \t TG_HEA_CNT \t TG_UNH_CNT \t REQ_HC_CNT" > mrb-svc-tcp-hc-https-3x.csv
jq -r '. |select ( .type == "metrics" ) | .msg ' examples/mrb-svc-tcp-hc-https-3x-54.162.236.31.log |jq -r ' [ .time , .app_healthy, .tg_healthy, .tg_health_count , .tg_unhealth_count , .reqc_hc ] | @tsv ' >> mrb-svc-tcp-hc-https-3x.csv
```
