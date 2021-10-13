# kube-api-watcher

This example was created to measure total time until the kube-apiserver
termination starts (signal received), become unhealthy (on target group),
and back to operation state (healthy on HC endpoint and TG).

## Steps

- Run watcher targeting the master node IP and target group

``` shell
./lab-k8sapi-watcher \
    --target-group-arn <arn> \
    --endpoint https://localhost:6443/healthy
```

- Waiting until `app_healthy` and `tg_healthy` are healthy (`true`)

- Send TERM signal to apiserver and watcher (it will ignore)

- Observe the results


## Usage

### Test it locally (fake apiserver)

- bind local `health check` to 6443 port:

``` shell
APP=k8sapi-watcher-server
./bin/lab-app-server --app-name ${APP} \
  --service-port 6444 --service-proto https \
  --health-check-port 6443 --health-check-proto https \
  --cert-pem ./server.crt --cert-key ./server.key
```
- start the watcher locally (without TG ARN)

```
./bin/lab-k8sapi-watcher
```

- Send termination signal to both apps

``` shell
kill -TERM $(pidof lab-app-server); kill -TERM $(pidof lab-k8sapi-watcher)
```

you will see on the event log the message:
``` json
{"app":"k8sapi-watcher","level":"info","msg":"Termination Signal receievd","resource":"k8s-watcher-signal","time":"2021-10-12T11:36:26-03:00","type":"runtime"}
{"app":"k8sapi-watcher","level":"info","msg":"Running Signal handler","resource":"hc-controller","time":"2021-10-12T11:36:26-03:00","type":"runtime"}

```

Then the metric will be set:
```
\"app_termination\":true
```

- Observe until the `app_healthy=true` again, and also termination progress should be cleared `"app_termination\":false`


### Run on kube-apiserver

- Start the k8sapi-watcher

```
export AWS_REGION=us-east-1
./lab-k8sapi-watcher \
    --target-group-arn "arn:aws:elasticloadbalancing:us-east-1:269733383066:targetgroup/aw-8h8pb-aint/b72b0d7162c0adc9" \
    --log-path ./k8sapi-watcher.log \
    --endpoint "https://localhost:6443/readyz"
```

- Send kill

```
kill -TERM $(pidof kube-apiserver); kill -TERM $(pidof lab-k8sapi-watcher)
```

- Observe it, wait for app is back and TG targets are all healthy

- Stop the app

- Extract insights from the logs/metrics

``` 
echo -e "TIMESTAMP \t APP_TERMINATION \t APP_HEALTHY \t TG_HEALTHY \t TG_HEA_CNT \t TG_UNH_CNT \t REQ_HC_CNT" > examples/k8sapi-watcher.csv
jq -r '. |select ( .type == "metrics" ) | .msg ' examples/k8sapi-watcher.log |jq -r ' [ .time , .app_termination, .app_healthy, .tg_healthy, .tg_health_count , .tg_unhealth_count , .reqc_hc ] | @tsv ' >> examples/k8sapi-watcher.csv
```
