# syntax=docker/dockerfile:experimental

FROM alpine:latest as alpine

FROM ubuntu:18.10 as squash

RUN apt update && apt install -y squashfs-tools


RUN --mount=type=bind,from=alpine,target=/tmp mksquashfs /tmp live.sqsh

FROM scratch

COPY --from=squash /live.sqsh /
