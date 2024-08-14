FROM golang:1.23 AS go_build
WORKDIR /build
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download -x
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=./internal,target=internal \
    --mount=type=bind,source=./controlpanel/cmd,target=controlpanel/cmd \
    --mount=type=bind,source=./controlpanel/internal,target=controlpanel/internal \
    cd /build/controlpanel/cmd/pmctl && \
    CGO_ENABLED=0 go build -o /pmctl . && \
    cd /build/controlpanel/cmd/premises && \
    CGO_ENABLED=0 go build -o /premises .

FROM node:22 AS frontend_build
WORKDIR /build
RUN --mount=type=cache,target=/root/.npm,sharing=locked \
    --mount=type=bind,source=controlpanel/frontend/package.json,target=package.json \
    --mount=type=bind,source=controlpanel/frontend/package-lock.json,target=package-lock.json \
    npm ci
RUN --mount=type=cache,target=/root/.npm \
    --mount=type=bind,source=controlpanel/frontend/package.json,target=package.json \
    --mount=type=bind,source=controlpanel/frontend/package-lock.json,target=package-lock.json \
    --mount=type=bind,source=controlpanel/frontend/public,target=public \
    --mount=type=bind,source=controlpanel/frontend/src,target=src \
    --mount=type=bind,source=controlpanel/frontend/index.html,target=index.html \
    --mount=type=bind,source=controlpanel/frontend/tsconfig.json,target=tsconfig.json \
    --mount=type=bind,source=controlpanel/frontend/vite.config.ts,target=vite.config.ts \
    npm run build

FROM scratch
COPY --from=go_build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=go_build /pmctl /bin/pmctl
COPY --from=go_build /premises /premises
COPY --from=frontend_build /build/gen /gen
CMD ["/premises"]
