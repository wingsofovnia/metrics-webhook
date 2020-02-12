# Gorand Webhook Example

```bash
cd $(git rev-parse --show-toplevel)

# Make sure minikube is running
minikube status

# Build gorand to minikube repository
$ eval $(minikube docker-env)
$ docker build -t docker.io/iovchynnikov/gorand:0.1 -f example/gorand/Dockerfile .

# Run gorand with memory restriction
$ kubectl run gorand --image=docker.io/iovchynnikov/gorand:0.1 --requests=cpu=500Mi --limits=cpu=500Mi

# Expose gorand service
$ kubectl expose deployment gorand --type=NodePort --port=8080
$ GORAND_URL=$(minikube service gorand --url)

# Create Metrics Webhook (target ram util = 50) and HPA (target ram util = 90)
$ kubectl create -f example/gorand/metrics-webhook.yaml
$ kubectl create -f example/gorand/hpa.yaml

# Watch for current metrics values in Metrics Webhook Status
$ watch kubectl describe metricwebhooks.metrics.wingsofovnia.github.com gorand-metricwebhook

# Apply load
go get -u github.com/tsenart/vegeta
echo "GET http://$GORAND_URL:8080" | vegeta attack -rate=150/s | vegeta report
```
