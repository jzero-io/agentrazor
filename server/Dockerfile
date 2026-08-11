FROM --platform=$BUILDPLATFORM ghcr.io/jzero-io/jzero:latest AS builder

ARG TARGETARCH
ARG LDFLAGS

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY ./ ./

RUN jzero serverless build

RUN --mount=type=cache,target=/go/pkg CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="$LDFLAGS" -o /dist/app main.go \
    && for plugin in plugins/*/; do \
         plugin_name=$(basename "$plugin" | tr '-' '_'); \
         (cd "$plugin" && CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="$LDFLAGS" -o "/dist/$plugin_name" .); \
       done \
    && cp -r etc /dist \
    && mkdir -p /dist/desc && cp -r desc/swagger /dist/desc && cp -r desc/sql_migration /dist/desc \
    && for plugin in $(ls -d plugins/*/); do \
         mkdir -p /dist/$plugin && cp -r $plugin/etc /dist/$plugin; \
         [ -d $plugin/skills ] && cp -r $plugin/skills /dist/$plugin || true; \
         [ -d $plugin/desc/sql_migration ] && mkdir -p /dist/$plugin/desc && cp -r $plugin/desc/sql_migration /dist/$plugin/desc || true; \
       done

FROM alpine:latest

ARG TARGETARCH

RUN apk add --no-cache bubblewrap ca-certificates curl tar tzdata \
    && case "$TARGETARCH" in \
         amd64) codex_arch="x86_64" ;; \
         arm64) codex_arch="aarch64" ;; \
         *) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
       esac \
    && curl -fsSL "https://github.com/openai/codex/releases/latest/download/codex-${codex_arch}-unknown-linux-musl.tar.gz" -o /tmp/codex.tar.gz \
    && tar -xzf /tmp/codex.tar.gz -C /usr/local/bin \
    && mv "/usr/local/bin/codex-${codex_arch}-unknown-linux-musl" /usr/local/bin/codex \
    && chmod +x /usr/local/bin/codex \
    && rm -f /tmp/codex.tar.gz

WORKDIR /dist

ENV PATH="/dist:${PATH}"

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
