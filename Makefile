.PHONY: init
init:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golang/mock/mockgen@latest
	go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: bootstrap
bootstrap:
	cd ./deploy/docker-compose && docker compose up -d && cd ../../
	go run ./cmd/migration
	nunu run ./cmd/server

.PHONY: mock
mock:
	mockgen -source=internal/service/user.go -destination test/mocks/service/user.go
	mockgen -source=internal/repository/user.go -destination test/mocks/repository/user.go
	mockgen -source=internal/repository/repository.go -destination test/mocks/repository/repository.go

.PHONY: test
test:
	go test -coverpkg=./internal/handler,./internal/service,./internal/repository -coverprofile=./coverage.out ./test/server/...
	go tool cover -html=./coverage.out -o coverage.html

.PHONY: build
build:
	go build -ldflags="-s -w" -o ./bin/server ./cmd/server
	go build -ldflags="-s -w" -o ./bin/controller ./cmd/controller

.PHONY: build-server
build-server:
	go build -ldflags="-s -w" -o ./bin/server ./cmd/server

.PHONY: build-controller
build-controller:
	go build -ldflags="-s -w" -o ./bin/controller ./cmd/controller

# Docker 相关命令
# 镜像标签（优先使用 VERSION，其次 git 标签，默认 latest）
IMAGE_TAG ?= $(shell if [ -f VERSION ]; then cat VERSION; else git describe --tags --always 2>/dev/null || echo "latest"; fi)
IMAGE_TAG := $(strip $(IMAGE_TAG))

# 前端本地构建目录检查
FRONTEND_LOCAL_DIR := deploy/build/.frontend-local
FRONTEND_LOCAL_PKG := $(FRONTEND_LOCAL_DIR)/package.json

.PHONY: docker-build-aio
docker-build-aio:
	@echo "构建 All-in-One 快速开始镜像..."
	docker build -f deploy/build/Dockerfile \
		--target allinone \
		-t pvesphere-aio:latest \
		-t pvesphere-aio:$(IMAGE_TAG) \
		.

.PHONY: docker-build-api
docker-build-api:
	@echo "构建 API 服务 Docker 镜像..."
	docker build -f deploy/build/Dockerfile \
		--target backend \
		--build-arg APP_RELATIVE_PATH=./cmd/server \
		--build-arg APP_NAME=server \
		--build-arg APP_ENV=prod \
		-t pvesphere-api:latest \
		-t pvesphere-api:$(IMAGE_TAG) \
		.

.PHONY: docker-build-controller
docker-build-controller:
	@echo "构建控制器服务 Docker 镜像..."
	docker build -f deploy/build/Dockerfile \
		--target backend \
		--build-arg APP_RELATIVE_PATH=./cmd/controller \
		--build-arg APP_NAME=controller \
		--build-arg APP_ENV=prod \
		-t pvesphere-controller:latest \
		-t pvesphere-controller:$(IMAGE_TAG) \
		.

.PHONY: docker-build-frontend
docker-build-frontend:
	@echo "构建前端服务 Docker 镜像..."
	docker build -f deploy/build/Dockerfile \
		--target frontend \
		-t pvesphere-frontend:latest \
		-t pvesphere-frontend:$(IMAGE_TAG) \
		.

.PHONY: docker-build
docker-build: docker-build-api docker-build-controller docker-build-frontend
	@echo "所有服务 Docker 镜像构建完成"

# 镜像仓库地址，可通过环境变量 REGISTRY 指定，默认为空（本地构建）
REGISTRY ?= 

.PHONY: docker-push-api
docker-push-api:
	@if [ -z "$(REGISTRY)" ]; then \
		echo "错误: 请设置 REGISTRY 环境变量，例如: make docker-push-api REGISTRY=registry.example.com/pvesphere"; \
		exit 1; \
	fi
	@echo "推送 API 服务镜像到仓库: $(REGISTRY)/pvesphere-api:{latest,$(IMAGE_TAG)}"
	docker tag pvesphere-api:latest $(REGISTRY)/pvesphere-api:latest
	docker tag pvesphere-api:$(IMAGE_TAG) $(REGISTRY)/pvesphere-api:$(IMAGE_TAG)
	docker push $(REGISTRY)/pvesphere-api:latest
	docker push $(REGISTRY)/pvesphere-api:$(IMAGE_TAG)

.PHONY: docker-push-controller
docker-push-controller:
	@if [ -z "$(REGISTRY)" ]; then \
		echo "错误: 请设置 REGISTRY 环境变量，例如: make docker-push-controller REGISTRY=registry.example.com/pvesphere"; \
		exit 1; \
	fi
	@echo "推送控制器服务镜像到仓库: $(REGISTRY)/pvesphere-controller:{latest,$(IMAGE_TAG)}"
	docker tag pvesphere-controller:latest $(REGISTRY)/pvesphere-controller:latest
	docker tag pvesphere-controller:$(IMAGE_TAG) $(REGISTRY)/pvesphere-controller:$(IMAGE_TAG)
	docker push $(REGISTRY)/pvesphere-controller:latest
	docker push $(REGISTRY)/pvesphere-controller:$(IMAGE_TAG)

.PHONY: docker-push-aio
docker-push-aio:
	@if [ -z "$(REGISTRY)" ]; then \
		echo "错误: 请设置 REGISTRY 环境变量，例如: make docker-push-aio REGISTRY=registry.example.com/pvesphere"; \
		exit 1; \
	fi
	@echo "推送 All-in-One 镜像到仓库: $(REGISTRY)/pvesphere-aio:{latest,$(IMAGE_TAG)}"
	docker tag pvesphere-aio:latest $(REGISTRY)/pvesphere-aio:latest
	docker tag pvesphere-aio:$(IMAGE_TAG) $(REGISTRY)/pvesphere-aio:$(IMAGE_TAG)
	docker push $(REGISTRY)/pvesphere-aio:latest
	docker push $(REGISTRY)/pvesphere-aio:$(IMAGE_TAG)

.PHONY: docker-push-frontend
docker-push-frontend:
	@if [ -z "$(REGISTRY)" ]; then \
		echo "错误: 请设置 REGISTRY 环境变量，例如: make docker-push-frontend REGISTRY=registry.example.com/pvesphere"; \
		exit 1; \
	fi
	@echo "推送前端服务镜像到仓库: $(REGISTRY)/pvesphere-frontend:{latest,$(IMAGE_TAG)}"
	docker tag pvesphere-frontend:latest $(REGISTRY)/pvesphere-frontend:latest
	docker tag pvesphere-frontend:$(IMAGE_TAG) $(REGISTRY)/pvesphere-frontend:$(IMAGE_TAG)
	docker push $(REGISTRY)/pvesphere-frontend:latest
	docker push $(REGISTRY)/pvesphere-frontend:$(IMAGE_TAG)

.PHONY: docker-push
docker-push: docker-push-api docker-push-controller docker-push-frontend docker-push-aio
	@echo "所有服务镜像推送完成"

.PHONY: docker-run-api
docker-run-api:
	docker run --rm -it \
		-p 8000:8000 \
		-v $$(pwd)/config:/data/app/config \
		pvesphere-api:latest

.PHONY: docker-run-controller
docker-run-controller:
	docker run --rm -it \
		-v $$(pwd)/config:/data/app/config \
		pvesphere-controller:latest

# Docker Compose 相关命令
.PHONY: docker-compose-up
docker-compose-up:
	@echo "启动所有服务（包括 API、Controller 和 Frontend）..."
	cd deploy/docker-compose && docker compose up -d

.PHONY: docker-compose-down
docker-compose-down:
	@echo "停止所有服务..."
	cd deploy/docker-compose && docker compose down

.PHONY: docker-compose-build
docker-compose-build:
	@if [ ! -f "$(FRONTEND_LOCAL_PKG)" ]; then \
		echo "前端构建产物未准备好: 未找到 $(FRONTEND_LOCAL_PKG)"; \
		echo "请先执行 ./deploy/build.sh -f /path/to/pvesphere-ui 准备前端代码，再运行 make docker-compose-build"; \
		exit 1; \
	fi
	@echo "构建并启动所有服务..."
	cd deploy/docker-compose && docker compose up -d --build

.PHONY: docker-compose-logs
docker-compose-logs:
	@echo "查看所有服务日志..."
	cd deploy/docker-compose && docker compose logs -f

.PHONY: docker-compose-logs-api
docker-compose-logs-api:
	@echo "查看 API 服务日志..."
	cd deploy/docker-compose && docker compose logs -f api-server

.PHONY: docker-compose-logs-controller
docker-compose-logs-controller:
	@echo "查看控制器服务日志..."
	cd deploy/docker-compose && docker compose logs -f controller

.PHONY: docker-compose-logs-frontend
docker-compose-logs-frontend:
	@echo "查看前端服务日志..."
	cd deploy/docker-compose && docker compose logs -f frontend

.PHONY: docker-compose-ps
docker-compose-ps:
	@echo "查看服务状态..."
	cd deploy/docker-compose && docker compose ps

.PHONY: docker-compose-restart
docker-compose-restart:
	@echo "重启所有服务..."
	cd deploy/docker-compose && docker compose restart

.PHONY: docker-compose-restart-api
docker-compose-restart-api:
	@echo "重启 API 服务..."
	cd deploy/docker-compose && docker compose restart api-server

.PHONY: docker-compose-restart-controller
docker-compose-restart-controller:
	@echo "重启控制器服务..."
	cd deploy/docker-compose && docker compose restart controller

.PHONY: docker-compose-restart-frontend
docker-compose-restart-frontend:
	@echo "重启前端服务..."
	cd deploy/docker-compose && docker compose restart frontend

.PHONY: docker-compose-stop
docker-compose-stop:
	@echo "停止所有服务（不删除容器）..."
	cd deploy/docker-compose && docker compose stop

.PHONY: docker-compose-start
docker-compose-start:
	@echo "启动已停止的服务..."
	cd deploy/docker-compose && docker compose start

# Docker Compose 健康检查
.PHONY: docker-compose-health
docker-compose-health:
	@echo "检查所有服务健康状态..."
	@cd deploy/docker-compose && docker compose ps
	@echo "\n前端健康检查:"
	@curl -s http://localhost:8080/health || echo "前端服务未就绪"
	@echo "\nAPI 健康检查:"
	@curl -s http://localhost:8000/api/health || echo "API 服务未就绪"

# 快速启动（推荐）
.PHONY: up
up: docker-compose-build
	@echo "\n==================================="
	@echo "所有服务已启动！"
	@echo "前端地址: http://localhost:8080"
	@echo "API 地址: http://localhost:8000"
	@echo "==================================="
	@echo "\n使用 'make logs' 查看日志"
	@echo "使用 'make ps' 查看服务状态"
	@echo "使用 'make down' 停止所有服务"

# 快速停止
.PHONY: down
down: docker-compose-down

# 快速查看日志
.PHONY: logs
logs: docker-compose-logs

# 快速查看状态
.PHONY: ps
ps: docker-compose-ps

# Swagger 文档生成
.PHONY: swag
swag:
	swag init  -g cmd/server/main.go -o ./docs
