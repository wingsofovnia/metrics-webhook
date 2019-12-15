#!/usr/bin/env bash
cd ../ || exit
minikube status

eval $(minikube docker-env)
operator-sdk build docker.io/wingsofovnia/metrics-webhook
eval $(minikube docker-env --unset)

printf '{"spec":{"template":{"metadata":{"labels":{"date":"%s"}}}}}' `date +%s` \
| xargs -0 kubectl patch deployment metrics-webhook -p

cd "$OLDPWD" || exit
