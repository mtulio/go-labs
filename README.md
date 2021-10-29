# go-lab-api (WIP)

Dummy Go API to run labs behind AWS *LB.

Supported protocols (both service and health-check servers):
- TCP
- TLS
- HTTP
- HTTPS

## Lab Apps

### Lab 'app-server'

- Start App to bind servers `service` and `healtch-check` using different protocols (allowed: TCP, TLS, HTTP and HTTPS) and ports
- Watch Target group
- Send termination signal (default timeout 2 minutes)
- observe the metrics

### Lab 'k8sapi-watcher'

- Handle signal to count whether the termination time have started
- Pull /healthy and register the response code (bool)
- Pull TG ARN healthy targets (bool)
- Dump metrics

### Lab 'bind-all'

## Examples `app-server`

## Service TCP and Health Check HTTPS

- Run the app that will bind in service and health-check ports


``` shell
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

- Create a CSV for the main metrics that is printed to stdout every 1s

```
echo -e "TIMESTAMP \t APP_TERMINATION \t APP_HEALTHY \t TG_HEALTHY \t TG_HEA_CNT \t TG_UNH_CNT \t REQ_HC_CNT" > examples/mrb-svc-tcp-hc-https.csv
jq -r '. |select ( .type == "metrics" ) | .msg ' examples/mrb-tcp-https-54.83.236.78-TERM.log |jq -r ' [ .time , .app_termination, .app_healthy, .tg_healthy, .tg_health_count , .tg_unhealth_count , .reqc_hc ] | @tsv ' >> examples/mrb-svc-tcp-hc-https.csv
```
