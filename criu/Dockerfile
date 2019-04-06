FROM ubuntu:18.04 as build

RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    sudo \
    gawk \
	git \
    iptables \
    jq \
    pkg-config \
    libaio-dev \
    libcap-dev \
    libprotobuf-dev \
    libprotobuf-c0-dev \
    libnl-3-dev \
    libnet-dev \
    libseccomp2 \
    libseccomp-dev \
    protobuf-c-compiler \
    protobuf-compiler \
    python-minimal \
    uidmap \
    kmod \
    && apt-get clean

RUN git clone https://github.com/checkpoint-restore/criu /criu
RUN cd /criu \
    && make install-criu

FROM scratch

COPY --from=build /usr/local/sbin/criu /bin/
