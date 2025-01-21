FROM ubuntu:22.04

RUN apt-get update \
  && apt-get install -y jq less curl

COPY fake_cloud_init.sh /init.sh

RUN mkdir /opt/premises

ENV OTEL_SERVICE_NAME=premises \
    OTEL_EXPORTER_OTLP_ENDPOINT="http://jaeger:4317"

EXPOSE 8521
EXPOSE 25565

ENTRYPOINT ["/init.sh"]
