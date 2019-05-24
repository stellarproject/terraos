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

FROM golang:1.12 as containerd

RUN apt update && \
	apt install -y \
	rsync \
	autoconf \
	automake \
	g++ \
	libtool \
	unzip \
	btrfs-tools \
	gcc \
	git \
	libseccomp-dev \
	make \
	xfsprogs

RUN git clone https://github.com/containerd/containerd /go/src/github.com/containerd/containerd

WORKDIR /go/src/github.com/containerd/containerd

# RUN rsync -au /go/src/github.com/containerd/containerd/vendor/ /go/src/ && \
#	rm -rf /go/src/github.com/containerd/containerd/vendor/*
RUN rm -rf /go/src/github.com/opencontainers/runc && \
	git clone https://github.com/opencontainers/runc /go/src/github.com/opencontainers/runc

RUN ./script/setup/install-protobuf
RUN ./script/setup/install-runc

RUN make BUILDTAGS='no_cri no_devmapper'

ADD config.toml /etc/containerd/

FROM scratch

COPY --from=containerd /etc/containerd/config.toml /etc/containerd/
COPY --from=containerd /go/src/github.com/containerd/containerd/containerd.service /etc/systemd/system/
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/* /usr/local/bin/
COPY --from=containerd /usr/local/sbin/runc /usr/local/sbin/
