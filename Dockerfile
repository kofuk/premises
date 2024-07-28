FROM golang:1.22 AS go_build
WORKDIR /build
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download -x
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=./internal,target=./internal \
    --mount=type=bind,source=./controlpanel,target=./controlpanel \
    cd /build/controlpanel/pmctl && \
    CGO_ENABLED=0 go build -o /pmctl . && \
    cd /build/controlpanel && \
    CGO_ENABLED=0 go build -o /premises .

FROM node:22 AS frontend_build
WORKDIR /build
RUN --mount=type=cache,target=/root/.npm,sharing=locked \
    --mount=type=bind,source=controlpanel/package.json,target=package.json \
    --mount=type=bind,source=controlpanel/package-lock.json,target=package-lock.json \
    npm ci
RUN --mount=type=bind,source=controlpanel/package.json,target=package.json \
    --mount=type=bind,source=controlpanel/package-lock.json,target=package-lock.json \
    --mount=type=bind,source=controlpanel/public,target=public \
    --mount=type=bind,source=controlpanel/frontend,target=frontend \
    --mount=type=bind,source=controlpanel/index.html,target=index.html \
    --mount=type=bind,source=controlpanel/tsconfig.json,target=tsconfig.json \
    --mount=type=bind,source=controlpanel/vite.config.ts,target=vite.config.ts \
    npm run build

FROM scratch
COPY --from=go_build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=go_build /pmctl /bin/pmctl
COPY --from=go_build /premises /premises
COPY --from=frontend_build /build/gen /gen
CMD ["/premises"]
