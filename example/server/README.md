# Golorem Webhook Example

```bash
cd $(git rev-parse --show-toplevel)

# Make sure minikube is running
minikube status

# Build golorem to minikube repository
$ eval $(minikube docker-env)
$ docker build -t docker.io/wingsofovnia/metrics-webhook/golorem:0.1 -f example/server/Dockerfile .

# Run golorem with memory restriction
$ kubectl run golorem --image=docker.io/wingsofovnia/metrics-webhook/golorem:0.1 --requests=memory=100Mi --limits=memory=100Mi --port=8080

# Expose golorem service
$ kubectl expose deployment golorem --type=NodePort
$ GOLOREM_URL=$(minikube service golorem --url)

# Create Metrics Webhook (target ram util = 50) and HPA (target ram util = 90)
$ kubectl create -f example/server/metrics-webhook.yaml
$ kubectl create -f example/server/hpa.yaml

# Watch for current metrics values in Metrics Webhook Status
$ watch kubectl describe metricwebhooks.metrics.wingsofovnia.github.com golorem-metricwebhook

# Apply load
command $(cd example/client && go build .) && ./example/client/client -server=$GOLOREM_URL -duration=0s
```
