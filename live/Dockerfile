# syntax=docker/dockerfile:experimental

FROM stellarproject/terra:2 as os

FROM alpine:latest as rootfs

RUN mkdir -p /boot

RUN apk update && \
	apk add -y \
	e2fsprogs \
	xfsprogs \
	btrfs-progs \
	grub \
	grub-bios

ADD grub /etc/default/grub
ADD install-terra /bin/
RUN chmod +x /bin/install-terra

FROM ubuntu:18.10 as squash

RUN apt update && \
	apt install -y squashfs-tools

RUN --mount=type=bind,from=rootfs,target=/rootfs,rw \
	mkdir -p /rootfs/proc /rootfs/sys /rootfs/run && \
	mksquashfs /rootfs live.sqsh

RUN --mount=type=bind,from=os,target=/rootfs \
	cd /rootfs && tar -zcf /os.tgz .

FROM scratch

COPY --from=squash /live.sqsh /
COPY --from=squash /os.tgz /
