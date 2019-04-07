# syntax=docker/dockerfile:experimental
ARG KERNEL_VERSION

FROM stellarproject/kernel:$KERNEL_VERSION as kernel

FROM stellarproject/ubuntu:18.10 as boot
ARG KERNEL_VERSION

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-image-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb

RUN apt update && \
	DEBIAN_FRONTEND=noninteractive apt install -y \
	initramfs-tools

ADD terra /etc/initramfs-tools/scripts/terra
ADD modules /etc/initramfs-tools/modules

RUN update-initramfs -u

FROM scratch

COPY --from=boot /boot /boot
