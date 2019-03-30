# syntax=docker/dockerfile:experimental
ARG KERNEL_VERSION

FROM stellarproject/containerd:latest as containerd
FROM stellarproject/cni:latest as cni
FROM stellarproject/buildkit:latest as buildkit
FROM stellarproject/orbit:latest as orbit
FROM stellarproject/node_exporter:latest as node
FROM stellarproject/kernel:$KERNEL_VERSION as kernel

FROM stellarproject/ubuntu:18.10
ARG KERNEL_VERSION

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-libc-dev_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-image-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb

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
COPY --from=cni /bin/* /opt/containerd/bin/

# orbit binaries
COPY --from=orbit /bin/* /usr/local/bin/
COPY --from=orbit /plugins/* /var/lib/containerd/plugins/
RUN ln -s /usr/local/bin/orbit-network /opt/containerd/bin/orbit-network

COPY --from=node /bin/* /usr/local/bin/
COPY --from=buildkit /bin/* /usr/local/bin/

RUN \
	systemctl disable systemd-resolved && \
	systemctl enable containerd

ONBUILD rm /etc/ld.so.cache && ldconfig -v
ONBUILD update-grub

CMD ["/sbin/init"]
