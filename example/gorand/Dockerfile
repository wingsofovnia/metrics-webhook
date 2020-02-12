FROM golang:1.13
ENV GO111MODULE=on

ADD . /go/src/github.com/wingsofovnia/metrics-webhook

WORKDIR /go/src/github.com/wingsofovnia/metrics-webhook/example/gorand
RUN go build .
ENTRYPOINT /go/src/github.com/wingsofovnia/metrics-webhook/example/gorand/gorand

EXPOSE 8080 4030
