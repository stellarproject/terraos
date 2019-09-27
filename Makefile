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
VERSION=v18-dev
GO_LDFLAGS=-s -w -X github.com/stellarproject/terraos/version.Version=$(VERSION) -X github.com/stellarproject/terraos/version.Revision=$(REVISION)
KERNEL=5.2.15
REPO=$(shell cat REPO || echo "stellarproject")
WIREGUARD=0.0.20190905
VAB_ARGS=""

ARGS=--arg KERNEL_VERSION=${KERNEL} --arg VERSION=${VERSION} --arg REPO=${REPO} --arg WIREGUARD=${WIREGUARD}

release: stage0 stage1 iso

stage0: pxe binaries live boot

stage1: defaults terraos

all: local

clean:
	@rm -fr build/

FORCE:

# -------------------- local -------------------------
local: orbit
	@cd cmd/terra && CGO_ENABLED=0 go build -v -ldflags '${GO_LDFLAGS}' -o ../../build/terra

install:
	@install build/terra* /usr/local/sbin/
	@install build/ob /usr/local/bin/
	@install build/orbit-log /usr/local/bin/
	@install build/orbit-syslog /usr/local/bin/
	@install build/orbit-server /usr/local/bin/
	@install build/orbit-network /usr/local/bin/
	@install cmd/terra/terra /usr/local/sbin/terra-opts

# -------------------- iso -------------------------

iso: clean
	@mkdir -p build
	@cd iso && vab build --local --http ${ARGS}
	@mv iso/terra.iso build/

live:
	@vab build ${VAB_ARGS} --push -c stage1/live -d stage1/live --ref ${REPO}/live:${VERSION} ${ARGS}

# -------------------- stage0 -------------------------

kernel: FORCE
	vab build ${VAB_ARGS} -c stage0/kernel -d stage0/kernel --push --ref ${REPO}/kernel:${KERNEL} ${ARGS}

pxe: FORCE
	vab build ${VAB_ARGS} --push -c stage0/pxe -d stage0/pxe --ref ${REPO}/pxe:${VERSION}  ${ARGS}

boot: FORCE
	@vab build ${VAB_ARGS} --push -c stage0/boot -d stage0/boot --ref ${REPO}/boot:${VERSION}  ${ARGS}

# -------------------- stage1 -------------------------

containerd:
	vab build ${VAB_ARGS} -p -c stage1/defaults/containerd -d stage1/defaults/containerd --ref ${REPO}/containerd:${VERSION} ${ARGS}

defaults: containerd wireguard orbit-release nodeexporter cni FORCE

criu:
	vab build ${VAB_ARGS} -p -d stage1/defaults/criu -c stage1/defaults/criu --ref ${REPO}/criu:${VERSION} ${ARGS}

cni: FORCE
	vab build ${VAB_ARGS} -p -c stage1/defaults/cni -d stage1/defaults/cni --ref ${REPO}/cni:${VERSION} ${ARGS}

nodeexporter:
	vab build ${VAB_ARGS} -p -c stage1/defaults/node_exporter -d stage1/defaults/node_exporter --ref ${REPO}/node_exporter:${VERSION} ${ARGS}

gvisor:
	vab build ${VAB_ARGS} -p -d stage1/defaults/gvisor -c stage1/defaults/gvisor --ref ${REPO}/gvisor:${VERSION} ${ARGS}

diod:
	vab build ${VAB_ARGS} -p -d stage1/extras/diod -c stage1/extras/diod --ref ${REPO}/diod:${VERSION} ${ARGS}

wireguard:
	vab build ${VAB_ARGS} -p -d stage1/defaults/wireguard -c stage1/defaults/wireguard --ref ${REPO}/wireguard:${VERSION} ${ARGS}

extras: buildkit diod FORCE
	vab build ${VAB_ARGS} -p -d stage1/extras/docker -c stage1/extras/docker --ref ${REPO}/docker:${VERSION} ${ARGS}

buildkit: FORCE
	vab build ${VAB_ARGS} -p -c stage1/extras/buildkit -d stage1/extras/buildkit --ref ${REPO}/buildkit:${VERSION} ${ARGS}

terraos: FORCE
	vab build ${VAB_ARGS} -c stage1/terraos -d stage1/terraos --push --ref ${REPO}/terraos:${VERSION} ${ARGS}

binaries:
	vab build ${VAB_ARGS} --push -d cmd --ref ${REPO}/terracmd:${VERSION} ${ARGS}

# ----------------------- ORBIT --------------------------------
protos:
	protobuild --quiet ${PACKAGES}

orbit-release: FORCE
	vab build ${VAB_ARGS} --push --ref ${REPO}/orbit:${VERSION}

orbit: FORCE
	go build -o build/orbit-server -v -ldflags '${GO_LDFLAGS}' github.com/stellarproject/terraos/cmd/orbit-server
	go build -o build/ob -v -ldflags '${GO_LDFLAGS}' github.com/stellarproject/terraos/cmd/ob
	go build -o build/orbit-log -v -ldflags '${GO_LDFLAGS}' github.com/stellarproject/terraos/cmd/orbit-log
	go build -o build/orbit-syslog -v -ldflags '${GO_LDFLAGS}' github.com/stellarproject/terraos/cmd/orbit-syslog
	gcc -static -o build/orbit-network cmd/orbit-network/main.c

example:
	@cd contrib/example && terra create --push server.toml
