FROM golang:1.12 as buildkit

RUN git clone https://github.com/moby/buildkit /go/src/github.com/moby/buildkit

WORKDIR /go/src/github.com/moby/buildkit/cmd/buildkitd
RUN go build

FROM scratch

COPY --from=buildkit /go/src/github.com/moby/buildkit/cmd/buildkitd/buildkitd /bin/
