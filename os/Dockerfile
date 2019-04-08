# Copyright (c) 2019 Stellar Project

# Permission is hereby granted, free of charge, to any person
# obtaining a copy of this software and associated documentation
# files (the "Software"), to deal in the Software without
# restriction, including without limitation the rights to use, copy,
# modify, merge, publish, distribute, sublicense, and/or sell copies
# of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED,
# INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
# IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
# HOLDERS BE LIABLE FOR ANY CLAIM,
# DAMAGES OR OTHER LIABILITY,
# WHETHER IN AN ACTION OF CONTRACT,
# TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH
# THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

# syntax=docker/dockerfile:experimental
ARG KERNEL_VERSION

FROM stellarproject/containerd:latest as containerd
FROM stellarproject/cni:latest as cni
FROM stellarproject/buildkit:latest as buildkit
FROM stellarproject/orbit:latest as orbit
FROM stellarproject/vab:latest as vab
FROM stellarproject/criu:latest as criu
FROM stellarproject/node_exporter:latest as node
FROM stellarproject/kernel:$KERNEL_VERSION as kernel

FROM stellarproject/ubuntu:18.10
ARG KERNEL_VERSION

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-libc-dev_$KERNEL_VERSION-terra-1_amd64.deb

RUN apt update && \
	DEBIAN_FRONTEND=noninteractive apt install -y \
	openssh-server \
	apparmor \
    libnl-3-dev \
    libnet-dev \
    libprotobuf-dev \
	initramfs-tools \
	pigz

ADD config.toml /etc/containerd/

# containerd binaries
COPY --from=containerd /go/src/github.com/containerd/containerd/containerd.service /etc/systemd/system/
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/* /usr/local/bin/
COPY --from=containerd /usr/local/sbin/runc /usr/local/sbin/

# additional directories
RUN mkdir -p /opt/containerd/bin && \
	mkdir -p /opt/containerd/plugins && \
	mkdir -p /etc/netplan

COPY --from=cni /bin/* /opt/containerd/bin/

# orbit binaries
COPY --from=orbit /bin/o* /usr/local/bin/
COPY --from=orbit /bin/plugins/* /opt/containerd/plugins/
RUN ln -s /usr/local/bin/orbit-network /opt/containerd/bin/orbit-network

COPY --from=node /bin/* /usr/local/bin/
COPY --from=buildkit /bin/* /usr/local/bin/
COPY --from=criu /bin/criu /usr/local/bin/
COPY --from=vab /bin/vab /usr/local/bin/

ADD buildkit.service /etc/systemd/system/
ADD node_exporter.service /etc/systemd/system/

RUN \
	systemctl disable systemd-resolved && \
	systemctl enable containerd && \
	systemctl enable ssh && \
	systemctl enable buildkit && \
	systemctl enable node_exporter && \
	systemctl enable serial-getty@ttyS0.service

# motd
RUN rm /etc/update-motd.d/*
ADD motd/* /etc/update-motd.d/

ADD sshd_config /etc/ssh/

CMD ["/sbin/init"]
