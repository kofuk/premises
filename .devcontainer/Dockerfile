FROM mcr.microsoft.com/devcontainers/base:ubuntu-22.04

RUN apt-get update && \
    apt-get install -y bash-completion lighttpd postgresql-client redis-tools tmux uuid-runtime

# Install Node.js
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y nodejs

# Install Go
RUN curl -fsSL https://go.dev/dl/go1.22.3.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# Install Deno
RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local sh

COPY welcome-message.txt /usr/local/etc/vscode-dev-containers/first-run-notice.txt
