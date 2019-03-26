# ---------------------- containerd build -------------------
FROM golang:1.12 as containerd

RUN apt update && \
	apt install -y \
	rsync \
	autoconf \
	automake \
	g++ \
	libtool \
	unzip \
	btrfs-tools \
	gcc \
	git \
	libseccomp-dev \
	make \
	xfsprogs

ENV CONTAINERD_VERSION 7f9eb54626ccf0fbccc96a22ff42677054cc1534
RUN git clone https://github.com/crosbymichael/containerd /go/src/github.com/containerd/containerd
RUN git clone https://github.com/opencontainers/runc /go/src/github.com/opencontainers/runc

WORKDIR /go/src/github.com/containerd/containerd
RUN git checkout ${CONTAINERD_VERSION}

RUN rsync -au /go/src/github.com/containerd/containerd/vendor/ /go/src/ && \
	rm -rf /go/src/github.com/containerd/containerd/vendor/

RUN ./script/setup/install-protobuf
RUN ./script/setup/install-runc
RUN make

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

RUN make && make plugin

# -------------------- production image ------------------------
FROM ubuntu:18.10

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
COPY --from=orbit /go/src/github.com/stellarproject/orbit/bin/* /usr/local/bin/
COPY --from=orbit /go/src/github.com/stellarproject/orbit/plugins/* /var/lib/containerd/plugins/
RUN ln -s /usr/local/bin/orbit-network /opt/containerd/bin/orbit-network

ADD *.deb /

ONBUILD dpkg -i /linux-headers.deb /linux-libc.deb /linux-image.deb
# ONBUILD apt update && apt upgrade -y
ONBUILD rm /etc/resolv.conf
# reload ld lib cache
ONBUILD rm /etc/ld.so.cache && ldconfig -v

ONBUILD systemctl disable systemd-resolved
ONBUILD systemctl enable containerd
