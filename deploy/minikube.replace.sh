#!/usr/bin/env bash
cd ../ || exit
minikube status

eval $(minikube docker-env)
operator-sdk build docker.io/iovchynnikov/metrics-webhook
eval $(minikube docker-env --unset)

kubectl replace --force -f deploy/service_account.yaml
kubectl replace --force -f deploy/role.yaml
kubectl replace --force -f deploy/role_binding.yaml

kubectl replace --force -f deploy/crds/metrics.wingsofovnia.github.com_metricwebhooks_crd.yaml
kubectl replace --force -f deploy/operator.yaml

cd "$OLDPWD" || exit
