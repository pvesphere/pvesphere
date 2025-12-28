ARG REGISTRY=docker.io
FROM ${REGISTRY}/golang:1.23-alpine AS builder
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

# 安装必要的构建工具
RUN apk add --no-cache git make

ARG APP_RELATIVE_PATH
ARG APP_NAME=server

# 复制项目文件
COPY . /data/app
WORKDIR /data/app

# 清理旧的构建产物
RUN rm -rf /data/app/bin/

# 下载依赖并构建
RUN export GOPROXY=https://goproxy.cn,direct && \
    go mod download && \
    go mod tidy && \
    go build -ldflags="-s -w" -o ./bin/${APP_NAME} ${APP_RELATIVE_PATH} && \
    go build -ldflags="-s -w" -o ./bin/migration ./cmd/migration

# 复制配置文件
RUN cp -r config /data/app/bin/ || true

# 运行时镜像
FROM ${REGISTRY}/alpine:3.19
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

# 设置时区
RUN apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

ARG APP_ENV=prod
ENV APP_ENV=${APP_ENV}

WORKDIR /data/app

# 从构建阶段复制二进制文件和配置
COPY --from=builder /data/app/bin /data/app

# 默认暴露 8000 端口（API 服务）
EXPOSE 8000

# 使用环境变量指定可执行文件名（从构建阶段传递）
ARG APP_NAME=server
ENV APP_NAME=${APP_NAME}

# 默认命令（可在 docker-compose 中覆盖）
CMD ["./server"]
