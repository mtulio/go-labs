# nginx service and hc servers

Lab to validate HTTPS health check endpoint reporting RST TCP packets on NLB metric (TCP_Target_Reset_Count).

The basic architecture:
- 2x EC2
- 1x NLB, with listener 6443/TCP
- 1x Target Group with service port 6443/tcp and health check HTTPS 6444/TCP checking path /readyz, with 2 check with 10s interval

## Deploy the infra/stack

Pre-req:
- CDK

Deploy:
``` shell
cd hack/deploy-stack
cdk deploy
```

Adjust the listener:
- open the Listener 6443 from balancer created by CDK
- change to target group appropriated to your tests. **In this case the problem is the Target group with health check HTTPS, so choose it.**

## ISET Web Server

Install, setup, enable and test web servers Nginx and Httpds.

### NGINX Install

- nginx
```
sudo amazon-linux-extras install epel
sudo yum install nginx -y
sudo systemctl enable nginx --now
```

- certbot/lets encrypt 
https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/SSL-on-amazon-linux-2.html#letsencrypt

```
sudo yum install -y certbot python2-certbot-nginx
```

sudo certbot --register-unsafely-without-email -d nlb-lab-s01.devcluster.openshift.com -d nlb-lab-s02.devcluster.openshift.com -d nlb-lab.devcluster.openshift.com

### NGINX Setup

#### HTTPS health check

- `/etc/nginx/conf.d/lab.conf`

``` conf
server {
    listen 6443;
    listen [::]:6443;
    location / {
        return 200 'ok';
    }
    location /readyz {
        return 200 'ok';
    }
    location /healthyz {
        return 200 'ok';
    }
}

server {
    listen 6444 ssl http2;
    listen [::]:6444 ssl http2;

    ssl_certificate "/home/ec2-user/server.crt";
    ssl_certificate_key "/home/ec2-user/server.key";

    location / {
        return 200 'ok';
    }
    location /readyz {
        return 200 'ok';
    }
    location /healthyz {
        return 200 'ok';
    }
}
```

#### HTTP health check

- `/etc/nginx/conf.d/lab.conf`

``` conf
server {
    listen 6443;
    listen [::]:6443;
    location / {
        return 200 'ok';
    }
    location /readyz {
        return 200 'ok';
    }
    location /healthyz {
        return 200 'ok';
    }
}

server {
    listen 6444;
    listen [::]:6444;

    location / {
        return 200 'ok';
    }
    location /readyz {
        return 200 'ok';
    }
    location /healthyz {
        return 200 'ok';
    }
}
```

#### NGINX Start

Check the config

```
sudo nginx -t
```

Start the service

``` shell
sudo systemctl restart nginx
```

Check if the ports are open

``` shell
sudo ss -nlp |grep -e 6444 -e 6444
```

Test using the balancer endpoint:

``` shell

curl http://<NLB_DNS>:6443/readyz
curl http://<NLB_DNS>:6443/ok

```

NLB metrics:

- Target reset Count
```

```

### Httpd Install

#### Httpd Install

``` shell
sudo yum install httpd mod_ssl -y
```

- static files

``` shell
mkdir -p /var/www/html/lab
echo "ok" > /var/www/html/lab/ok.html
echo "home" > /var/www/html/lab/index.html
```

#### Httpd Setup

- HTTPS health check

``` conf
LoadModule ssl_module modules/mod_ssl.so

Listen 6443
<VirtualHost *:6443>
    DocumentRoot "/var/www/html/lab"
    ErrorLog "logs/lab-6443.log"
    TransferLog "logs/lab-6443.log"

    RewriteEngine  on
    RewriteRule    "^/readyz$"  "/ok.html" [PT]
</VirtualHost>

Listen 6444
<VirtualHost *:6444>
    DocumentRoot "/var/www/html/lab"
    ErrorLog "logs/lab-6444.log"
    TransferLog "logs/lab-6444.log"

    RewriteEngine  on
    RewriteRule    "^/readyz$"  "/ok.html" [PT]

    SSLEngine on
    SSLCertificateFile "/home/ec2-user/server.crt"
    SSLCertificateKeyFile "/home/ec2-user/server.key"
</VirtualHost>
```

- HTTP health check

``` conf
Listen 6443
<VirtualHost *:6443>
    DocumentRoot "/var/www/html/lab"
    ErrorLog "logs/lab-6443.log"
    TransferLog "logs/lab-6443.log"

    RewriteEngine  on
    RewriteRule    "^/readyz$"  "/ok.html" [PT]
</VirtualHost>

Listen 6444
<VirtualHost *:6444>
    DocumentRoot "/var/www/html/lab"
    ErrorLog "logs/lab-6444.log"
    TransferLog "logs/lab-6444.log"

    RewriteEngine  on
    RewriteRule    "^/readyz$"  "/ok.html" [PT]
</VirtualHost>
```

#### Httpd Enable

``` shell
systemctl restart httpd
```

#### Httpd Test

- https
``` shell
curl -sk https://localhost:6444/readyz
```
