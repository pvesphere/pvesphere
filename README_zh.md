# PVESphere

[![license](https://img.shields.io/github/license/pvesphere/pvesphere-ui.svg)](LICENSE)

**中文** | [English](./README.md)

## 项目简介

PVESphere 是一个基于 Web 的 Proxmox VE (PVE) 集群综合管理平台。它提供了一个现代化、直观的界面，用于从统一的仪表板管理多个 PVE 集群、节点、虚拟机、存储和模板。
<!-- <img src="./docs/pvesphere-review-rc01.gif" width="100%" /> -->

## 功能特性

### 🎯 核心功能

- **资源总览**: 实时监控集群资源、健康状态和利用率指标
- **集群管理**: 集中式认证和配置管理多个 PVE 集群
- **节点管理**: 跨集群监控和管理物理节点，包括控制台访问
- **虚拟机管理**: 完整的虚拟机生命周期管理，包括创建、启动、停止、迁移、备份和恢复
- **存储管理**: 监控存储使用情况，管理存储池，查看存储内容
- **模板管理**: 导入、同步和管理虚拟机模板，实现快速部署

### 🚀 主要能力

- 多集群支持，统一管理界面
- 实时资源监控和指标可视化
- 通过 VNC/NoVNC 访问虚拟机控制台
- Cloud-Init 配置支持
- 自动化模板同步
- 备份和恢复功能
- 任务监控和管理
- 响应式设计，支持移动端

## 技术栈

### 前端 (`pvesphere-ui`)

- **框架**: Vue 3 (组合式 API)
- **构建工具**: Vite 7
- **UI 组件库**: Element Plus 2
- **语言**: TypeScript 5
- **状态管理**: Pinia 3
- **路由**: Vue Router 4
- **国际化**: Vue I18n
- **图表**: ECharts 6
- **终端**: xterm.js, @novnc/novnc
- **样式**: Tailwind CSS 4, SCSS

### 后端 (`pvesphere`)

- **语言**: Go 1.23
- **Web 框架**: Gin 1.10
- **ORM**: GORM 1.30
- **数据库**: SQLite / MySQL / PostgreSQL
- **身份认证**: JWT (golang-jwt/jwt/v5)
- **任务调度**: gocron
- **日志**: Zap
- **API 文档**: Swagger
- **依赖注入**: Google Wire

## 项目结构

### 前端结构

```
pvesphere-ui/
├── src/
│   ├── api/              # API 接口
│   ├── components/       # 可复用组件
│   ├── layout/           # 布局组件
│   ├── router/           # 路由配置
│   ├── store/            # Pinia 状态管理
│   ├── views/            # 页面组件
│   │   └── pve/          # PVE 管理页面
│   │       ├── dashboard/ # 资源总览
│   │       ├── cluster/   # 集群管理
│   │       ├── node/      # 节点管理
│   │       ├── vm/        # 虚拟机管理
│   │       ├── storage/   # 存储管理
│   │       └── template/  # 模板管理
│   ├── utils/            # 工具函数
│   └── plugins/          # 插件配置
├── locales/              # 国际化语言文件
└── public/               # 静态资源
```

### 后端结构

```
pvesphere/
├── api/v1/               # API 路由处理器
├── cmd/                   # 应用入口
│   ├── server/            # HTTP 服务器
│   ├── controller/        # PVE 控制器
│   ├── task/              # 后台任务
│   └── migration/         # 数据库迁移
├── internal/
│   ├── handler/           # 业务逻辑处理器
│   ├── service/           # 业务服务
│   ├── repository/        # 数据访问层
│   ├── model/             # 数据模型
│   ├── middleware/        # HTTP 中间件
│   └── router/            # 路由定义
├── pkg/                   # 共享包
│   ├── proxmox/           # PVE API 客户端
│   ├── config/            # 配置管理
│   └── log/               # 日志工具
└── docs/                  # 文档
```

## 快速开始

### 环境要求

- **前端**: Node.js >= 20.19.0 或 >= 22.13.0, pnpm >= 9
- **后端**: Go >= 1.23
- **Docker** (可选): Docker >= 20.10, Docker Compose >= 2.0

### 前端启动

```bash
cd pvesphere-ui
pnpm install
pnpm dev
```

### 后端启动

```bash
cd pvesphere
go mod download

# 运行数据库迁移（会自动创建默认用户）
go run cmd/migration/main.go

# 启动服务器
go run cmd/server/main.go
```

### 默认用户信息

首次运行数据库迁移后，系统会自动创建默认管理员账户，可直接使用以下信息登录：

- **邮箱**: `pvesphere@gmail.com`
- **密码**: `Ab123456`
- **昵称**: `PveSphere Admin`

> 注意：如果默认用户已存在，迁移过程不会重复创建。建议首次登录后及时修改密码。

### 生产构建

**前端:**
```bash
cd pvesphere-ui
pnpm build
```

**后端:**
```bash
cd pvesphere
go build -o bin/server cmd/server/main.go
```

## Docker 部署

### 环境要求

- Docker >= 20.10
- Docker Compose >= 2.0

### 快速启动（推荐）

使用 Makefile 命令快速构建并启动所有服务：

```bash
# 构建并启动所有服务（包括数据库迁移）
make docker-compose-build

# 查看服务状态
make docker-compose-ps

# 查看服务日志
make docker-compose-logs

# 停止所有服务
make docker-compose-down
```

### Docker 镜像构建

#### 构建单个服务镜像

```bash
# 构建 API 服务镜像
make docker-build-api

# 构建控制器服务镜像
make docker-build-controller

# 构建所有服务镜像
make docker-build
```

#### 手动构建镜像

```bash
# 构建 API 服务
docker build -f deploy/build/Dockerfile \
  --build-arg APP_RELATIVE_PATH=./cmd/server \
  --build-arg APP_NAME=server \
  --build-arg APP_ENV=prod \
  -t pvesphere-api:latest .

# 构建控制器服务
docker build -f deploy/build/Dockerfile \
  --build-arg APP_RELATIVE_PATH=./cmd/controller \
  --build-arg APP_NAME=controller \
  --build-arg APP_ENV=prod \
  -t pvesphere-controller:latest .
```

### Docker Compose 使用

项目使用 Docker Compose 管理服务，默认使用 SQLite 数据库。

#### 常用命令

```bash
# 启动所有服务
make docker-compose-up

# 构建并启动（首次运行）
make docker-compose-build

# 查看服务状态
make docker-compose-ps

# 查看所有服务日志
make docker-compose-logs

# 查看 API 服务日志
make docker-compose-logs-api

# 查看控制器服务日志
make docker-compose-logs-controller

# 重启所有服务
make docker-compose-restart

# 停止服务（保留容器）
make docker-compose-stop

# 启动已停止的服务
make docker-compose-start

# 停止并删除所有服务
make docker-compose-down
```

#### 服务说明

- **api-server**: API 服务（端口 8000）
- **controller**: 控制器服务
- **migration**: 数据库迁移服务（自动运行）

#### 访问服务

- **API 服务**: http://localhost:8000
- **API 文档**: http://localhost:8000/swagger/index.html

#### 默认用户信息

首次运行数据库迁移后，系统会自动创建默认管理员账户，可直接使用以下信息登录：

- **邮箱**: `pvesphere@gmail.com`
- **密码**: `Ab123456`
- **昵称**: `PveSphere Admin`

> 注意：如果默认用户已存在，迁移过程不会重复创建。建议首次登录后及时修改密码。

#### 数据持久化

所有数据（数据库、日志）存储在 Docker volume `pvesphere-storage` 中，容器重启后数据不会丢失。

### 本地开发（使用 Makefile）

项目提供了便捷的 Makefile 命令用于本地开发：

```bash
# 初始化开发环境（安装工具）
make init

# 本地启动（需要本地 Go 环境）
# 1. 启动依赖服务（MySQL、Redis）
# 2. 运行数据库迁移
# 3. 启动 API 服务
make bootstrap

# 构建本地二进制文件
make build              # 构建所有服务
make build-server       # 仅构建 API 服务
make build-controller   # 仅构建控制器服务

# 运行测试
make test

# 生成 Swagger 文档
make swag
```

### 数据库迁移

#### Docker 环境

数据库迁移会在服务启动时自动运行。如果需要手动运行：

```bash
# 使用 docker compose 运行迁移
cd deploy/docker-compose
docker compose run --rm migration

# 或在容器中运行
docker exec -it pvesphere-api ./migration -conf /data/app/config/docker.yml
```

#### 本地环境

```bash
# 使用 go run
go run ./cmd/migration -conf config/local.yml

# 或使用 nunu
nunu run ./cmd/migration -conf config/local.yml
```

### 推送镜像到仓库

```bash
# 推送 API 服务镜像
make docker-push-api REGISTRY=your-registry.com/pvesphere

# 推送控制器服务镜像
make docker-push-controller REGISTRY=your-registry.com/pvesphere

# 推送所有服务镜像
make docker-push REGISTRY=your-registry.com/pvesphere
```

更多 Docker 使用说明请参考 [deploy/docker-compose/README.md](deploy/docker-compose/README.md)

## 配置说明

### 前端配置

前端配置文件位于 `src/config/index.ts`，可以配置：
- API 基础地址
- 请求超时时间
- 其他应用设置

### 后端配置

后端使用 Viper 进行配置管理，配置文件位于 `config/` 目录：
- `config/local.yml` - 本地开发配置
- `config/prod.yml` - 生产环境配置

主要配置项：
- 数据库连接设置
- JWT 密钥
- PVE API 端点
- 服务器主机和端口

## API 文档

后端服务器运行后，可以通过以下地址访问 Swagger API 文档：
```
http://localhost:8000/swagger/index.html
```

## 主要组件

### 前端组件

- **ReIcon**: 图标组件，支持 Iconify 和 iconfont
- **ReDialog**: 增强型对话框组件
- **RePureTableBar**: 表格工具栏组件
- **ReAuth**: 权限控制组件
- **RePerms**: 权限指令组件
- **ReCol**: 响应式列组件
- **ReSegmented**: 分段控制器组件

### 后端服务

- **PVE Controller**: 管理 PVE 集群连接和操作
- **Dashboard Service**: 提供总览统计和指标
- **VM Service**: 处理虚拟机操作
- **Storage Service**: 管理存储资源
- **Template Service**: 处理模板导入和同步
- **Task Service**: 管理后台任务和作业调度

## 开发指南

### 代码规范

- 前端遵循 Vue 3 组合式 API 最佳实践
- 后端遵循 Go 标准项目布局
- 两个项目都使用 ESLint/Prettier 进行代码格式化

### Git 提交规范

遵循约定式提交格式：
```
<type>(<scope>): <subject>
```

类型：`feat`、`fix`、`docs`、`style`、`refactor`、`perf`、`test`、`chore`、`revert`、`build`

## 许可证

[Apache License 2.0](LICENSE)

版权所有 © 2025-present PveSphere Contributors

## 贡献

欢迎贡献代码！请随时提交 Pull Request。