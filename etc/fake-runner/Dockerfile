FROM ubuntu:24.04@sha256:186072bba1b2f436cbb91ef2567abca677337cfc786c86e107d25b7072feef0c

RUN apt-get update && apt-get install -y curl jq less

COPY fake_cloud_init.sh /init.sh

RUN mkdir /opt/premises

EXPOSE 8521
EXPOSE 25565

ENTRYPOINT ["/init.sh"]
