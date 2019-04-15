FROM docker.io/stellarproject/terraos:v6

RUN echo "testing" > /etc/hostname
ADD hosts /etc/hosts
ADD 01-netcfg.yaml  /etc/netplan/

RUN chown -R terra:terra /home/terra
USER terra

RUN mkdir -p /home/terra/.ssh && \
	curl -Ssl https://github.com/crosbymichael.keys -o /home/terra/.ssh/authorized_keys
