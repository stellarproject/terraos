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

FROM ubuntu:19.04 as builder
ARG KERNEL_VERSION
ARG WIREGUARD

RUN apt-get update && apt-get install -y \
	build-essential \
	bc \
	devscripts \
	xz-utils \
	wget \
	curl \
	ca-certificates \
	bison \
	flex \
	cpio \
	libelf-dev \
	kmod \
	libssl-dev \
	git \
	libmnl-dev \
	btrfs-progs \
	btrfs-tools \
	libssl-dev \
	make

ENV WIREGUARD_URL=https://git.zx2c4.com/WireGuard/snapshot/WireGuard-${WIREGUARD}.tar.xz

RUN curl -SsL -o linux.tar.xz https://cdn.kernel.org/pub/linux/kernel/v5.x/linux-5.2.tar.xz

RUN mkdir /linux && \
	tar --strip-components=1 -xpf linux.tar.xz -C /linux

ADD config /linux/.config

# WireGuard
RUN curl -SsL "${WIREGUARD_URL}" -o /wireguard.tar.xz

RUN mkdir /wireguard && \
	tar --strip-components=1 -xpf wireguard.tar.xz -C /wireguard

# patches

ADD patches /patches
RUN bash -c 'cd /linux; for i in $(cd /patches; ls | grep patch); do echo "applying $i"; patch -Np1 -i /patches/$i ; done'

WORKDIR /linux

RUN /wireguard/contrib/kernel-tree/create-patch.sh | patch -p1

RUN make olddefconfig
RUN make -j "$(getconf _NPROCESSORS_ONLN)" KDEB_PKGVERSION=$PKGVERSION INSTALL_MOD_STRIP=1 bindeb-pkg

FROM scratch

COPY --from=builder /*.deb /
