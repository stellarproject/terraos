# syntax=docker/dockerfile:experimental
ARG KERNEL_VERSION

FROM stellarproject/containerd:latest as containerd
FROM stellarproject/cni:latest as cni
FROM stellarproject/buildkit:latest as buildkit
FROM stellarproject/orbit:latest as orbit
FROM stellarproject/terra:latest as terra
FROM stellarproject/criu:latest as criu
FROM stellarproject/node_exporter:latest as node
FROM stellarproject/kernel:$KERNEL_VERSION as kernel

FROM stellarproject/ubuntu:18.10
ARG KERNEL_VERSION

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-libc-dev_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-image-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb

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
ADD terra /etc/initramfs-tools/scripts/terra
ADD modules /etc/initramfs-tools/modules

# os needs the initrd updated with terra script to boot on its own
RUN update-initramfs -u

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
COPY --from=terra /terra /usr/local/bin/
COPY --from=criu /bin/criu /usr/local/bin/

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
