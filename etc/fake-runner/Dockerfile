FROM ubuntu:24.04@sha256:224a1869083a311ef3f13648a154ba79832fbef6364d31493642ca03082da254

RUN apt-get update && apt-get install -y curl jq less

COPY fake_cloud_init.sh /init.sh

RUN mkdir /opt/premises

EXPOSE 8521
EXPOSE 25565

ENTRYPOINT ["/init.sh"]
