# syntax=docker/dockerfile:experimental
FROM stellarproject/ubuntu:18.10 as rootfs
FROM ubuntu:18.10 as squash

RUN apt update && apt install -y squashfs-tools
RUN --mount=type=bind,from=rootfs,target=/rootfs mksquashfs /rootfs live.sqsh

FROM scratch
COPY --from=squash /live.sqsh /
