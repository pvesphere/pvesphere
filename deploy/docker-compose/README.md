# Docker Compose 使用指南

本目录包含使用 Docker Compose 启动 PveSphere 所有服务的配置文件。

## 服务说明

- **api-server**: PveSphere API 服务（端口 8000）
- **controller**: PveSphere 控制器服务

**数据库**: 默认使用 SQLite3，数据存储在 Docker volume 中，两个服务共享同一个数据库文件。

> 注意：如果需要使用 MySQL 或 Redis，可以取消注释 `docker-compose.yml` 中相应的服务配置，并修改 `config/docker.yml` 中的数据库配置。

## 快速开始

### 1. 启动所有服务

```bash
# 方式一：使用 Makefile（推荐）
make docker-compose-up

# 方式二：直接使用 docker compose
cd deploy/docker-compose
docker compose up -d
```

### 2. 构建并启动服务（首次运行或代码更新后）

```bash
# 方式一：使用 Makefile（推荐）
make docker-compose-build

# 方式二：直接使用 docker compose
cd deploy/docker-compose
docker compose up -d --build
```

### 3. 查看服务状态

```bash
# 方式一：使用 Makefile
make docker-compose-ps

# 方式二：直接使用 docker compose
cd deploy/docker-compose
docker compose ps
```

### 4. 查看服务日志

```bash
# 查看所有服务日志
make docker-compose-logs

# 查看 API 服务日志
make docker-compose-logs-api

# 查看控制器服务日志
make docker-compose-logs-controller

# 直接使用 docker compose
cd deploy/docker-compose
docker compose logs -f [service-name]
```

### 5. 停止服务

```bash
# 停止所有服务（保留容器）
make docker-compose-stop

# 停止并删除所有服务
make docker-compose-down
```

### 6. 重启服务

```bash
make docker-compose-restart
```

## 数据库迁移

数据库迁移会在服务启动时自动运行。`docker-compose.yml` 中配置了 `migration` 服务，它会在 API 和 Controller 服务启动前自动执行。

### 自动迁移

使用 `docker-compose` 启动服务时，迁移会自动运行：

```bash
make docker-compose-build
# 或
cd deploy/docker-compose && docker compose up -d --build
```

迁移服务会：
1. 自动运行并创建数据库表
2. 完成后自动退出
3. API 和 Controller 服务会在迁移完成后启动

### 手动运行迁移

如果需要手动运行迁移：

```bash
# 方式一：使用 docker compose 运行迁移服务
cd deploy/docker-compose
docker compose run --rm migration

# 方式二：在已运行的容器中执行
docker exec -it pvesphere-api ./migration -conf /data/app/config/docker.yml

# 方式三：在容器外运行（需要本地 Go 环境）
go run ./cmd/migration -conf config/docker.yml
# 或使用 nunu
nunu run ./cmd/migration -conf config/docker.yml
```

### 查看迁移日志

```bash
# 查看迁移服务日志
docker compose logs migration

# 或查看所有服务日志
make docker-compose-logs
```

## 访问服务

- **API 服务**: http://localhost:8000
- **API 文档**: http://localhost:8000/swagger/index.html

## 配置文件

服务使用 `config/docker.yml` 配置文件，默认配置为：
- **数据库**: SQLite3，存储在 `/data/app/storage/pvesphere-test.db`
- **日志**: 存储在 `/data/app/storage/logs/` 目录

### 切换到 MySQL/Redis

如果需要使用 MySQL 或 Redis：

1. 取消注释 `docker-compose.yml` 中的 `user-db` 和 `cache-redis` 服务
2. 修改 `config/docker.yml` 中的数据库配置，取消注释 MySQL/Redis 相关配置
3. 在服务配置中添加对数据库和 Redis 的依赖

## 数据持久化

以下数据会持久化到 Docker volume：
- `pvesphere-storage`: SQLite 数据库文件和日志（两个服务共享）

### 查看数据

```bash
# 查看 volume 信息
docker volume inspect pvesphere-storage

# 进入容器查看数据库文件
docker exec -it pvesphere-api ls -lh /data/app/storage/

# 备份数据库（在容器内）
docker exec -it pvesphere-api cp /data/app/storage/pvesphere-test.db /data/app/storage/pvesphere-test.db.backup
```

## 常用命令

```bash
# 进入 API 服务容器
docker exec -it pvesphere-api sh

# 进入控制器服务容器
docker exec -it pvesphere-controller sh

# 查看 SQLite 数据库（需要安装 sqlite3）
docker exec -it pvesphere-api sh -c "apk add sqlite && sqlite3 /data/app/storage/pvesphere-test.db '.tables'"

# 查看数据库文件大小
docker exec -it pvesphere-api ls -lh /data/app/storage/pvesphere-test.db

# 查看日志文件
docker exec -it pvesphere-api tail -f /data/app/storage/logs/server.log
```

## 故障排查

### 服务无法启动

1. 检查服务日志：`make docker-compose-logs`
2. 检查服务状态：`make docker-compose-ps`
3. 检查端口占用：确保 8000 端口未被占用
4. 检查构建日志：`docker compose build --no-cache`

### 数据库相关问题

1. **数据库文件权限问题**
   ```bash
   # 检查数据库文件权限
   docker exec -it pvesphere-api ls -l /data/app/storage/
   
   # 修复权限（如果需要）
   docker exec -it pvesphere-api chmod 666 /data/app/storage/pvesphere-test.db
   ```

2. **数据库锁定问题**
   - SQLite 使用 WAL 模式支持并发访问
   - 如果遇到锁定问题，检查是否有其他进程占用数据库文件
   - 查看日志：`docker logs pvesphere-api`

3. **数据库文件损坏**
   ```bash
   # 检查数据库完整性
   docker exec -it pvesphere-api sh -c "apk add sqlite && sqlite3 /data/app/storage/pvesphere-test.db 'PRAGMA integrity_check;'"
   ```

### 配置文件问题

1. 确保 `config/docker.yml` 文件存在且配置正确
2. 检查数据库路径：`storage/pvesphere-test.db`
3. 验证配置文件挂载：`docker exec -it pvesphere-api cat /data/app/config/docker.yml`

### SQLite 并发访问说明

- SQLite 使用 WAL（Write-Ahead Logging）模式，支持多进程并发读取
- 写入操作会自动加锁，确保数据一致性
- 两个服务（API 和 Controller）可以安全地共享同一个数据库文件

