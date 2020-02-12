# metrics-webhook <img src="https://travis-ci.com/wingsofovnia/metrics-webhook.svg?branch=master">
Webhook metric alerts for Kuberentes Metrics Server

## Deploy
```bash
# (opt) eval $(minikube docker-env)
operator-sdk build docker.io/iovchynnikov/metrics-webhook

kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml

kubectl create -f deploy/crds/metrics.wingsofovnia.github.com_metricwebhooks_crd.yaml
kubectl create -f deploy/operator.yaml 
```

See `deploy/minikube.*.sh` scripts for Minikube deployment.

## Example
See example/README.md
