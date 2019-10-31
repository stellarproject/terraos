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
ARG REPO

FROM $REPO/terracmd:$VERSION as terra
FROM $REPO/pxe:$VERSION as pxe

FROM alpine:latest

RUN mkdir -p /boot /sd

RUN apk update && \
	apk add --no-cache \
	e2fsprogs \
	xfsprogs \
	btrfs-progs \
	dhclient \
	openssh \
	ca-certificates \
	iptables \
	gparted \
	sgdisk \
	zfs \
	syslinux

COPY --from=terra / /
COPY --from=pxe /tftp /boot
COPY --from=pxe /lib/modules/ /lib/modules/
RUN rm -rf /boot/*.c32 /boot/pxelinux.0 /boot/pxelinux.cfg

ADD interfaces /etc/network/interfaces
RUN rm /sbin/init
ADD resolv.conf /etc/resolv.conf
ADD init /sbin/init
RUN chmod +x /sbin/init

RUN ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa && \
	ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa && \
	echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
RUN echo 'root:root' | chpasswd
