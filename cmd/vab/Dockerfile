FROM golang:1.12 as builder

ADD . /go/src/github.com/stellarproject/terraos
WORKDIR /go/src/github.com/stellarproject/terraos

RUN cd cmd/vab && CGO_ENABLED=0 go build -v -ldflags '-s -w -extldflags "-static"'

FROM scratch

COPY --from=builder /go/src/github.com/stellarproject/terraos/cmd/vab/vab /usr/local/bin/
