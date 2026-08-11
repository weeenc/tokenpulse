FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
COPY tokenpulse-agent/package.json tokenpulse-agent/package.json
COPY tokenpulse-web/package.json tokenpulse-web/package.json
RUN npm ci --workspace tokenpulse-web --include-workspace-root
COPY tokenpulse-web/ tokenpulse-web/
RUN npm run build --workspace tokenpulse-web

FROM nginxinc/nginx-unprivileged:1.30-alpine
COPY --from=build /app/tokenpulse-web/dist /usr/share/nginx/html
COPY tokenpulse-web/nginx-site.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/ >/dev/null || exit 1
