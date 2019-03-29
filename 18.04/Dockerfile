# ---------------------- continerd build -------------------
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

# ------------------ orbit build -------------------------
FROM containerd as orbit

RUN git clone https://github.com/stellarproject/orbit /go/src/github.com/stellarproject/orbit

RUN	rm -rf /go/src/github.com/stellarproject/orbit/vendor/github.com/containerd/ && \
	rsync -au --ignore-existing /go/src/github.com/stellarproject/orbit/vendor/ /go/src/ && \
	rm -rf /go/src/github.com/stellarproject/orbit/vendor/

WORKDIR /go/src/github.com/stellarproject/orbit

RUN make

# -------------------- production image ------------------------
FROM ubuntu:18.04

# containerd binaries
COPY --from=containerd /go/src/github.com/containerd/containerd/containerd.service /lib/systemd/system/
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/* /usr/local/bin/
COPY --from=containerd /usr/local/sbin/runc /usr/local/sbin/

# orbit binaries
COPY --from=orbit /go/src/github.com/stellarproject/orbit/bin/o* /usr/local/bin/
COPY --from=orbit /go/src/github.com/stellarproject/orbit/bin/plugins/* /var/lib/containerd/plugins/

ONBUILD systemctl enable containerd
ONBUILD systemctl disable systemd-resolved

ONBUILD rm /etc/ld.so.cache && ldconfig -v
ONBUILD rm /etc/resolv.conf && echo 'nameserver 127.0.0.1' > /etc/resolv.conf
