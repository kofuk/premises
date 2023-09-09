FROM golang:latest
WORKDIR /build
COPY . .
RUN cd /build/home && make

FROM node:latest
WORKDIR /build
COPY /home .
RUN npm ci && npm run prod

FROM rust:alpine
WORKDIR /build
COPY . .
RUN apk --no-cache add musl-dev
RUN cargo build --release

FROM alpine:latest
COPY --from=0 /build/home/premises /premises
COPY --from=1 /build/gen /gen
COPY --from=2 /build/target/release/cli /bin/pmctl
RUN apk --no-cache add openssl && mkdir -p /opt/premises
ENTRYPOINT ["/premises"]
