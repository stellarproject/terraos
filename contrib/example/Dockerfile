# syntax=docker/dockerfile:experimental



FROM registry.compute:5000/terraos:v14



ADD etc/hostname /etc/
ADD etc/hosts /etc/
ADD etc/resolv.conf /etc/
ADD etc/hostname /etc/
ADD etc/netplan/01-netcfg.yaml /etc/netplan/

ADD home/terra/.ssh /home/terra/.ssh

RUN chown -R terra:terra /home/terra

RUN dbus-uuidgen --ensure=/etc/machine-id && dbus-uuidgen --ensure

RUN echo 'terra:terra' | chpasswd
ADD sshd_config /etc/ssh/


CMD ["/sbin/init"]
