FROM --platform=$BUILDPLATFORM ghcr.io/jzero-io/jzero:latest AS builder

ARG TARGETARCH
ARG LDFLAGS


RUN apk add --no-cache git

WORKDIR /app

COPY ./ ./

RUN jzero serverless build \
    && jzero gen swagger \
    && test -f desc/swagger/swagger.json

RUN --mount=type=cache,target=/go/pkg CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="$LDFLAGS" -o /dist/app main.go \
    && jzero serverless delete && for plugin in plugins/*/; do \
         [ -d "$plugin" ] || continue; \
         plugin_name=$(basename "$plugin" | tr '-' '_'); \
         (cd "$plugin" && CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -trimpath -ldflags="-s -w -buildid= $LDFLAGS" -o "/dist/$plugin_name" ./cmd/skillcli); \
       done \
    && cp -r etc /dist \
    && mkdir -p /dist/desc && cp -r desc/swagger /dist/desc && cp -r desc/sql_migration /dist/desc \
    && for plugin in plugins/*/; do \
         [ -d "$plugin" ] || continue; \
         plugin_name=$(basename "$plugin" | tr '-' '_'); \
         mkdir -p "/dist/$plugin" && cp -r "$plugin/etc" "/dist/$plugin"; \
         [ -d "$plugin/skills" ] && cp -r "$plugin/skills" "/dist/$plugin" || true; \
         for skill in "$plugin"/skills/*/; do \
           [ -d "$skill" ] || continue; \
           skill_name=$(basename "$skill"); \
           mkdir -p "/dist/$plugin/skills/$skill_name/scripts"; \
           cp "/dist/$plugin_name" "/dist/$plugin/skills/$skill_name/scripts/$plugin_name"; \
         done; \
         [ -d "$plugin/desc/sql_migration" ] && mkdir -p "/dist/$plugin/desc" && cp -r "$plugin/desc/sql_migration" "/dist/$plugin/desc" || true; \
         rm -f "/dist/$plugin_name"; \
       done

FROM alpine:latest

RUN apk add --no-cache ca-certificates curl jq py3-pycryptodome python3 tar tzdata \
    && python3 -c "import Crypto"
WORKDIR /dist

ENV PATH="/dist:${PATH}"
ENV AGENTRAZOR_PLUGIN_SKILLS_ROOT="/dist/plugins"

COPY --from=builder /dist .

EXPOSE 8001

CMD ["./app", "server"]
