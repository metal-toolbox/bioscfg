FROM ubuntu:jammy AS stage1

# Dell racadm
WORKDIR /tmp/racadm

## Install Dependencies
RUN apt update && apt install -y libssl-dev wget pciutils libargtable2-0

## Download
RUN wget -U="Mozilla/5.0 (X11; Linux x86_64; rv:10.0) Gecko/20100101 Firefox/10.0" https://dl.dell.com/FOLDER12638439M/1/Dell-iDRACTools-Web-LX-11.3.0.0-795_A00.tar.gz
RUN tar -xzvf Dell-iDRACTools-Web-LX-11.3.0.0-795_A00.tar.gz

## Workaround to ignore systemctl
RUN echo -e '#!/bin/bash\nexit 0' > /bin/systemctl && chmod +x /bin/systemctl

## Install racadm
WORKDIR /tmp/racadm/iDRACTools/racadm
RUN ./install_racadm.sh

## Install ipmitool
WORKDIR /tmp/racadm/iDRACTools/ipmitool/UBUNTU22_x86_64
RUN dpkg -i ./ipmitool_1.8.18_amd64.deb

WORKDIR /tmp
RUN rm -rf /tmp/racadm

## Get IPMI IANA resource, to prevent dependency on third party servers at runtime.
WORKDIR /usr/share/misc
RUN wget https://www.iana.org/assignments/enterprise-numbers.txt

# Supermicro SUM
WORKDIR /tmp/sum

## Download
RUN wget https://www.supermicro.com/Bios/sw_download/698/sum_2.14.0_Linux_x86_64_20240215.tar.gz -O sum.tar.gz
RUN mkdir -p unzipped
RUN tar -xvzf sum.tar.gz -C unzipped --strip-components=1

## Install
RUN cp unzipped/sum /usr/bin/sum #TODO; smc sum has the same name as the gnu command sum (/usr/bin/sum). So we are overwritting it. Sorry not Sorry.
RUN chmod +x /usr/bin/sum

WORKDIR /tmp
RUN rm -rf /tmp/sum

# Build a lean image with dependencies installed.
FROM ubuntu:jammy
COPY --from=stage1 / /

COPY bioscfg /usr/sbin/bioscfg
RUN chmod +x /usr/sbin/bioscfg

ENTRYPOINT ["/usr/sbin/bioscfg"]
