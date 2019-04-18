FROM ubuntu:18.10 as ipxe
RUN apt-get update && apt-get install -y --no-install-recommends git \
	build-essential \
	ca-certificates \
	curl \
	make \
	binutils \
	perl \
	liblzma-dev \
	mtools \
	genisoimage \
	syslinux
RUN git clone https://git.ipxe.org/ipxe.git && \
	cd ipxe/src && \
	make bin/undionly.kpxe

FROM alpine:latest as pxe
RUN apk add -U syslinux
RUN mkdir /tftp && \
	cp /usr/share/syslinux/pxelinux.0 /tftp/ && \
	cp -r /usr/share/syslinux/libmenu.c32 \
		/usr/share/syslinux/menu.c32 \
		/usr/share/syslinux/memdisk \
		/usr/share/syslinux/ldlinux.c32 \
		/usr/share/syslinux/libutil.c32 /tftp/ && \
	mkdir /tftp/pxelinux.cfg
ADD default /tftp/pxelinux.cfg/default

FROM scratch
# pxe
COPY --from=pxe /tftp /tftp
