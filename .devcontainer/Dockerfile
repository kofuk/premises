FROM mcr.microsoft.com/devcontainers/base:ubuntu-22.04
ARG GO_VERSION=1.26.0 \
    KIND_VERSION=0.31.0 \
    KUBECTL_VERSION=1.35.1 \
    CLOUD_PROVIDER_KIND_VERSION=0.10.0

RUN apt-get update && \
    apt-get install -y bash-completion lighttpd postgresql-client redis-tools tmux uuid-runtime && \
    mkdir -p /usr/local/share/bash-completion/completions

# Install Node.js
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y nodejs && \
    corepack enable

# Install Go
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# Install Deno
RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local sh

# Install Helm
RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && \
    ./get_helm.sh && \
    rm get_helm.sh

# Install kind
RUN curl -fsSLo /usr/local/bin/kind "https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64" && \
    chmod +x /usr/local/bin/kind && \
    kind completion bash >/usr/local/share/bash-completion/completions/kind

# Install kubectl
RUN  curl -fsSLo /usr/local/bin/kubectl "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl" && \
    chmod +x /usr/local/bin/kubectl && \
    kubectl completion bash >/usr/local/share/bash-completion/completions/kubectl

COPY welcome-message.txt /usr/local/etc/vscode-dev-containers/first-run-notice.txt

CMD ["sleep", "infinity"]
