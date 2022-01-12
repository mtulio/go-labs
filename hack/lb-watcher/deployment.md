---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: lb-watcher
  labels:
    app: lb-watcher
spec:
  replicas: 1
  selector:
    matchLabels:
      app: lb-watcher
  template:
    metadata:
      labels:
        app: lb-watcher
    spec:
      containers:
      - name: lb-watcher
        image: quay.io/rhn_support_mrbraga/lb-watcher:latest
        imagePullPolicy: Always
        command: # ToDo move to read from config injected by init
          - /app/lb-watcher
          - --lb-aws-target-group-arn
          - "arn:aws:elasticloadbalancing:us-east-1:12345689:targetgroup/mrblts22-zlplk-aext/78b2b4e23a84ed85"
          - --internal-endpoints
          - "https://10.0.2.249:6443/readyz,https://10.0.22.104:6443/readyz,https://10.0.2.250:6443/readyz"
          - --lb-endpoint
          - "https://1.1.1.1:6443/readyz"
          - --log-path
          - "/tmp/kas-watcher.log"
        env:
          - name: AWS_REGION
            value: us-east-1
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: lb-watcher
  name: lb-watcher
spec:
  ports:
  - port: 9999
    protocol: TCP
    targetPort: 9999
    name: web
  selector:
    app: lb-watcher
  type: ClusterIP
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    app: lb-watcher
  name: lb-watcher
spec:
  endpoints:
  - interval: 1s
    port: web
    scheme: http
  selector:
    matchLabels:
      app: lb-watcher
