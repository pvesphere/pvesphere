ARG REGISTRY=docker.io

# ============================================
# 阶段1: Go 后端构建
# ============================================
FROM ${REGISTRY}/golang:1.23-alpine AS go-builder
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

# 安装必要的构建工具
RUN apk add --no-cache git make

# 复制项目文件
COPY . /data/app
WORKDIR /data/app

# 清理旧的构建产物
RUN rm -rf /data/app/bin/

# 下载依赖并构建所有服务
RUN export GOPROXY=https://goproxy.cn,direct && \
    go mod download && \
    go mod tidy && \
    go build -ldflags="-s -w" -o ./bin/server ./cmd/server && \
    go build -ldflags="-s -w" -o ./bin/controller ./cmd/controller && \
    go build -ldflags="-s -w" -o ./bin/migration ./cmd/migration

# 复制配置文件
RUN cp -r config /data/app/bin/ || true

# ============================================
# 阶段2: 前端构建（仅支持本地代码）
# ============================================
FROM ${REGISTRY}/node:20-alpine AS front-builder
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

# 安装 pnpm（不需要 git）
RUN npm install -g pnpm

WORKDIR /app

# 从本地目录复制前端代码（构建脚本会准备代码到 deploy/build/.frontend-local）
# 构建上下文是项目根目录
# 注意：必须先运行 ./deploy/build.sh 准备前端代码，否则 COPY 会失败
COPY deploy/build/.frontend-local /tmp/frontend-local/

# 检查并复制前端代码（仅保留必要的校验）
RUN if [ ! -d "/tmp/frontend-local" ] || [ ! -f "/tmp/frontend-local/package.json" ]; then \
        echo "ERROR: 前端代码目录为空或缺少 package.json"; \
        echo "请先在项目根目录执行: ./deploy/build.sh -f /path/to/pvesphere-ui"; \
        exit 1; \
    fi && \
    cp -r /tmp/frontend-local/* . && \
    rm -rf /tmp/frontend-local

# 创建/覆盖 .env.production 文件，配置 API 地址为空（使用相对路径，通过 Nginx 代理）
# 必须在复制代码后立即创建，确保覆盖前端代码中的配置
# 注意：由于前端代码使用 || 操作符，空字符串会被当作 falsy，所以使用特殊值 "/api/v1"
# 这样前端会使用相对路径，通过 Nginx 代理
RUN echo '# 生产环境配置（Docker 部署）' > .env.production && \
    echo '# API 地址使用相对路径，通过 Nginx 代理到后端' >> .env.production && \
    echo 'VITE_API_BASE_URL=' >> .env.production && \
    echo 'VITE_BASE_API=/api/v1' >> .env.production && \
    echo 'VITE_PORT=4173' >> .env.production && \
    echo 'VITE_CDN=false' >> .env.production && \
    echo 'VITE_COMPRESSION=gzip' >> .env.production && \
    echo 'VITE_PUBLIC_PATH=/' >> .env.production && \
    echo 'VITE_ROUTER_HISTORY=hash' >> .env.production && \
    echo 'VITE_HIDE_HOME=false' >> .env.production

# 配置 pnpm 使用国内镜像并安装依赖
RUN if [ ! -f "package.json" ]; then \
        echo "ERROR: 找不到 package.json"; \
        exit 1; \
    fi && \
    if [ ! -f ".npmrc" ]; then \
        echo "shamefully-hoist=true" > .npmrc && \
        echo "strict-peer-dependencies=false" >> .npmrc; \
    fi && \
    pnpm config set registry https://registry.npmmirror.com && \
    pnpm config set shamefully-hoist true && \
    pnpm config set strict-peer-dependencies false && \
    rm -rf node_modules .pnpm-store 2>/dev/null || true && \
    pnpm install --frozen-lockfile || pnpm install && \
    pnpm run build

# ============================================
# 阶段3: Go 后端运行时镜像
# ============================================
FROM ${REGISTRY}/alpine:3.19 AS backend
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

# 设置时区
RUN apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

ARG APP_ENV=prod
ENV APP_ENV=${APP_ENV}

WORKDIR /data/app

# 从构建阶段复制所有二进制文件和配置
# 注意：所有服务（server、controller、migration）都包含在这个镜像中
COPY --from=go-builder /data/app/bin /data/app

# 默认暴露 8000 端口（API 服务）
EXPOSE 8000

# 使用构建参数指定要运行的服务
ARG APP_NAME=server
ENV APP_NAME=${APP_NAME}

# 检查所有必需的二进制文件是否存在
RUN for binary in server controller migration; do \
        if [ ! -f "/data/app/$binary" ]; then \
            echo "ERROR: Binary /data/app/$binary not found!"; \
            exit 1; \
        fi; \
    done && \
    chmod +x /data/app/*

# 默认命令（可在 docker-compose 中覆盖）
CMD ["sh", "-c", "./${APP_NAME}"]

# ============================================
# 阶段4: 前端运行时镜像（Nginx）
# ============================================
FROM ${REGISTRY}/nginx:alpine AS frontend
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

# 设置时区
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

# 复制前端构建产物（通常在 dist 目录）
COPY --from=front-builder /app/dist /usr/share/nginx/html

# 复制 nginx 配置（构建上下文是项目根目录）
COPY deploy/nginx/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]

# ============================================
# 阶段5: All-in-One 运行时镜像（后端 + 前端）
# 用于快速演示/测试
# ============================================
FROM ${REGISTRY}/alpine:3.19 AS allinone
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

# 基础工具与时间、证书
RUN apk add --no-cache tzdata ca-certificates nginx tini wget curl && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

RUN mkdir -p /etc/nginx/conf.d /var/log/nginx /data/app/storage && \
cat > /etc/nginx/nginx.conf <<'EOF'
user  nginx;
worker_processes auto;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    sendfile        on;
    keepalive_timeout  65;

    access_log  /var/log/nginx/access.log;

    include /etc/nginx/conf.d/*.conf;
}
EOF

ENV APP_ENV=docker
ENV CONFIG_PATH=/data/app/config/docker.yml

WORKDIR /data/app

# 后端二进制与配置
COPY --from=go-builder /data/app/bin /data/app

# 前端静态资源
COPY --from=front-builder /app/dist /usr/share/nginx/html
# 复制 nginx 配置文件
COPY deploy/nginx/nginx.conf /etc/nginx/conf.d/default.conf

# 简单入口脚本，同时拉起后端与 nginx
RUN sed -i 's#http://api:8000#http://127.0.0.1:8000#' /etc/nginx/conf.d/default.conf && \
cat > /entrypoint.sh <<'EOF'
#!/bin/sh
set -e

APP_ENV="${APP_ENV:-docker}"
CONFIG_PATH="${CONFIG_PATH:-/data/app/config/docker.yml}"

echo "=========================================="
echo "PVESphere All-in-One 启动中..."
echo "APP_ENV=${APP_ENV}"
echo "CONFIG_PATH=${CONFIG_PATH}"
echo "=========================================="

# 确保存储目录存在
mkdir -p /data/app/storage

# 1. 执行数据库迁移
echo ""
echo "=== 执行数据库迁移 ==="
/data/app/migration -conf "${CONFIG_PATH}" || {
    echo "ERROR: 数据库迁移失败"
    exit 1
}
echo "✓ 数据库迁移完成"

# 2. 启动 API 服务
echo ""
echo "=== 启动 API 服务 ==="
/data/app/server -conf "${CONFIG_PATH}" &
SERVER_PID=$!
echo "✓ API 服务已启动 (PID: ${SERVER_PID})"

# 等待 API 服务就绪（健康检查）
echo "等待 API 服务就绪..."
for i in $(seq 1 30); do
    # 用 curl 做 GET 检查，-f 只接受 2xx/3xx，-s 静默，-S 出错时打印
    if curl -fsS http://localhost:8000/ >/dev/null; then
        echo "✓ API 服务已就绪"
        break
    fi

    if [ "$i" -eq 30 ]; then
        echo "ERROR: API 服务启动超时"
        kill "${SERVER_PID}" 2>/dev/null || true
        exit 1
    fi

    sleep 1
done

# 3. 启动 Controller 服务
echo ""
echo "=== 启动 Controller 服务 ==="
/data/app/controller -conf "${CONFIG_PATH}" &
CONTROLLER_PID=$!
echo "✓ Controller 服务已启动 (PID: ${CONTROLLER_PID})"
sleep 2

# 4. 启动 Nginx
echo ""
echo "=== 启动 Nginx ==="
nginx -g 'daemon off;' &
NGINX_PID=$!
echo "✓ Nginx 已启动 (PID: ${NGINX_PID})"
sleep 1

echo ""
echo "=========================================="
echo "✓ 所有服务已启动！"
echo "前端地址: http://localhost:8080"
echo "API 地址: http://localhost:8000/api"
echo "=========================================="

# 信号处理：优雅关闭所有服务
cleanup() {
    echo ""
    echo "收到停止信号，正在关闭所有服务..."
    kill ${SERVER_PID} ${CONTROLLER_PID} ${NGINX_PID} 2>/dev/null || true
    wait ${SERVER_PID} ${CONTROLLER_PID} ${NGINX_PID} 2>/dev/null || true
    echo "所有服务已停止"
    exit 0
}

trap cleanup TERM INT

# 等待任意一个进程退出
wait -n ${SERVER_PID} ${CONTROLLER_PID} ${NGINX_PID}
EXIT_CODE=$?

echo ""
echo "检测到进程退出，退出码: ${EXIT_CODE}"
cleanup
EOF

RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["tini", "--", "/entrypoint.sh"]
