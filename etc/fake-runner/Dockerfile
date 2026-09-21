FROM ubuntu:24.04@sha256:008173c23f95b170204355c12626cb5a965d779a7e1283b09e9cffbb1bf33ca3

RUN apt-get update && apt-get install -y curl jq less

COPY fake_cloud_init.sh /init.sh

RUN mkdir /opt/premises

EXPOSE 8521
EXPOSE 25565

ENTRYPOINT ["/init.sh"]
