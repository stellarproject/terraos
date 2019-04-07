VERSION=2
KERNEL=5.0.7

all: terra
	vab build --arg KERNEL_VERSION=5.0.5 --local -c iso -d iso

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
