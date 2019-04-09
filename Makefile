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

VERSION=2
KERNEL=5.0.7

all: terra
	vab build --arg KERNEL_VERSION=${KERNEL} --local -c iso -d iso

FORCE:

boot: FORCE
	vab build -p -c boot -d boot --arg KERNEL_VERSION=${KERNEL} --ref docker.io/stellarproject/boot:${VERSION}

os: boot FORCE
	vab build -c os -d os --arg KERNEL_VERSION=${KERNEL} -p --ref docker.io/stellarproject/terraos:${VERSION}

containerd: FORCE
	vab build -c containerd -d containerd -p --ref docker.io/stellarproject/containerd:latest

extras: FORCE
	vab build -c cni -d cni -p --ref docker.io/stellarproject/cni:latest
	vab build -c node_exporter -d node_exporter -p --ref docker.io/stellarproject/node_exporter:latest
	vab build -c buildkit -d buildkit -p --ref docker.io/stellarproject/buildkit:latest
	vab build -d criu -c criu -p --ref docker.io/stellarproject/criu:latest

kernel: FORCE
	vab build --arg KERNEL_VERSION=${KERNEL} -c kernel -d kernel -p --ref docker.io/stellarproject/kernel:${KERNEL}

base: FORCE
	vab build -c base -d base -p --ref docker.io/stellarproject/ubuntu:18.10

terra: FORCE
	vab build -p -d terra --ref docker.io/stellarproject/terra:latest
