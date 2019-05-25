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

FROM ubuntu:18.10 as builder
ARG WIREGUARD

RUN apt-get update && apt-get install -y \
	build-essential \
	curl \
	ca-certificates \
	cpio \
	libelf-dev \
	libssl-dev \
	libmnl-dev \
	make

ENV WIREGUARD_URL=https://git.zx2c4.com/WireGuard/snapshot/WireGuard-${WIREGUARD}.tar.xz

# WireGuard
RUN curl -SsL "${WIREGUARD_URL}" -o /wireguard.tar.xz

RUN mkdir /wireguard && \
	tar --strip-components=1 -xpf wireguard.tar.xz -C /wireguard

# compile wg(8)
WORKDIR /wireguard/src
RUN make tools && make tools-install

FROM scratch

COPY --from=builder /wireguard/src/tools/wg /usr/local/bin/wg
COPY --from=builder /wireguard/src/tools/wg-quick/linux.bash /usr/local/bin/wg-quick
ADD wg-quick@.service /etc/systemd/system/
