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

REVISION=$(shell git rev-parse HEAD)$(shell if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)
VERSION=v7
KERNEL=5.0.8

all: iso
	terra os "releases/${VERSION}.toml"

FORCE:

iso: terra FORCE
	cd iso && vab build --local --arg KERNEL_VERSION=${KERNEL}
	mv iso/terra.iso "terra-${VERSION}.iso"

containerd-build: FORCE
	vab build -c extras/containerd-build -d extras/containerd-build -p --ref docker.io/stellarproject/containerd-build:latest

extras: containerd-build FORCE
	vab build -c extras/containerd -d extras/containerd -p --ref docker.io/stellarproject/containerd:latest
	vab build -c extras/cni -d extras/cni -p --ref docker.io/stellarproject/cni:latest
	vab build -c extras/node_exporter -d extras/node_exporter -p --ref docker.io/stellarproject/node_exporter:latest
	vab build -c extras/buildkit -d extras/buildkit -p --ref docker.io/stellarproject/buildkit:latest
	vab build -d extras/criu -c extras/criu -p --ref docker.io/stellarproject/criu:latest

kernel: FORCE
	vab build --arg KERNEL_VERSION=${KERNEL} -c kernel -d kernel -p --ref docker.io/stellarproject/kernel:${KERNEL}

base: FORCE
	vab build -c base -d base -p --ref docker.io/stellarproject/ubuntu:18.10

terra: FORCE
	vab build -p -c terra -d terra --ref docker.io/stellarproject/terra:latest

pxe: iso FORCE
	@cd pxe && vab build --local
	@mv pxe/tftp tftp
	@cp terra-${VERSION}.iso tftp/terra.iso
