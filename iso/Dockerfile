# syntax=docker/dockerfile:experimental
ARG KERNEL_VERSION

FROM stellarproject/live:latest as live
FROM docker.io/stellarproject/kernel:$KERNEL_VERSION as kernel
FROM ubuntu:18.10 as iso
ARG KERNEL_VERSION

RUN apt-get update && apt-get install -y --no-install-recommends \
	syslinux \
	mkisofs \
	curl \
	ca-certificates \
	xz-utils \
	xorriso \
	initramfs-tools
RUN curl -sSL https://mirrors.edge.kernel.org/pub/linux/utils/boot/syslinux/syslinux-6.03.tar.xz -o /tmp/syslinux.tar.xz && \
	tar xf /tmp/syslinux.tar.xz -C /
RUN mkdir -p /boot/isolinux
RUN cp /syslinux-6.03/bios/core/isolinux.bin /boot
RUN cp /syslinux-6.03/bios/com32/elflink/ldlinux/ldlinux.c32 /boot
RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-libc-dev_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-image-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb
ADD terra /etc/initramfs-tools/scripts/terra
ADD live /etc/initramfs-tools/scripts/live
ADD modules /etc/initramfs-tools/modules
RUN update-initramfs -u

RUN mv /boot/vmlinuz-$KERNEL_VERSION-terra /boot/vmlinuz
RUN mv /boot/initrd.img-$KERNEL_VERSION-terra /boot/initrd.img
COPY isolinux.cfg /boot/isolinux.cfg
COPY boot.txt /boot/boot.txt
COPY --from=live /live.sqsh /boot/root.squashfs

RUN mkisofs -o /terra.iso \
   -b isolinux.bin -c boot.cat \
   -no-emul-boot -boot-load-size 4 -boot-info-table \
   /boot

FROM scratch
COPY --from=iso /terra.iso .
