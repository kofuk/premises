FROM golang:latest
WORKDIR /build
COPY . .
RUN make

FROM node:latest
WORKDIR /build
COPY . .
RUN npm ci && npm run prod

FROM scratch
COPY --from=0 /build/premises /premises
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=1 /build/gen /gen
ENTRYPOINT ["/premises"]
