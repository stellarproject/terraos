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

# ------------------ imports ------------------------------ #
FROM $REPO/terracmd:$VERSION as terra
FROM $REPO/pxe:$VERSION as pxe

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
	syslinux-utils

COPY --from=pxe / /

RUN mkisofs -o /terra.iso \
   -b syslinux/isolinux.bin -c syslinux/boot.cat \
   -no-emul-boot -boot-load-size 4 -boot-info-table \
   /boot

RUN isohybrid /terra.iso

FROM scratch

COPY --from=iso /terra.iso /
