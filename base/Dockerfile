# Copyright (c) 2019 Stellar Project

# Permission is hereby granted, free of charge, to any person
# obtaining a copy of this software and associated documentation
# files (the "Software"), to deal in the Software without
# restriction, including without limitation the rights to use, copy,
# modify, merge, publish, distribute, sublicense, and/or sell copies
# of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED,
# INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
# IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
# HOLDERS BE LIABLE FOR ANY CLAIM,
# DAMAGES OR OTHER LIABILITY,
# WHETHER IN AN ACTION OF CONTRACT,
# TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH
# THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

FROM ubuntu:18.10 as debootstrap

RUN apt update && \
	apt install -y \
	debootstrap

RUN debootstrap --arch amd64 cosmic /terra/ http://archive.ubuntu.com/ubuntu

# base image
FROM scratch

COPY --from=debootstrap /terra/ /

# add full apt sources
ADD sources.list /etc/apt/
RUN apt update && \
	apt install -y \
	openssh-server \
	apparmor

ADD 01-netcfg.yaml  /etc/netplan

RUN rm /etc/fstab

# add the default terra user
RUN addgroup --system terra && \
	useradd -s /bin/bash -g terra --groups adm --groups cdrom --groups sudo --groups dip --groups plugdev -m -k /dev/null terra && \
	chown -R terra:terra /home/terra

ADD sudoers /etc/
ADD hosts /etc/
ADD hostname /etc/
ADD resolv.conf /etc/

RUN chmod 0440 /etc/sudoers

ADD sshd_config /etc/ssh/
RUN systemctl disable systemd-resolved && \
	systemctl enable ssh

CMD ["bash"]
