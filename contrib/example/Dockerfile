# syntax=docker/dockerfile:experimental



FROM registry.compute:5000/terraos:v16-dev



ADD etc/hostname /etc/
ADD etc/hosts /etc/
ADD etc/fstab /etc/
ADD etc/resolv.conf /etc/
ADD etc/hostname /etc/
ADD etc/netplan/01-netcfg.yaml /etc/netplan/
ADD etc/network/interfaces /etc/network/

ADD home/terra/.ssh /home/terra/.ssh

RUN chown -R terra:terra /home/terra

RUN echo 'terra:terra' | chpasswd
RUN echo 'root:root' | chpasswd
ADD sshd_config /etc/ssh/


CMD ["/sbin/init"]
