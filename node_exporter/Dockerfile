FROM golang:1.12 as prom

RUN go get -d github.com/prometheus/node_exporter
WORKDIR /go/src/github.com/prometheus/node_exporter

RUN go build

FROM scratch

COPY --from=prom /go/src/github.com/prometheus/node_exporter/node_exporter /bin/
