FROM golang:1.12 as cni

RUN go get -d github.com/containernetworking/plugins || true
WORKDIR /go/src/github.com/containernetworking/plugins

RUN ./build_linux.sh

FROM scratch

COPY --from=cni /go/src/github.com/containernetworking/plugins/bin/* /bin/
