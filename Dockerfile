FROM golang:1.26.1@sha256:595c7847cff97c9a9e76f015083c481d26078f961c9c8dca3923132f51fe12f1 AS go_build
WORKDIR /build
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=bind,source=./backend/common/go.mod,target=backend/common/go.mod \
    --mount=type=bind,source=./backend/common/go.sum,target=backend/common/go.sum \
    --mount=type=bind,source=./backend/runner/go.mod,target=backend/runner/go.mod \
    --mount=type=bind,source=./backend/runner/go.sum,target=backend/runner/go.sum \
    --mount=type=bind,source=./backend/ctrlplane/common/go.mod,target=backend/ctrlplane/common/go.mod \
    --mount=type=bind,source=./backend/ctrlplane/common/go.sum,target=backend/ctrlplane/common/go.sum \
    --mount=type=bind,source=./backend/ctrlplane/monolith/go.mod,target=backend/ctrlplane/monolith/go.mod \
    --mount=type=bind,source=./backend/ctrlplane/monolith/go.sum,target=backend/ctrlplane/monolith/go.sum \
    --mount=type=bind,source=./backend/ctrlplane/pmctl/go.mod,target=backend/ctrlplane/pmctl/go.mod \
    --mount=type=bind,source=./backend/ctrlplane/pmctl/go.sum,target=backend/ctrlplane/pmctl/go.sum \
    --mount=type=bind,source=./backend/tools/mcserver-fake/go.mod,target=backend/tools/mcserver-fake/go.mod \
    --mount=type=bind,source=./backend/tools/mcserver-fake/go.sum,target=backend/tools/mcserver-fake/go.sum \
    --mount=type=bind,source=./backend/tools/ostack-fake/go.mod,target=backend/tools/ostack-fake/go.mod \
    --mount=type=bind,source=./backend/tools/ostack-fake/go.sum,target=backend/tools/ostack-fake/go.sum \
    --mount=type=bind,source=./go.work,target=go.work \
    --mount=type=bind,source=./go.work.sum,target=go.work.sum \
    go mod download -x
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=./go.work,target=go.work \
    --mount=type=bind,source=./go.work.sum,target=go.work.sum \
    --mount=type=bind,source=./backend,target=backend \
    cd /build/backend/ctrlplane/pmctl && \
    CGO_ENABLED=0 go build -o /pmctl . && \
    cd /build/backend/ctrlplane/monolith && \
    CGO_ENABLED=0 go build -o /premises .

FROM node:24.14.0@sha256:5a593d74b632d1c6f816457477b6819760e13624455d587eef0fa418c8d0777b AS frontend_build
WORKDIR /build
RUN corepack enable
RUN --mount=type=cache,target=/root/.local/share/pnpm/store,sharing=locked \
    --mount=type=bind,source=frontend/package.json,target=package.json \
    --mount=type=bind,source=frontend/pnpm-lock.yaml,target=pnpm-lock.yaml \
    pnpm install --frozen-lockfile
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    --mount=type=bind,source=frontend/package.json,target=package.json \
    --mount=type=bind,source=frontend/pnpm-lock.yaml,target=pnpm-lock.yaml \
    --mount=type=bind,source=frontend/public,target=public \
    --mount=type=bind,source=frontend/src,target=src \
    --mount=type=bind,source=frontend/index.html,target=index.html \
    --mount=type=bind,source=frontend/tsconfig.json,target=tsconfig.json \
    --mount=type=bind,source=frontend/vite.config.ts,target=vite.config.ts \
    pnpm run build

FROM scratch
ENV PREMISES_STATIC_DIR=/static
COPY --from=go_build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=go_build /pmctl /bin/pmctl
COPY --from=go_build /premises /premises
COPY --from=frontend_build /build/gen /static
CMD ["/premises"]
