FROM registry.service.aegis/containerd:latest AS boss

FROM ubuntu:18.10

RUN apt-get update && \
        apt-get upgrade -y

RUN apt-get install -y \
	systemd \
	openssh-server \
	iproute2

RUN (cd /lib/systemd/system/sysinit.target.wants/; for i in *; do [ $i == systemd-tmpfiles-setup.service ] || rm -f $i; done); \
        rm -f /lib/systemd/system/multi-user.target.wants/*; \
        rm -f /etc/systemd/system/*.wants/*; \
        rm -f /lib/systemd/system/local-fs.target.wants/*; \
        rm -f /lib/systemd/system/sockets.target.wants/*udev*; \
        rm -f /lib/systemd/system/sockets.target.wants/*initctl*; \
        rm -f /lib/systemd/system/basic.target.wants/*;\
        rm -f /lib/systemd/system/anaconda.target.wants/*;


COPY --from=boss /bin/* /usr/local/bin/
Add containerd.service /lib/systemd/system/

RUN systemctl enable ssh containerd

CMD ["/sbin/init"]
