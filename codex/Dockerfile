FROM --platform=$BUILDPLATFORM ghcr.io/jzero-io/jzero:latest AS builder

ARG TARGETARCH
ARG LDFLAGS

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY server/plugins/ ./plugins/

RUN --mount=type=cache,target=/go/pkg mkdir -p /dist/skills \
    && for plugin in plugins/*/; do \
         [ -d "$plugin" ] || continue; \
         plugin_name=$(basename "$plugin"); \
         binary_name=$(printf '%s' "$plugin_name" | tr '-' '_'); \
         (cd "$plugin" && CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="$LDFLAGS" -o "/dist/$binary_name" .); \
         if [ -d "$plugin/etc" ]; then \
           mkdir -p "/dist/plugins/$plugin_name"; \
           cp -r "$plugin/etc" "/dist/plugins/$plugin_name"; \
         fi; \
         [ -d "$plugin/skills" ] && cp -r "$plugin/skills/." /dist/skills/ || true; \
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

WORKDIR /dist

ENV PATH="/dist:/opt/codex/bin:/opt/codex/codex-path:$PATH" \
    CODEX_HOME=/dist/data \
    CODEX_SOCKET=/run/codex/app-server.sock

COPY --from=builder /dist/ /dist/
COPY codex/config/ /dist/defaults/
COPY --chmod=0755 codex/entrypoint.sh /usr/local/bin/agentrazor-codex-entrypoint

ENTRYPOINT ["agentrazor-codex-entrypoint"]
