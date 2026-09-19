FROM node:20.12.2-bullseye-slim AS builder

# Build context is the repository root so the generic locale loader can include
# the plugin bundles that are present in this build.
WORKDIR /workspace

COPY ./admin ./admin
COPY ./server/plugins ./server/plugins

WORKDIR /workspace/admin

RUN --mount=type=cache,target=/node_modules npm install -g pnpm@8.15.6 \
  && pnpm i \
  && pnpm build

FROM nginx:alpine

WORKDIR /usr/share/nginx/

COPY ./admin/nginx/default.conf /etc/nginx/conf.d/default.conf
RUN nginx -t

COPY --from=builder /workspace/admin/dist ./html
