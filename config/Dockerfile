FROM ubuntu:18.10 as setup

RUN apt update && \
	apt install -y \
	curl \
	ca-certificates

RUN echo "aegis-testing" > /etc/hostname
RUN curl -Ssl https://github.com/crosbymichael.keys -o /keys

FROM scratch

COPY --from=setup /etc/hostname /etc/hostname
COPY --from=setup /keys /home/terra/.ssh/authorized_keys
ADD hosts /etc/hosts
