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

RUN apk add --no-cache ca-certificates curl jq py3-pycryptodome python3 tar tzdata \
    && python3 -c "import Crypto"
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
