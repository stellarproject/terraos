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

ARG KERNEL_VERSION
ARG VERSION
ARG REPO

# ------------------ imports ------------------------------ #
FROM $REPO/terracmd:$VERSION as terra
FROM $REPO/kernel:$KERNEL_VERSION as kernel

# ------------------ live liveos -------------------------- #
FROM alpine:latest as liveos

RUN mkdir -p /boot /sd

RUN apk update && \
	apk add \
	e2fsprogs \
	xfsprogs \
	btrfs-progs \
	grub \
	dhclient \
	openssh \
	ca-certificates \
	grub-bios

COPY --from=terra / /
ADD interfaces /etc/network/interfaces
RUN rm /sbin/init
ADD resolv.conf /etc/resolv.conf
ADD init /sbin/init
RUN chmod +x /sbin/init && \
	touch /etc/default/grub

RUN ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa && \
	ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa && \
	echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
RUN echo 'root:root' | chpasswd

# ------------------ squasher ---------------------------- #
FROM ubuntu:18.10 as squasher
ARG KERNEL_VERSION

RUN apt update && \
	apt install -y squashfs-tools

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-image-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb

COPY --from=liveos / /rootfs
RUN mkdir -p /rootfs/proc /rootfs/sys /rootfs/run && \
	cp -R /lib/modules /rootfs/lib/ && \
	mksquashfs /rootfs live.sqsh

# ------------------ iso build --------------------------- #
FROM ubuntu:18.10 as iso
ARG KERNEL_VERSION

RUN apt-get update && apt-get install -y \
	syslinux \
	mkisofs \
	curl \
	ca-certificates \
	xz-utils \
	xorriso \
	syslinux-utils \
	open-iscsi \
	isc-dhcp-common \
	isc-dhcp-client \
	initramfs-tools

RUN --mount=type=bind,from=kernel,target=/tmp dpkg -i \
	/tmp/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-libc-dev_$KERNEL_VERSION-terra-1_amd64.deb \
	/tmp/linux-image-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb

RUN curl -sSL https://mirrors.edge.kernel.org/pub/linux/utils/boot/syslinux/syslinux-6.03.tar.xz -o /tmp/syslinux.tar.xz && \
	tar xf /tmp/syslinux.tar.xz -C /

RUN mkdir -p /boot/syslinux

RUN cp /syslinux-6.03/bios/core/isolinux.bin /boot/syslinux && \
	cp /syslinux-6.03/bios/mbr/mbr.bin /boot/syslinux && \
	cp /syslinux-6.03/bios/com32/elflink/ldlinux/ldlinux.c32 /boot/syslinux && \
	cp /syslinux-6.03/bios/com32/menu/vesamenu.c32 /boot/syslinux/ && \
	cp /syslinux-6.03/bios/com32/menu/menu.c32 /boot/syslinux/ && \
	cp /syslinux-6.03/bios/com32/chain/chain.c32 /boot/syslinux/ && \
	cp /syslinux-6.03/bios/com32/libutil/libutil.c32 /boot/syslinux/ && \
	cp /syslinux-6.03/bios/com32/lib/libcom32.c32 /boot/syslinux/ && \
	cp /syslinux-6.03/bios/com32/modules/sanboot.c32 /boot/syslinux/

ADD live /etc/initramfs-tools/scripts/live
ADD squashhook /etc/initramfs-tools/hooks/squash
RUN chmod +x /etc/initramfs-tools/hooks/squash
COPY --from=squasher /live.sqsh /live.sq
ADD terra /etc/initramfs-tools/scripts/terra
ADD btrfs /etc/initramfs-tools/scripts/btrfs
ADD modules /etc/initramfs-tools/modules
RUN touch /etc/iscsi/iscsi.initramfs

RUN update-initramfs -u

RUN mv /boot/vmlinuz-$KERNEL_VERSION-terra /boot/vmlinuz
RUN mv /boot/initrd.img-$KERNEL_VERSION-terra /boot/initrd.img

ADD syslinux.cfg /boot/syslinux/syslinux.cfg
ADD splash.png /boot/syslinux/splash.png

RUN mkisofs -o /terra.iso \
   -b syslinux/isolinux.bin -c syslinux/boot.cat \
   -no-emul-boot -boot-load-size 4 -boot-info-table \
   /boot

RUN isohybrid /terra.iso

# ------------------ pxe ------------------------------ #
FROM alpine:latest as pxe

RUN apk add -U syslinux

RUN mkdir /tftp && \
	cp /usr/share/syslinux/pxelinux.0 /tftp/ && \
	cp -r /usr/share/syslinux/libmenu.c32 \
		/usr/share/syslinux/menu.c32 \
		/usr/share/syslinux/sanboot.c32 \
		/usr/share/syslinux/memdisk \
		/usr/share/syslinux/ldlinux.c32 \
		/usr/share/syslinux/libutil.c32 /tftp/ && \
	mkdir /tftp/pxelinux.cfg

ADD default /tftp/pxelinux.cfg/default

# ------------------ build results --------------------- #
FROM scratch

COPY --from=pxe /tftp /tftp
COPY --from=iso /terra.iso /tftp/
COPY --from=iso /boot/vmlinuz /tftp/
COPY --from=iso /boot/initrd.img /tftp/
# ------------------ build results --------------------- #
