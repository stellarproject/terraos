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
VERSION=v8
KERNEL=5.0.9

all: clean terra
	@mkdir -p build
	@cd iso && vab build --local --arg KERNEL_VERSION=${KERNEL}
	@mv iso/tftp build/tftp
	@rm -f ./build/terra-${VERSION}.iso
	@cd ./build && ln -s ./tftp/terra.iso terra-${VERSION}.iso

FORCE:

extras: FORCE
	vab build -c extras/cni -d extras/cni -p --ref docker.io/stellarproject/cni:${VERSION}
	vab build -c extras/node_exporter -d extras/node_exporter -p --ref docker.io/stellarproject/node_exporter:${VERSION}
	vab build -c extras/buildkit -d extras/buildkit -p --ref docker.io/stellarproject/buildkit:${VERSION}
	vab build -d extras/criu -c extras/criu -p --ref docker.io/stellarproject/criu:${VERSION}

kernel: FORCE
	vab build --arg KERNEL_VERSION=${KERNEL} -c kernel -d kernel -p --ref docker.io/stellarproject/kernel:${KERNEL}

os: FORCE
	vab build -c os -d os -p --ref docker.io/stellarproject/terraos:${VERSION} --arg KERNEL_VERSION=${KERNEL} --arg VERSION=${VERSION}

terra: FORCE
	@cd cmd && CGO_ENABLED=0 go build -v -ldflags '-s -w -extldflags "-static"' -o ../build/terra
	vab build -p -c cmd -d cmd --ref docker.io/stellarproject/terra:${VERSION}

install:
	@install build/terra /usr/local/sbin/terra

clean:
	@rm -fr build/
