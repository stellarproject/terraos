FROM ubuntu:18.10 as debootstrap

RUN apt update && \
	apt install -y \
	debootstrap

RUN debootstrap --arch amd64 cosmic /terra/ http://archive.ubuntu.com/ubuntu

FROM scratch

COPY --from=debootstrap /terra/ /

# add full apt sources
ADD sources.list /etc/apt/
ADD 01-netcfg.yaml  /etc/netplan

RUN rm /etc/fstab

# add the default terra user
RUN addgroup --system terra && \
	useradd -s /bin/bash -g terra --groups adm -m -k /dev/null terra && \
	echo 'terra:terra' | chpasswd

ADD sudoers /etc/
ADD hosts /etc/
ADD hostname /etc/
ADD resolv.conf /etc/

RUN chmod 0440 /etc/sudoers

CMD ["bash"]
