FROM golang:latest AS controlpanel_build
WORKDIR /build
COPY ./go.* .
COPY ./common ./common
COPY ./controlpanel ./controlpanel
RUN cd /build/controlpanel && make

FROM node:21 AS frontend_build
WORKDIR /build
COPY ./controlpanel/frontend ./frontend
COPY ./controlpanel/public ./public
COPY ./controlpanel/*.js .
COPY ./controlpanel/*.ts .
COPY ./controlpanel/*.html .
COPY ./controlpanel/*.json .
RUN npm ci
RUN npm run build

FROM rust:alpine AS tool_build
WORKDIR /build
COPY ./Cargo.* .
COPY ./pmctl ./pmctl
COPY ./mcserver-fake ./mcserver-fake
RUN apk --no-cache add musl-dev
RUN cargo build --release

FROM scratch AS prod_base
COPY --from=controlpanel_build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=tool_build /build/target/release/pmctl /bin/pmctl
COPY --from=controlpanel_build /build/controlpanel/premises /premises
COPY --from=frontend_build /build/gen /gen

# Hack to merge all layers without --squash.
FROM scratch
COPY --from=prod_base / /
ENTRYPOINT ["/premises"]
