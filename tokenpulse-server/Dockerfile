# 构建阶段：先复制依赖清单以复用模块下载缓存，再编译静态 Linux 二进制。
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY tokenpulse-server/go.mod tokenpulse-server/go.sum ./
RUN go mod download
COPY tokenpulse-server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /tokenpulse-server ./cmd/server

# 运行阶段：仅保留证书、时区、迁移脚本和非 root 用户，缩小攻击面。
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H tokenpulse
COPY --from=build /tokenpulse-server /usr/local/bin/tokenpulse-server
COPY --from=build /src/migrations /app/migrations
USER tokenpulse
WORKDIR /app
# 服务默认监听 8080，实际端口可通过 SERVER_PORT 修改。
EXPOSE 8080
ENTRYPOINT ["tokenpulse-server"]
