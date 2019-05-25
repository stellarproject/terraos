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
