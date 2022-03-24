FROM golang:latest
WORKDIR /build
COPY . .
RUN make

FROM node:latest
WORKDIR /build
COPY . .
RUN npm ci && npm run prod

FROM alpine:latest
COPY --from=0 /build/premises /premises
COPY --from=1 /build/gen /gen
RUN apk --no-cache add openssl
ENTRYPOINT ["/premises"]
