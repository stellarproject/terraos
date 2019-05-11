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

REVISION=$(shell git rev-parse HEAD)
VERSION=v10
GO_LDFLAGS=-s -w -X github.com/stellarproject/terraos/version.Version=$(VERSION) -X github.com/stellarproject/terraos/version.Revision=$(REVISION)
KERNEL=5.1.0

all: clean local
	@mkdir -p build
	@cd iso && vab build --local --arg KERNEL_VERSION=${KERNEL} --arg VERSION=${VERSION}
	@mv iso/tftp build/tftp
	@rm -f ./build/terra-${VERSION}.iso
	@cd ./build && ln -s ./tftp/terra.iso terra-${VERSION}.iso

FORCE:

extras: FORCE
	vab build -c extras/containerd -d extras/containerd --ref stellarproject/containerd:${VERSION}
	vab build -c extras/cni -d extras/cni --ref stellarproject/cni:${VERSION}
	vab build -c extras/node_exporter -d extras/node_exporter --ref stellarproject/node_exporter:${VERSION}
	vab build -c extras/buildkit -d extras/buildkit --ref stellarproject/buildkit:${VERSION}
	vab build -d extras/criu -c extras/criu --ref stellarproject/criu:${VERSION}

kernel: FORCE
	vab build --arg KERNEL_VERSION=${KERNEL} -c kernel -d kernel --push --ref docker.io/stellarproject/kernel:${KERNEL}

os: FORCE
	vab build -c os -d os --push --ref docker.io/stellarproject/terraos:${VERSION} --arg KERNEL_VERSION=${KERNEL} --arg VERSION=${VERSION}

local: FORCE
	@cd cmd/terra && CGO_ENABLED=0 go build -v -ldflags '${GO_LDFLAGS}' -o ../../build/terra
	@cd cmd/vab && CGO_ENABLED=0 go build -v -ldflags '${GO_LDFLAGS}' -o ../../build/vab
	@cd cmd/rdns && CGO_ENABLED=0 go build -v -ldflags '${GO_LDFLAGS}' -o ../../build/rdns

cmd: FORCE
	vab build --push -d cmd --ref stellarproject/terracmd:${VERSION}

install:
	@install build/terra /usr/local/sbin/terra
	@install build/vab /usr/local/bin/vab

clean:
	@rm -fr build/
