# syntax=docker/dockerfile:experimental



FROM docker.io/stellarproject/terraos:v13



ADD . /

RUN dbus-uuidgen --ensure=/etc/machine-id && dbus-uuidgen --ensure

RUN echo 'terra:terra' | chpasswd
ADD sshd_config /etc/ssh/


CMD ["/sbin/init"]
