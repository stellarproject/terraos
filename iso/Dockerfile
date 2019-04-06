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

RUN mkdir -p /boot/syslinux

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-libc-dev_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-image-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb

RUN cp /syslinux-6.03/bios/core/isolinux.bin /boot/syslinux
RUN cp /syslinux-6.03/bios/mbr/mbr.bin /boot/syslinux
RUN cp /syslinux-6.03/bios/com32/elflink/ldlinux/ldlinux.c32 /boot/syslinux
RUN cp /syslinux-6.03/bios/com32/menu/vesamenu.c32 /boot/syslinux/
RUN cp /syslinux-6.03/bios/com32/menu/menu.c32 /boot/syslinux/
RUN cp /syslinux-6.03/bios/com32/chain/chain.c32 /boot/syslinux/
RUN cp /syslinux-6.03/bios/com32/libutil/libutil.c32 /boot/syslinux/
RUN cp /syslinux-6.03/bios/com32/lib/libcom32.c32 /boot/syslinux/

ADD live /etc/initramfs-tools/scripts/live

RUN update-initramfs -u

RUN mv /boot/vmlinuz-$KERNEL_VERSION-terra /boot/vmlinuz
RUN mv /boot/initrd.img-$KERNEL_VERSION-terra /boot/initrd.img

ADD syslinux.cfg /boot/syslinux/syslinux.cfg
ADD splash.png /boot/syslinux/splash.png

# COPY boot.txt /boot/boot.txt
COPY --from=live /live.sqsh /boot/live.sq
COPY --from=live /os.tgz /boot/os.tgz

RUN mkisofs -o /terra.iso \
   -b syslinux/isolinux.bin -c syslinux/boot.cat \
   -no-emul-boot -boot-load-size 4 -boot-info-table \
   /boot

FROM scratch

COPY --from=iso /terra.iso .
