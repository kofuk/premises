FROM ubuntu:24.04@sha256:d1e2e92c075e5ca139d51a140fff46f84315c0fdce203eab2807c7e495eff4f9

RUN apt-get update \
  && apt-get install -y jq less curl

COPY fake_cloud_init.sh /init.sh

RUN mkdir /opt/premises

ENV OTEL_SERVICE_NAME=premises-runner \
    OTEL_EXPORTER_OTLP_ENDPOINT="http://jaeger:4317"

EXPOSE 8521
EXPOSE 25565

ENTRYPOINT ["/init.sh"]
