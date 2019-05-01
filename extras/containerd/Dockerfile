ARG VERSION

FROM docker.io/stellarproject/containerd-build:${VERSION} as containerd

FROM scratch

COPY --from=containerd /etc/containerd/config.toml /etc/containerd/
COPY --from=containerd /go/src/github.com/containerd/containerd/containerd.service /etc/systemd/system/
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/* /usr/local/bin/
COPY --from=containerd /usr/local/sbin/runc /usr/local/sbin/
