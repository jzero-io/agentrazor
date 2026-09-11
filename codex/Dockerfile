FROM alpine:latest

ARG TARGETARCH
ARG CODEX_VERSION=0.153.4

RUN apk add --no-cache ca-certificates curl tar tzdata \
    && case "$TARGETARCH" in \
         amd64) codex_arch=x86_64 ;; \
         arm64) codex_arch=aarch64 ;; \
         *) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
       esac \
    && mkdir -p /opt/codex \
    && curl -fsSL -o /tmp/codex.tar.gz \
       "https://github.com/openai/codex/releases/download/rust-v${CODEX_VERSION}/codex-app-server-package-${codex_arch}-unknown-linux-musl.tar.gz" \
    && tar -xzf /tmp/codex.tar.gz -C /opt/codex \
    && rm /tmp/codex.tar.gz

RUN rm /var/run && mkdir -p /var/run

WORKDIR /dist

ENV PATH="/opt/codex/bin:/opt/codex/codex-path:$PATH" \
    CODEX_HOME=/dist/data \
    CODEX_SOCKET=/var/run/codex-app-server.sock

COPY codex/skills/ /etc/codex/skills/
COPY codex/config/ /dist/defaults/
COPY --chmod=0755 codex/entrypoint.sh /usr/local/bin/agentrazor-codex-entrypoint

ENTRYPOINT ["agentrazor-codex-entrypoint"]
