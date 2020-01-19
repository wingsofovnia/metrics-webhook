FROM golang:1.13
ENV GO111MODULE=on

ADD . /go/src/github.com/wingsofovnia/metrics-webhook

WORKDIR /go/src/github.com/wingsofovnia/metrics-webhook/example/server
RUN go build .
ENTRYPOINT /go/src/github.com/wingsofovnia/metrics-webhook/example/server/server

EXPOSE 8080 4030
