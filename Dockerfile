FROM golang:latest
WORKDIR /build
COPY . .
RUN cd /build/home && make

FROM node:latest
WORKDIR /build
COPY /home .
RUN npm ci && npm run prod

FROM alpine:latest
COPY --from=0 /build/home/premises /premises
COPY --from=1 /build/gen /gen
RUN apk --no-cache add openssl && mkdir -p /opt/premises
ENTRYPOINT ["/premises"]
