GO_LDFLAGS=-s -w

all:
	vab build --arg KERNEL_VERSION=5.0.5 --local -c iso -d iso

FORCE:

os: FORCE
	vab build -c os -d os --arg KERNEL_VERSION=5.0.6 -p --ref docker.io/stellarproject/terra:2

containerd: FORCE
	vab build -c containerd -d containerd -p --ref docker.io/stellarproject/containerd:latest

extras: FORCE
	vab build -c cni -d cni -p --ref docker.io/stellarproject/cni:latest
	vab build -c node_exporter -d node_exporter -p --ref docker.io/stellarproject/node_exporter:latest
	vab build -c buildkit -d buildkit -p --ref docker.io/stellarproject/buildkit:latest

kernel: FORCE
	vab build -c kernel -d kernel -p --ref docker.io/stellarproject/kernel:5.0.6

base: FORCE
	vab build -c base -d base -p --ref docker.io/stellarproject/ubuntu:18.10

live: FORCE
	vab build -c live -d live --ref docker.io/stellarproject/live:latest

terra: FORCE
	vab build -d terra -p --ref docker.io/stellarproject/terra:latest
