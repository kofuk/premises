FROM mcr.microsoft.com/devcontainers/base:ubuntu-22.04@sha256:81380e4c9c14e8a629ff39029639e4b7893e67400246fa7782a0fe7dc193a02a

RUN apt-get update -y && \
    install -dm 755 /etc/apt/keyrings && \
    curl -fSs https://mise.jdx.dev/gpg-key.pub | sudo tee /etc/apt/keyrings/mise-archive-keyring.asc 1>/dev/null && \
    echo "deb [signed-by=/etc/apt/keyrings/mise-archive-keyring.asc] https://mise.jdx.dev/deb stable main" >/etc/apt/sources.list.d/mise.list && \
    apt-get update -y

RUN apt-get install -y bash-completion jq mise postgresql-client redis-tools tmux uuid-runtime && \
    mkdir -p /usr/local/share/bash-completion/completions

COPY welcome-message.txt /usr/local/etc/vscode-dev-containers/first-run-notice.txt

CMD ["sleep", "infinity"]
