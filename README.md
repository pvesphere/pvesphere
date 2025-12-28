# PveSphere

[![license](https://img.shields.io/github/license/pvesphere/pvesphere-ui.svg)](LICENSE)

**English** | [中文](#中文)

---

## Introduction

PveSphere is a comprehensive web-based management platform for Proxmox VE (PVE) clusters. It provides a modern, intuitive interface for managing multiple PVE clusters, nodes, virtual machines, storage, and templates from a single unified dashboard.

### What is PveSphere?

PveSphere is a multi-cluster management solution that enables centralized control and monitoring of Proxmox VE infrastructure. It simplifies the complexity of managing distributed PVE environments by providing a single pane of glass for all your virtualization resources.

---

## Features

### 🎯 Core Functionality

- **Dashboard Overview**: Real-time monitoring of cluster resources, health status, and utilization metrics
- **Cluster Management**: Manage multiple PVE clusters with centralized authentication and configuration
- **Node Management**: Monitor and manage physical nodes across clusters, including console access
- **Virtual Machine Management**: Full lifecycle management of VMs including create, start, stop, migrate, backup, and restore
- **Storage Management**: Monitor storage usage, manage storage pools, and view storage content
- **Template Management**: Import, sync, and manage VM templates for rapid deployment

### 🚀 Key Capabilities

- Multi-cluster support with unified management interface
- Real-time resource monitoring and metrics visualization
- VM console access via VNC/NoVNC
- Node console access via terminal proxy
- Cloud-Init configuration support
- Automated template synchronization
- Backup and restore functionality
- Task monitoring and management
- Network configuration management
- Service management (start, stop, restart)
- Responsive design with mobile support
- Internationalization (i18n) support

---

## Technology Stack

### Frontend (`pvesphere-ui`)

- **Framework**: Vue 3 (Composition API)
- **Build Tool**: Vite 7
- **UI Library**: Element Plus 2
- **Language**: TypeScript 5
- **State Management**: Pinia 3
- **Routing**: Vue Router 4
- **Internationalization**: Vue I18n
- **Charts**: ECharts 6
- **Terminal**: xterm.js 5, noVNC
- **Styling**: Tailwind CSS 4, SCSS
- **Base Template**: [vue-pure-admin](https://github.com/pure-admin/vue-pure-admin)

### Backend (`pvesphere`)

- **Language**: Go 1.23
- **Web Framework**: Gin 1.10
- **ORM**: GORM 1.30
- **Database**: MySQL / PostgreSQL / SQLite
- **Authentication**: JWT (golang-jwt/jwt)
- **Cache**: Redis 9
- **WebSocket**: Gorilla WebSocket
- **Task Scheduling**: gocron
- **Logging**: Zap
- **API Documentation**: Swagger
- **Architecture**: Based on [Nunu](https://github.com/go-nunu/nunu) framework

---

## Project Structure

### Frontend Structure

```
pvesphere-ui/
├── src/
│   ├── api/              # API interfaces
│   ├── assets/          # Static resources
│   ├── components/      # Reusable components
│   ├── config/          # Configuration files
│   ├── directives/      # Vue directives
│   ├── layout/          # Layout components
│   ├── plugins/         # Plugin configurations
│   ├── router/          # Route configuration
│   ├── store/           # Pinia stores
│   ├── style/           # Global styles
│   ├── utils/           # Utility functions
│   └── views/           # Page components
│       └── pve/         # PVE management pages
│           ├── cluster/     # Cluster management
│           ├── dashboard/   # Dashboard
│           ├── node/        # Node management
│           ├── storage/     # Storage management
│           ├── template/    # Template management
│           └── vm/          # VM management
├── locales/             # i18n translation files
└── package.json
```

### Backend Structure

```
pvesphere/
├── api/v1/              # API route handlers
├── cmd/                 # Application entry points
│   ├── server/          # HTTP server
│   ├── controller/      # Kubernetes controller
│   ├── migration/       # Database migration
│   └── task/            # Background tasks
├── internal/
│   ├── handler/         # Request handlers
│   ├── service/         # Business logic
│   ├── repository/      # Data access layer
│   ├── model/           # Data models
│   ├── middleware/      # HTTP middleware
│   └── router/          # Route definitions
├── pkg/                 # Shared packages
│   ├── proxmox/         # Proxmox API client
│   ├── jwt/             # JWT utilities
│   └── log/             # Logging utilities
└── config/              # Configuration files
```

---

## Getting Started

### Prerequisites

- **Frontend**:
  - Node.js >= 20.19.0 or >= 22.13.0
  - pnpm >= 9

- **Backend**:
  - Go >= 1.23
  - MySQL / PostgreSQL / SQLite
  - Redis (optional, for caching)

- **Docker** (optional):
  - Docker >= 20.10
  - Docker Compose >= 2.0

### Frontend Setup

```bash
# Navigate to frontend directory
cd pvesphere-ui

# Install dependencies
pnpm install

# Start development server
pnpm dev

# Build for production
pnpm build
```

### Backend Setup

```bash
# Navigate to backend directory
cd pvesphere

# Install dependencies
go mod download

# Run database migration (will automatically create default user)
go run cmd/migration/main.go

# Start server
go run cmd/server/main.go
```

### Default User Information

After running the database migration for the first time, the system will automatically create a default administrator account. You can log in using the following credentials:

- **Email**: `pvesphere@gmail.com`
- **Password**: `Ab123456`
- **Nickname**: `PveSphere Admin`

> Note: If the default user already exists, the migration process will not create it again. It is recommended to change the password after the first login.

## Docker Deployment

### Quick Start (Recommended)

Use Makefile commands to quickly build and start all services:

```bash
# Build and start all services (including database migration)
make docker-compose-build

# Check service status
make docker-compose-ps

# View service logs
make docker-compose-logs

# Stop all services
make docker-compose-down
```

### Docker Image Building

#### Build Individual Service Images

```bash
# Build API service image
make docker-build-api

# Build controller service image
make docker-build-controller

# Build all service images
make docker-build
```

#### Manual Image Building

```bash
# Build API service
docker build -f deploy/build/Dockerfile \
  --build-arg APP_RELATIVE_PATH=./cmd/server \
  --build-arg APP_NAME=server \
  --build-arg APP_ENV=prod \
  -t pvesphere-api:latest .

# Build controller service
docker build -f deploy/build/Dockerfile \
  --build-arg APP_RELATIVE_PATH=./cmd/controller \
  --build-arg APP_NAME=controller \
  --build-arg APP_ENV=prod \
  -t pvesphere-controller:latest .
```

### Docker Compose Usage

The project uses Docker Compose to manage services, with SQLite as the default database.

#### Common Commands

```bash
# Start all services
make docker-compose-up

# Build and start (first run)
make docker-compose-build

# Check service status
make docker-compose-ps

# View all service logs
make docker-compose-logs

# View API service logs
make docker-compose-logs-api

# View controller service logs
make docker-compose-logs-controller

# Restart all services
make docker-compose-restart

# Stop services (keep containers)
make docker-compose-stop

# Start stopped services
make docker-compose-start

# Stop and remove all services
make docker-compose-down
```

#### Service Overview

- **api-server**: API service (port 8000)
- **controller**: Controller service
- **migration**: Database migration service (runs automatically)

#### Access Services

- **API Service**: http://localhost:8000
- **API Documentation**: http://localhost:8000/swagger/index.html

#### Default User Information

After running the database migration for the first time, the system will automatically create a default administrator account. You can log in using the following credentials:

- **Email**: `pvesphere@gmail.com`
- **Password**: `Ab123456`
- **Nickname**: `PveSphere Admin`

> Note: If the default user already exists, the migration process will not create it again. It is recommended to change the password after the first login.

#### Data Persistence

All data (database, logs) is stored in Docker volume `pvesphere-storage`, ensuring data persistence across container restarts.

### Local Development (Using Makefile)

The project provides convenient Makefile commands for local development:

```bash
# Initialize development environment (install tools)
make init

# Local startup (requires local Go environment)
# 1. Start dependency services (MySQL, Redis)
# 2. Run database migration
# 3. Start API service
make bootstrap

# Build local binaries
make build              # Build all services
make build-server       # Build API service only
make build-controller   # Build controller service only

# Run tests
make test

# Generate Swagger documentation
make swag
```

### Database Migration

#### Docker Environment

Database migration runs automatically when services start. To run manually:

```bash
# Run migration using docker compose
cd deploy/docker-compose
docker compose run --rm migration

# Or run in container
docker exec -it pvesphere-api ./migration -conf /data/app/config/docker.yml
```

#### Local Environment

```bash
# Using go run
go run ./cmd/migration -conf config/local.yml

# Or using nunu
nunu run ./cmd/migration -conf config/local.yml
```

### Push Images to Registry

```bash
# Push API service image
make docker-push-api REGISTRY=your-registry.com/pvesphere

# Push controller service image
make docker-push-controller REGISTRY=your-registry.com/pvesphere

# Push all service images
make docker-push REGISTRY=your-registry.com/pvesphere
```

For more Docker usage instructions, see [deploy/docker-compose/README.md](deploy/docker-compose/README.md)

---

## Main Features Overview

### 1. Dashboard

- Global overview of all clusters, nodes, VMs, and storage
- Resource utilization metrics (CPU, Memory, Storage)
- Hotspots and risk alerts
- Multi-cluster scope switching

### 2. Cluster Management

- Add and configure multiple PVE clusters
- API connection verification
- Cluster health monitoring
- Enable/disable cluster scheduling

### 3. Node Management

- View node status and resources
- Node console access (terminal proxy)
- Network configuration
- Service management (start, stop, restart)
- Disk and storage monitoring

### 4. Virtual Machine Management

- Create, start, stop, and delete VMs
- VM migration between nodes
- VM console access (VNC)
- Backup and restore
- Cloud-Init configuration
- Hardware configuration
- Network configuration

### 5. Storage Management

- View storage pools and usage
- Monitor storage capacity
- Browse storage content (ISO, backup, templates)

### 6. Template Management

- Import templates from backups
- Template synchronization across nodes
- Template instance management
- Support for shared and local storage

---

## API Documentation

Backend API documentation is available via Swagger UI when the server is running:

```
http://localhost:8000/swagger/index.html
```

---

## Development

### Frontend Development

```bash
# Development mode
pnpm dev

# Type checking
pnpm typecheck

# Linting
pnpm lint

# Build
pnpm build
```

### Backend Development

```bash
# Run server
go run cmd/server/main.go

# Run tests
go test ./...

# Generate Swagger docs
swag init
```

---

## Internationalization

The frontend supports multiple languages. Translation files are located in `locales/`:

- `zh-CN.yaml` - Simplified Chinese
- `en.yaml` - English

To add a new language, create a new YAML file in the `locales/` directory and update the i18n configuration.

---

## License

[MIT License](LICENSE)

Copyright © 2025-present PveSphere Contributors

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

---

## Related Projects

- [Proxmox VE](https://www.proxmox.com/) - The underlying virtualization platform
- [vue-pure-admin](https://github.com/pure-admin/vue-pure-admin) - Frontend base template
- [Nunu](https://github.com/go-nunu/nunu) - Backend framework

---

# 中文

## 简介

PveSphere 是一个基于 Web 的 Proxmox VE (PVE) 集群综合管理平台。它提供了一个现代化、直观的界面，用于从统一的仪表板管理多个 PVE 集群、节点、虚拟机、存储和模板。

### 什么是 PveSphere？

PveSphere 是一个多集群管理解决方案，支持集中控制和监控 Proxmox VE 基础设施。它通过为所有虚拟化资源提供单一管理界面，简化了管理分布式 PVE 环境的复杂性。

---

## 功能特性

### 🎯 核心功能

- **仪表板概览**：实时监控集群资源、健康状态和利用率指标
- **集群管理**：通过集中式身份验证和配置管理多个 PVE 集群
- **节点管理**：跨集群监控和管理物理节点，包括控制台访问
- **虚拟机管理**：完整的 VM 生命周期管理，包括创建、启动、停止、迁移、备份和恢复
- **存储管理**：监控存储使用情况、管理存储池和查看存储内容
- **模板管理**：导入、同步和管理 VM 模板，实现快速部署

### 🚀 主要能力

- 多集群支持，统一管理界面
- 实时资源监控和指标可视化
- 通过 VNC/NoVNC 访问 VM 控制台
- 通过终端代理访问节点控制台
- Cloud-Init 配置支持
- 自动化模板同步
- 备份和恢复功能
- 任务监控和管理
- 网络配置管理
- 服务管理（启动、停止、重启）
- 响应式设计，支持移动端
- 国际化（i18n）支持

---

## 技术栈

### 前端 (`pvesphere-ui`)

- **框架**：Vue 3 (Composition API)
- **构建工具**：Vite 7
- **UI 库**：Element Plus 2
- **语言**：TypeScript 5
- **状态管理**：Pinia 3
- **路由**：Vue Router 4
- **国际化**：Vue I18n
- **图表**：ECharts 6
- **终端**：xterm.js 5, noVNC
- **样式**：Tailwind CSS 4, SCSS
- **基础模板**：[vue-pure-admin](https://github.com/pure-admin/vue-pure-admin)

### 后端 (`pvesphere`)

- **语言**：Go 1.23
- **Web 框架**：Gin 1.10
- **ORM**：GORM 1.30
- **数据库**：MySQL / PostgreSQL / SQLite
- **身份验证**：JWT (golang-jwt/jwt)
- **缓存**：Redis 9
- **WebSocket**：Gorilla WebSocket
- **任务调度**：gocron
- **日志**：Zap
- **API 文档**：Swagger
- **架构**：基于 [Nunu](https://github.com/go-nunu/nunu) 框架

---

## 项目结构

### 前端结构

```
pvesphere-ui/
├── src/
│   ├── api/              # API 接口
│   ├── assets/          # 静态资源
│   ├── components/      # 可复用组件
│   ├── config/          # 配置文件
│   ├── directives/      # Vue 指令
│   ├── layout/          # 布局组件
│   ├── plugins/         # 插件配置
│   ├── router/          # 路由配置
│   ├── store/           # Pinia 状态管理
│   ├── style/           # 全局样式
│   ├── utils/           # 工具函数
│   └── views/           # 页面组件
│       └── pve/         # PVE 管理页面
│           ├── cluster/     # 集群管理
│           ├── dashboard/   # 仪表板
│           ├── node/        # 节点管理
│           ├── storage/     # 存储管理
│           ├── template/   # 模板管理
│           └── vm/         # 虚拟机管理
├── locales/             # 国际化翻译文件
└── package.json
```

### 后端结构

```
pvesphere/
├── api/v1/              # API 路由处理器
├── cmd/                 # 应用入口点
│   ├── server/          # HTTP 服务器
│   ├── controller/      # Kubernetes 控制器
│   ├── migration/       # 数据库迁移
│   └── task/            # 后台任务
├── internal/
│   ├── handler/         # 请求处理器
│   ├── service/         # 业务逻辑
│   ├── repository/      # 数据访问层
│   ├── model/           # 数据模型
│   ├── middleware/      # HTTP 中间件
│   └── router/          # 路由定义
├── pkg/                 # 共享包
│   ├── proxmox/         # Proxmox API 客户端
│   ├── jwt/             # JWT 工具
│   └── log/             # 日志工具
└── config/              # 配置文件
```

---

## 快速开始

### 前置要求

- **前端**：
  - Node.js >= 20.19.0 或 >= 22.13.0
  - pnpm >= 9

- **后端**：
  - Go >= 1.23
  - MySQL / PostgreSQL / SQLite
  - Redis（可选，用于缓存）

### 前端设置

```bash
# 进入前端目录
cd pvesphere-ui

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev

# 构建生产版本
pnpm build
```

### 后端设置

```bash
# 进入后端目录
cd pvesphere

# 安装依赖
go mod download

# 运行数据库迁移
go run cmd/migration/main.go

# 启动服务器
go run cmd/server/main.go
```

---

## 主要功能概览

### 1. 仪表板

- 所有集群、节点、虚拟机和存储的全局概览
- 资源利用率指标（CPU、内存、存储）
- 热点和风险告警
- 多集群范围切换

### 2. 集群管理

- 添加和配置多个 PVE 集群
- API 连接验证
- 集群健康监控
- 启用/禁用集群调度

### 3. 节点管理

- 查看节点状态和资源
- 节点控制台访问（终端代理）
- 网络配置
- 服务管理（启动、停止、重启）
- 磁盘和存储监控

### 4. 虚拟机管理

- 创建、启动、停止和删除虚拟机
- 虚拟机在节点间迁移
- 虚拟机控制台访问（VNC）
- 备份和恢复
- Cloud-Init 配置
- 硬件配置
- 网络配置

### 5. 存储管理

- 查看存储池和使用情况
- 监控存储容量
- 浏览存储内容（ISO、备份、模板）

### 6. 模板管理

- 从备份导入模板
- 跨节点模板同步
- 模板实例管理
- 支持共享和本地存储

---

## API 文档

后端 API 文档在服务器运行时可通过 Swagger UI 访问：

```
http://localhost:8000/swagger/index.html
```

---

## 开发

### 前端开发

```bash
# 开发模式
pnpm dev

# 类型检查
pnpm typecheck

# 代码检查
pnpm lint

# 构建
pnpm build
```

### 后端开发

```bash
# 运行服务器
go run cmd/server/main.go

# 运行测试
go test ./...

# 生成 Swagger 文档
swag init
```

---

## 国际化

前端支持多种语言。翻译文件位于 `locales/`：

- `zh-CN.yaml` - 简体中文
- `en.yaml` - 英文

要添加新语言，请在 `locales/` 目录中创建新的 YAML 文件并更新 i18n 配置。

---

## 许可证

[MIT License](LICENSE)

版权所有 © 2025-present PveSphere Contributors

---

## 贡献

欢迎贡献！请随时提交 Pull Request。

---

## 相关项目

- [Proxmox VE](https://www.proxmox.com/) - 底层虚拟化平台
- [vue-pure-admin](https://github.com/pure-admin/vue-pure-admin) - 前端基础模板
- [Nunu](https://github.com/go-nunu/nunu) - 后端框架
