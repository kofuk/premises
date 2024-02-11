FROM golang:latest AS controlpanel_build
WORKDIR /build
COPY . .
RUN cd /build/controlpanel && make

FROM node:21 AS frontend_build
WORKDIR /build
COPY /controlpanel .
RUN npm ci
RUN npm run build

FROM rust:alpine AS tool_build
WORKDIR /build
COPY . .
RUN apk --no-cache add musl-dev
RUN cargo build --release

FROM scratch AS prod_base
COPY --from=controlpanel_build /build/controlpanel/premises /premises
COPY --from=controlpanel_build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=frontend_build /build/gen /gen
COPY --from=tool_build /build/target/release/pmctl /bin/pmctl

# Hack to merge all layers without --squash.
FROM scratch
COPY --from=prod_base / /
ENTRYPOINT ["/premises"]
