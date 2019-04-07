FROM ubuntu:18.10 as builder
ARG KERNEL_VERSION

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
	make

ENV WIREGUARD_VERSION=0.0.20190227
ENV WIREGUARD_URL=https://git.zx2c4.com/WireGuard/snapshot/WireGuard-${WIREGUARD_VERSION}.tar.xz

ADD https://cdn.kernel.org/pub/linux/kernel/v5.x/linux-${KERNEL_VERSION}.tar.xz linux.tar.xz

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
RUN make -j "$(getconf _NPROCESSORS_ONLN)" KDEB_PKGVERSION=$PKGVERSION INSTALL_MOD_STRIP=1 bindeb-pkg

RUN dpkg -i \
	/linux-headers-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb \
	/linux-libc-dev_$KERNEL_VERSION-terra-1_amd64.deb \
	/linux-image-$KERNEL_VERSION-terra_$KERNEL_VERSION-terra-1_amd64.deb

FROM scratch

COPY --from=builder /*.deb /
