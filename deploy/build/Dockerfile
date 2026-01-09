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
# 一次性构建所有二进制文件，避免多次构建时的缓存问题
RUN export GOPROXY=https://goproxy.cn,direct && \
    go mod download && \
    go mod tidy && \
    echo "=== Building all services ===" && \
    echo "Building server..." && \
    go build -ldflags="-s -w" -o ./bin/server ./cmd/server && \
    echo "Building controller..." && \
    go build -ldflags="-s -w" -o ./bin/controller ./cmd/controller && \
    echo "Building migration..." && \
    go build -ldflags="-s -w" -o ./bin/migration ./cmd/migration && \
    echo "=== Build complete ===" && \
    echo "Checking binaries:" && \
    ls -lh ./bin/

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

# 检查并复制前端代码
RUN echo "=== 检查前端代码 ===" && \
    ls -la /tmp/frontend-local/ && \
    if [ ! -d "/tmp/frontend-local" ] || [ ! -f "/tmp/frontend-local/package.json" ]; then \
        echo ""; \
        echo "✗ 错误：前端代码目录为空或缺少 package.json"; \
        echo ""; \
        echo "请先运行构建脚本准备前端代码:"; \
        echo "  cd deploy"; \
        echo "  ./build.sh -f /path/to/pvesphere-ui"; \
        echo ""; \
        exit 1; \
    fi && \
    echo "✓ 使用本地前端代码" && \
    cp -r /tmp/frontend-local/* . && \
    rm -rf /tmp/frontend-local

# 创建/覆盖 .env.production 文件，配置 API 地址为空（使用相对路径，通过 Nginx 代理）
# 必须在复制代码后立即创建，确保覆盖前端代码中的配置
# 注意：由于前端代码使用 || 操作符，空字符串会被当作 falsy，所以使用特殊值 "/api/v1"
# 这样前端会使用相对路径，通过 Nginx 代理
RUN echo "=== 创建生产环境配置 ===" && \
    echo '# 生产环境配置（Docker 部署）' > .env.production && \
    echo '# API 地址使用相对路径，通过 Nginx 代理到后端' >> .env.production && \
    echo 'VITE_API_BASE_URL=' >> .env.production && \
    echo 'VITE_BASE_API=/api/v1' >> .env.production && \
    echo 'VITE_PORT=4173' >> .env.production && \
    echo 'VITE_CDN=false' >> .env.production && \
    echo 'VITE_COMPRESSION=gzip' >> .env.production && \
    echo 'VITE_PUBLIC_PATH=/' >> .env.production && \
    echo 'VITE_ROUTER_HISTORY=hash' >> .env.production && \
    echo 'VITE_HIDE_HOME=false' >> .env.production && \
    echo "=== 验证 .env.production ===" && \
    cat .env.production

# 配置 pnpm 使用国内镜像并安装依赖
RUN echo "=== 检查项目文件 ===" && \
    ls -la | head -20 && \
    if [ ! -f "package.json" ]; then \
        echo "✗ 错误：找不到 package.json"; \
        exit 1; \
    fi && \
    echo "✓ package.json 存在" && \
    echo "=== 检查 .npmrc 配置 ===" && \
    if [ -f ".npmrc" ]; then \
        echo "✓ 找到 .npmrc 配置:"; \
        cat .npmrc; \
    else \
        echo "⚠ 未找到 .npmrc，创建默认配置"; \
        echo "shamefully-hoist=true" > .npmrc && \
        echo "strict-peer-dependencies=false" >> .npmrc; \
    fi && \
    echo "=== 配置 pnpm 镜像 ===" && \
    pnpm config set registry https://registry.npmmirror.com && \
    pnpm config get registry && \
    echo "=== 确保 pnpm 配置正确 ===" && \
    pnpm config set shamefully-hoist true && \
    pnpm config set strict-peer-dependencies false && \
    echo "=== 清理旧依赖（如果有） ===" && \
    rm -rf node_modules .pnpm-store 2>/dev/null || true && \
    echo "=== 安装依赖 ===" && \
    pnpm install --frozen-lockfile || \
    (echo "✗ pnpm install 失败，尝试不使用 frozen-lockfile" && \
     pnpm install || (echo "✗ pnpm install 完全失败" && exit 1)) && \
    echo "=== 验证依赖安装 ===" && \
    if [ ! -d "node_modules" ]; then \
        echo "✗ 错误：node_modules 目录不存在"; \
        exit 1; \
    fi && \
    echo "✓ 依赖安装完成" && \
    echo "=== 检查关键依赖 ===" && \
    (test -d "node_modules/vue-demi" && echo "✓ vue-demi 已安装在 node_modules 根目录" || \
     (echo "⚠ vue-demi 不在根目录，查找位置:" && \
      find node_modules -name "vue-demi" -type d 2>/dev/null | head -3)) && \
    echo "=== 构建项目 ===" && \
    pnpm run build || (echo "✗ pnpm build 失败" && exit 1) && \
    echo "✓ 构建完成"

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
RUN echo "=== 检查构建的二进制文件 ===" && \
    ls -lh /data/app/ && \
    for binary in server controller migration; do \
        if [ ! -f "/data/app/$binary" ]; then \
            echo "ERROR: Binary /data/app/$binary not found!"; \
            ls -la /data/app/; \
            exit 1; \
        fi; \
    done && \
    echo "SUCCESS: All binaries found" && \
    echo "Default APP_NAME=${APP_NAME}" && \
    chmod +x /data/app/*

# 默认命令（可在 docker-compose 中覆盖）
# 使用 sh -c 来支持环境变量展开
CMD sh -c "echo 'Starting ${APP_NAME}...' && ./${APP_NAME}"

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
