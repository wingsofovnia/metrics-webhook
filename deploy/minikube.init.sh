#!/usr/bin/env bash
cd ../ || exit
minikube status

eval $(minikube docker-env)
operator-sdk build docker.io/wingsofovnia/metrics-webhook
eval $(minikube docker-env --unset)

kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml

kubectl create -f deploy/crds/metrics.wingsofovnia.github.com_metricwebhooks_crd.yaml
kubectl create -f deploy/operator.yaml

cd "$OLDPWD" || exit
