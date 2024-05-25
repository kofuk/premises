FROM golang:1.22 AS go_build
WORKDIR /build
COPY ./pmctl ./pmctl
COPY ./internal ./internal
COPY ./go.* .
RUN cd /build/pmctl && CGO_ENABLED=0 go build .
COPY ./controlpanel ./controlpanel
RUN cd /build/controlpanel && make

FROM node:21 AS frontend_build
WORKDIR /build
COPY ./controlpanel/public ./public
COPY ./controlpanel/*.js .
COPY ./controlpanel/*.ts .
COPY ./controlpanel/*.html .
COPY ./controlpanel/frontend ./frontend
COPY ./controlpanel/*.json .
RUN npm ci
RUN npm run build

FROM scratch AS prod_base
COPY --from=go_build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=go_build /build/pmctl/pmctl /bin/pmctl
COPY --from=go_build /build/controlpanel/premises /premises
COPY --from=frontend_build /build/gen /gen

# Hack to merge all layers without --squash.
FROM scratch
COPY --from=prod_base / /
CMD ["/premises"]
