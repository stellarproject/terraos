FROM crosbymichael/containerd:latest as containerd

# ---------------------- cni build -------------------
FROM golang:1.12 as cni

RUN go get -d github.com/containernetworking/plugins || true
WORKDIR /go/src/github.com/containernetworking/plugins

RUN ./build_linux.sh

# ------------------ orbit build -------------------------
FROM containerd as orbit

ADD orbit/ /go/src/github.com/stellarproject/orbit

RUN	rm -rf /go/src/github.com/stellarproject/orbit/vendor/github.com/containerd/ && \
	rsync -au --ignore-existing /go/src/github.com/stellarproject/orbit/vendor/ /go/src/ && \
	rm -rf /go/src/github.com/stellarproject/orbit/vendor/

WORKDIR /go/src/github.com/stellarproject/orbit

RUN make

# -------------------- production image ------------------------
FROM ubuntu:18.10

RUN apt update && \
	apt install -y \
	pigz

ADD config.toml /etc/containerd/

# containerd binaries
COPY --from=containerd /go/src/github.com/containerd/containerd/containerd.service /lib/systemd/system/
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/* /usr/local/bin/
COPY --from=containerd /usr/local/sbin/runc /usr/local/sbin/

# cni binaries
RUN mkdir -p /opt/containerd/bin
COPY --from=cni /go/src/github.com/containernetworking/plugins/bin/macvlan /opt/containerd/bin/
COPY --from=cni /go/src/github.com/containernetworking/plugins/bin/loopback /opt/containerd/bin/
COPY --from=cni /go/src/github.com/containernetworking/plugins/bin/bridge /opt/containerd/bin/
COPY --from=cni /go/src/github.com/containernetworking/plugins/bin/host-local /opt/containerd/bin/

# orbit binaries
COPY --from=orbit /go/src/github.com/stellarproject/orbit/bin/o* /usr/local/bin/
COPY --from=orbit /go/src/github.com/stellarproject/orbit/bin/plugins/* /var/lib/containerd/plugins/
RUN ln -s /usr/local/bin/orbit-network /opt/containerd/bin/orbit-network

ADD *.deb /

ONBUILD dpkg -i /linux-headers.deb /linux-libc.deb /linux-image.deb
# ONBUILD apt update && apt upgrade -y
ONBUILD rm /etc/resolv.conf
# reload ld lib cache
ONBUILD rm /etc/ld.so.cache && ldconfig -v

ONBUILD systemctl disable systemd-resolved
ONBUILD systemctl enable containerd
