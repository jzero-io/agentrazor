FROM --platform=$BUILDPLATFORM ghcr.io/jzero-io/jzero:latest AS builder

ARG TARGETARCH
ARG LDFLAGS

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY ./ ./

RUN jzero serverless build \
    && jzero gen swagger \
    && test -f desc/swagger/swagger.json

RUN --mount=type=cache,target=/go/pkg CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="$LDFLAGS" -o /dist/app main.go \
    && jzero serverless delete && for plugin in plugins/*/; do \
         [ -d "$plugin" ] || continue; \
         plugin_name=$(basename "$plugin" | tr '-' '_'); \
         (cd "$plugin" && CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="$LDFLAGS" -o "/dist/$plugin_name" .); \
       done \
    && cp -r etc /dist \
    && mkdir -p /dist/desc && cp -r desc/swagger /dist/desc && cp -r desc/sql_migration /dist/desc \
    && for plugin in plugins/*/; do \
         [ -d "$plugin" ] || continue; \
         mkdir -p "/dist/$plugin" && cp -r "$plugin/etc" "/dist/$plugin"; \
         [ -d "$plugin/skills" ] && cp -r "$plugin/skills" "/dist/$plugin" || true; \
         [ -d "$plugin/desc/sql_migration" ] && mkdir -p "/dist/$plugin/desc" && cp -r "$plugin/desc/sql_migration" "/dist/$plugin/desc" || true; \
       done

FROM alpine:latest

ARG TARGETARCH
ARG CODEX_VERSION=0.153.4

RUN set -o pipefail \
    && apk add --no-cache ca-certificates curl jq py3-pycryptodome python3 tar tzdata \
    && python3 -c "import Crypto" \
    && case "$TARGETARCH" in \
         amd64) codex_arch="x86_64" ;; \
         arm64) codex_arch="aarch64" ;; \
         *) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
       esac \
    && mkdir -p /opt/codex \
    && curl -fsSL "https://github.com/openai/codex/releases/download/rust-v${CODEX_VERSION}/codex-app-server-package-${codex_arch}-unknown-linux-musl.tar.gz" \
       | tar -xzf - -C /opt/codex \
    && test -x /opt/codex/bin/codex-app-server \
    && test -x /opt/codex/bin/codex-code-mode-host \
    && test -x /opt/codex/codex-resources/bwrap \
    && test -x /opt/codex/codex-path/rg

WORKDIR /dist

ENV PATH="/dist:/opt/codex/bin:/opt/codex/codex-path:${PATH}"

COPY --from=builder /dist .

# Login shells may reset PATH, so expose every built plugin CLI from a
# standard system path as well as /dist.
RUN for plugin in /dist/plugins/*/; do \
      [ -d "$plugin" ] || continue; \
      plugin_name=$(basename "$plugin" | tr '-' '_'); \
      [ -x "/dist/$plugin_name" ] || continue; \
      ln -sf "/dist/$plugin_name" "/usr/local/bin/$plugin_name"; \
    done

EXPOSE 8001

CMD ["./app", "server"]
