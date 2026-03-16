FROM ubuntu:24.04@sha256:d1e2e92c075e5ca139d51a140fff46f84315c0fdce203eab2807c7e495eff4f9

RUN apt-get update && apt-get install -y curl jq less

COPY fake_cloud_init.sh /init.sh

RUN mkdir /opt/premises

EXPOSE 8521
EXPOSE 25565

ENTRYPOINT ["/init.sh"]
