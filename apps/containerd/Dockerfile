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

FROM golang:alpine as containerd

RUN apk update && \
	apk add \
	autoconf \
	automake \
	g++ \
	libtool \
	libseccomp-dev \
	curl \
	unzip \
	gcc \
	git \
	make

ENV COMMIT 93391048bd2cc8dc4921071407bb88e7a8fdf30b
RUN git clone https://github.com/containerd/containerd /go/src/github.com/containerd/containerd
WORKDIR /go/src/github.com/containerd/containerd
RUN git checkout $COMMIT

RUN rm -rf /go/src/github.com/opencontainers/runc && \
	git clone https://github.com/opencontainers/runc /go/src/github.com/opencontainers/runc

ADD install-runc ./script/setup/install-runc
RUN ./script/setup/install-protobuf
RUN ./script/setup/install-runc

RUN make BUILDTAGS='no_cri no_devmapper no_btrfs'

ADD config.toml /etc/containerd/

FROM scratch

COPY --from=containerd /etc/containerd/config.toml /etc/containerd/
COPY --from=containerd /go/src/github.com/containerd/containerd/containerd.service /etc/systemd/system/
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/* /usr/local/bin/
COPY --from=containerd /usr/local/sbin/runc /usr/local/sbin/
ADD containerd /etc/init.d/containerd
