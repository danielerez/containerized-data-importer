FROM registry.fedoraproject.org/fedora-minimal:29
MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

COPY fedora.repo /tmp/fedora_ci.dnf.repo

RUN sed -i 's/proxy = None//gI' /tmp/fedora_ci.dnf.repo && \
    mkdir /etc/yum.repos.d/old && \
	mv /etc/yum.repos.d/*.repo /etc/yum.repos.d/old  && \
	mv /tmp/fedora_ci.dnf.repo /etc/yum.repos.d/fedora.repo && \
	microdnf update -y && microdnf install tar -y && microdnf clean all && \
	mv /etc/yum.repos.d/old/* /etc/yum.repos.d/ && \
	rmdir /etc/yum.repos.d/old

ADD cloner_startup.sh /usr/bin/cloner_startup.sh
RUN chmod +x /usr/bin/cloner_startup.sh

COPY ./cdi-cloner /usr/bin/cdi-cloner

ENTRYPOINT [ "/usr/bin/cloner_startup.sh" ]
