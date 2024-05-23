FROM ubuntu:22.04

RUN apt-get update \
  && apt-get install -y jq less curl

COPY fake_cloud_init.sh /init.sh

EXPOSE 8521
EXPOSE 25565

ENTRYPOINT ["/init.sh"]
