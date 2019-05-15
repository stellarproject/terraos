# syntax=docker/dockerfile:experimental

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

ARG VERSION
ARG KERNEL_VERSION

FROM docker.io/stellarproject/kernel:${KERNEL_VERSION} as kernel
FROM docker.io/stellarproject/orbit:${VERSION} as orbit
FROM docker.io/stellarproject/containerd:${VERSION} as containerd
FROM docker.io/stellarproject/ion:v8 as ion

FROM stellarproject/terracmd:${VERSION} as terra
FROM stellarproject/node_exporter:${VERSION} as node_exporter
FROM stellarproject/criu:${VERSION} as criu
FROM stellarproject/cni:${VERSION} as cni

FROM ubuntu:19.04 as debootstrap

RUN apt update && \
	apt install -y \
	debootstrap

RUN debootstrap --arch amd64 disco /terra/ http://archive.ubuntu.com/ubuntu

# main OS image
FROM scratch
ARG KERNEL_VERSION

COPY --from=debootstrap /terra/ /

# add full apt sources
ADD sources.list /etc/apt/

RUN apt update && apt upgrade -y && \
	DEBIAN_FRONTEND=noninteractive apt install -y \
	ca-certificates \
	openssh-server \
	curl \
	pigz \
	git \
	htop \
	tmux \
	apparmor \
	libnl-3-dev \
	libnet-dev \
	libprotobuf-dev \
	haveged && \
	rm -rf /var/lib/apt/lists/*

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
        /tmp/linux-headers-${KERNEL_VERSION}-terra_${KERNEL_VERSION}-terra-1_amd64.deb \
        /tmp/linux-image-${KERNEL_VERSION}-terra_${KERNEL_VERSION}-terra-1_amd64.deb \
        /tmp/linux-libc-dev_${KERNEL_VERSION}-terra-1_amd64.deb && \
        cp /tmp/wg* /usr/local/bin/

ADD 01-netcfg.yaml  /etc/netplan

RUN rm /etc/fstab

# add the default terra user
RUN addgroup --system terra && \
	useradd -s /bin/bash -g terra --groups adm --groups cdrom --groups sudo --groups dip --groups plugdev -m -k /dev/null terra && \
	chown -R terra:terra /home/terra

ADD sudoers /etc/
ADD hosts /etc/
ADD hostname /etc/
ADD resolv.conf /etc/

RUN chmod 0440 /etc/sudoers

ADD sshd_config /etc/ssh/
RUN systemctl disable systemd-resolved

RUN rm -f /etc/machine-id && \
	rm -f /var/lib/dbus/machine-id

RUN rm /etc/update-motd.d/*
ADD motd/* /etc/update-motd.d/

COPY --from=terra / /
COPY --from=node_exporter / /
COPY --from=cni / /
COPY --from=ion / /
COPY --from=orbit / /
COPY --from=containerd / /
COPY --from=criu / /

RUN systemctl enable containerd node_exporter ssh orbit

CMD ["/sbin/init"]
