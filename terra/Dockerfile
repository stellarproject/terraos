FROM golang:1.12 as builder

ADD . /go/src/github.com/stellarproject/terraos
WORKDIR /go/src/github.com/stellarproject/terraos/terra

RUN CGO_ENABLED=0 go build -v -ldflags '-s -w -extldflags "-static"'

FROM scratch

COPY --from=builder /go/src/github.com/stellarproject/terraos/terra /
