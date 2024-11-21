# Demo Kubernetes Webhook

This Kubernetes Mutating Webhook is developed to inject sidecar container to kubernetes' pod by using given value.



## How to start

HTTP server
```shell
make run
```

HTTPS server
```shell
make generate-localhost-ca
make run -- --tls-enable --tls-key ca-key.pem --tls-cert ca-cert.pem
```