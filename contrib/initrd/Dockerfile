FROM alpine:latest as netboot

RUN apk add -U curl squashfs-tools
RUN cd tmp && curl -sSL http://dl-cdn.alpinelinux.org/alpine/v3.9/releases/x86_64/alpine-netboot-3.9.3-x86_64.tar.gz -o netboot.tar.gz && \
	tar zxf netboot.tar.gz && \
	unsquashfs boot/modloop-vanilla && \
	cp -r squashfs-root/modules /modules

FROM alpine:latest as initrd
ENV ALPINE_VERSION v3.4
RUN apk add -U xz
RUN apk --arch x86_64 -X http://dl-cdn.alpinelinux.org/alpine/${ALPINE_VERSION}/main/ -U --allow-untrusted --root /initrd-root --initdb add alpine-base openssh ethtool syslinux squashfs-tools && \
	cp /etc/apk/repositories /initrd-root/etc/apk/
RUN ln -vs /etc/init.d/hostname /initrd-root/etc/runlevels/boot/hostname && \
	ln -vs /etc/init.d/procfs /initrd-root/etc/runlevels/boot/procfs && \
	ln -vs /etc/init.d/sysfs /initrd-root/etc/runlevels/boot/sysfs && \
	ln -vs /etc/init.d/urandom /initrd-root/etc/runlevels/boot/urandom && \
	ln -vs /etc/init.d/hwdrivers /initrd-root/etc/runlevels/boot/hwdrivers && \
	ln -vs /etc/init.d/sshd /initrd-root/etc/runlevels/default/sshd && \
	ln -vs /etc/init.d/local /initrd-root/etc/runlevels/default/local

RUN sed -i 's/#\(ttyS0.*\)/\1/' /initrd-root/etc/inittab && \
	echo ttyS0 >> /initrd-root/etc/securetty
ADD init /initrd-root/init
ADD dhcp.start /initrd-root/etc/local.d/dhcp.start
RUN echo terra > /initrd-root/etc/hostname
COPY --from=netboot /modules /initrd-root/lib/modules
RUN cd /initrd-root && find . | cpio -o -H newc | xz -C crc32 -z -9 --threads=0 -c - > ../initrd.xz

FROM scratch
COPY --from=initrd /initrd.xz /initrd.xz
