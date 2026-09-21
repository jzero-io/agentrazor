FROM node:20.12.2-bullseye-slim AS builder

WORKDIR /agentrazor-agent

COPY ./ ./

RUN --mount=type=cache,target=/node_modules npm install -g pnpm@8.15.6 \
  && pnpm i \
  && pnpm build

FROM nginx:alpine

WORKDIR /usr/share/nginx/

# Start nginx directly at runtime. This skips the base image entrypoint hooks,
# whose optional IPv6 migration can block on an Alpine repository request.
ENTRYPOINT ["nginx"]
CMD ["-g", "daemon off;"]

COPY ./nginx/default.conf /etc/nginx/conf.d/default.conf
RUN nginx -t

COPY --from=builder /agentrazor-agent/dist ./html
