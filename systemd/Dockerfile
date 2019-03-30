FROM ubuntu:18.10

RUN apt update && \
	apt install -y \
	build-essential \
	ca-certificates \
	bison \
	git \
	meson \
	gperf \
	make \
	libssl-dev \
	libcap-dev \
	pkg-config \
	mount \
	libmount-dev \
	gettext \
	libapparmor-dev \
	libpam-dev \
	libaudit-dev \
	libcryptsetup-dev \
	libgcrypt-dev \
	libkmod-dev \
	libacl1-dev \
	libseccomp-dev \
	liblz-dev \
	libidn2-dev \
	libcurl4-openssl-dev

ENV SYSTEMD_VERSION v240
RUN git clone https://github.com/systemd/systemd /systemd

WORKDIR /systemd
RUN git checkout ${SYSTEMD_VERSION}

RUN ./configure && \
	make
