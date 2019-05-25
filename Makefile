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

PACKAGES=$(shell go list ./... | grep -v /vendor/)
REVISION=$(shell git rev-parse HEAD)
VERSION=v11
GO_LDFLAGS=-s -w -X github.com/stellarproject/terraos/version.Version=$(VERSION) -X github.com/stellarproject/terraos/version.Revision=$(REVISION)
KERNEL=5.0.18
REPO=stellarproject
WIREGUARD=0.0.20190406

ARGS=--arg KERNEL_VERSION=${KERNEL} --arg VERSION=${VERSION} --arg REPO=${REPO} --arg WIREGUARD=${WIREGUARD}

release: orbit-release cmd defaults os pxe iso

FORCE:

all: iso

iso: clean local live
	@mkdir -p build
	@cd iso && vab build --local ${ARGS}
	@mv iso/tftp build/tftp
	@rm -f ./build/terra-${VERSION}.iso
	@cd ./build && ln -s ./tftp/terra.iso terra-${VERSION}.iso

live: FORCE
	@vab build -p -c live -d live --ref ${REPO}/live:${VERSION} ${ARGS}

pxe: live FORCE
	@vab build -p -c iso -d iso --ref ${REPO}/pxe:${VERSION}  ${ARGS}


defaults: wireguard FORCE
	vab build -p -c defaults/containerd -d defaults/containerd --ref ${REPO}/containerd:${VERSION} ${ARGS}
	vab build -p -c defaults/node_exporter -d defaults/node_exporter --ref ${REPO}/node_exporter:${VERSION} ${ARGS}
	vab build -p -c defaults/cni -d defaults/cni --ref ${REPO}/cni:${VERSION} ${ARGS}
	vab build -p -d defaults/criu -c defaults/criu --ref ${REPO}/criu:${VERSION} ${ARGS}

wireguard:
	vab build -p -d defaults/wireguard -c defaults/wireguard --ref ${REPO}/wireguard:${VERSION} ${ARGS}

extras: FORCE
	vab build -p -c extras/buildkit -d extras/buildkit --ref ${REPO}/buildkit:${VERSION} ${ARGS}
	vab build -p -d extras/docker -c extras/docker --ref ${REPO}/docker:${VERSION} ${ARGS}

kernel: FORCE
	vab build -c kernel -d kernel --push --ref ${REPO}/kernel:${KERNEL} ${ARGS}

os: FORCE
	vab build -c os -d os --push --ref ${REPO}/terraos:${VERSION} ${ARGS}

local: orbit FORCE
	@cd cmd/terra && CGO_ENABLED=0 go build -v -ldflags '${GO_LDFLAGS}' -o ../../build/terra
	@cd cmd/terra-install && CGO_ENABLED=0 go build -v -ldflags '${GO_LDFLAGS}' -o ../../build/terra-install
	@cd cmd/rdns && CGO_ENABLED=0 go build -v -ldflags '${GO_LDFLAGS}' -o ../../build/rdns

cmd: FORCE
	vab build --push -d cmd --ref ${REPO}/terracmd:${VERSION} ${ARGS}

install:
	@install build/terra* /usr/local/sbin/
	@install build/ob /usr/local/bin/
	@install build/orbit-log /usr/local/bin/
	@install build/orbit-server /usr/local/bin/
	@install build/orbit-network /usr/local/bin/

clean:
	@rm -fr build/

# ----------------------- ORBIT --------------------------------
protos:
	protobuild --quiet ${PACKAGES}

orbit-release: FORCE
	vab build -p --ref docker.io/stellarproject/orbit:v10

orbit-latest: FORCE
	vab build -p --ref docker.io/stellarproject/orbit:latest

orbit:
	go build -o build/orbit-server -v -ldflags '${GO_LDFLAGS}' github.com/stellarproject/terraos/cmd/orbit-server
	go build -o build/ob -v -ldflags '${GO_LDFLAGS}' github.com/stellarproject/terraos/cmd/ob
	go build -o build/orbit-log -v -ldflags '${GO_LDFLAGS}' github.com/stellarproject/terraos/cmd/orbit-log
	gcc -static -o build/orbit-network cmd/orbit-network/main.c

example:
	@cd contrib/example && terra create --push server.toml
