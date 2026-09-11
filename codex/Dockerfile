FROM --platform=$BUILDPLATFORM ghcr.io/jzero-io/jzero:latest AS builder

ARG TARGETARCH
ARG LDFLAGS

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY server/plugins/ ./plugins/

RUN --mount=type=cache,target=/go/pkg mkdir -p /opt/agentrazor/bin /opt/agentrazor/skills \
    && for plugin in plugins/*/; do \
         [ -d "$plugin" ] || continue; \
         plugin_name=$(basename "$plugin"); \
         binary_name=$(printf '%s' "$plugin_name" | tr '-' '_'); \
         (cd "$plugin" && CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="$LDFLAGS" -o "/opt/agentrazor/bin/$binary_name" .); \
         [ -d "$plugin/skills" ] && cp -r "$plugin/skills/." /opt/agentrazor/skills/ || true; \
       done

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

ENV PATH="/opt/agentrazor/bin:/opt/codex/bin:/opt/codex/codex-path:$PATH" \
    CODEX_HOME=/dist/data \
    CODEX_SOCKET=/var/run/codex-app-server.sock

COPY --from=builder /opt/agentrazor/ /opt/agentrazor/
COPY codex/skills/ /opt/agentrazor/skills/
COPY codex/config/ /dist/defaults/
COPY --chmod=0755 codex/entrypoint.sh /usr/local/bin/agentrazor-codex-entrypoint

RUN for binary in /opt/agentrazor/bin/*; do \
      [ -f "$binary" ] && [ -x "$binary" ] || continue; \
      ln -sf "$binary" "/usr/local/bin/$(basename "$binary")"; \
    done

ENTRYPOINT ["agentrazor-codex-entrypoint"]
