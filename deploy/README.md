# PVESphere Docker 部署指南

## 快速启动

### 1. 准备前端代码

前端构建**仅支持本地代码**，需要先准备前端代码：

```bash
cd deploy
./build.sh -f /path/to/pvesphere-ui
```

或使用默认路径：

```bash
cd deploy
./build.sh
```

### 2. 启动所有服务

```bash
cd docker-compose
docker-compose up -d
```

## 访问地址

- 前端: http://localhost:8080
- API: http://localhost:8000
- API 文档: http://localhost:8000/swagger/index.html

## 服务说明

| 服务 | 端口 | 说明 |
|------|------|------|
| api | 8000 | API 服务，启动时自动执行数据库迁移 |
| controller | - | 控制器服务，负责 PVE 资源同步 |
| frontend | 8080 | 前端服务，使用本地代码构建 |

## 构建说明

### 前端构建

前端构建**仅支持本地代码**，不支持从 Git 下载：

```bash
# 使用默认路径
./deploy/build.sh

# 指定前端目录
./deploy/build.sh -f /path/to/pvesphere-ui

# 使用符号链接（节省空间）
./deploy/build.sh -l

# 清理缓存
./deploy/build.sh -c
```

**构建流程**：
1. 复制前端代码到 `deploy/build/.frontend-local/`
2. Docker 构建时从该目录复制代码
3. 使用 pnpm 安装依赖并构建
4. 生成前端镜像

### 后端构建

后端构建会自动进行，无需额外操作：

```bash
docker-compose build api controller
```

## 常用命令

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 查看状态
docker-compose ps

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 重新构建前端
cd ../..
./deploy/build.sh
cd deploy/docker-compose
docker-compose up -d --build frontend
```

## 生产环境部署说明

### 目录布局

推荐在生产环境使用类似以下目录结构（以 `/opt/pvesphere` 为例）：

```bash
/opt/pvesphere/
├── docker-compose.yml          # 生产环境 docker-compose 配置（从 deploy/docker-compose/docker-compose.prod.yml 复制）
├── config/                     # 生产配置目录
│   ├── api/                    # API 服务配置
│   │   └── docker.yml          # API 使用的配置文件
│   └── controller/             # Controller 服务配置
│       └── docker.yml          # Controller 使用的配置文件
└── storage/                    # 数据与日志目录（自动创建）
    ├── pvesphere-test.db       # SQLite 数据库文件（运行后自动生成）
    └── logs/
        ├── server.log          # API 服务日志
        ├── controller.log      # Controller 服务日志
        ├── access.log          # Nginx 访问日志
        └── error.log           # Nginx 错误日志
```

### 生产环境部署步骤

#### 1. 准备镜像

你可以选择以下任意一种方式准备镜像：

- **在开发环境构建**：

  ```bash
  cd deploy/docker-compose
  docker-compose build
  ```

  然后将构建好的镜像推送到私有镜像仓库，或在生产机上直接 `docker save` / `docker load`；

- **从 Docker Hub 下载**：如果镜像已发布到公共或私有仓库，直接使用 `docker pull` 拉取。

#### 2. 在目标服务器上创建目录并复制配置文件

在生产服务器上（例如 `/opt/pvesphere`）创建目录结构，并复制必要的配置文件：

```bash
# 在生产服务器上
mkdir -p /opt/pvesphere/config/api
mkdir -p /opt/pvesphere/config/controller
mkdir -p /opt/pvesphere/storage/logs

# 复制生产 docker-compose 配置文件
# 从项目仓库复制 deploy/docker-compose/docker-compose.prod.yml 到 /opt/pvesphere/docker-compose.yml
cp deploy/docker-compose/docker-compose.prod.yml /opt/pvesphere/docker-compose.yml

# 复制服务配置文件（先复制两份，再分别按需修改）
cp config/docker.yml /opt/pvesphere/config/api/docker.yml
cp config/docker.yml /opt/pvesphere/config/controller/docker.yml
```

#### 3. 修改生产配置

根据生产环境需要，分别编辑：

- `/opt/pvesphere/config/api/docker.yml`：API 服务配置（如端口、数据库、日志级别等）
- `/opt/pvesphere/config/controller/docker.yml`：Controller 服务配置（可以使用不同的日志文件名，如 `./storage/logs/controller.log`）

#### 4. 启动服务

```bash
cd /opt/pvesphere
docker-compose -f docker-compose.yml up -d
```

#### 5. 查看运行状态与日志

```bash
cd /opt/pvesphere
docker-compose ps
docker-compose logs -f api
docker-compose logs -f controller
docker-compose logs -f frontend
```

### 数据备份

所有数据（SQLite 数据库、日志）都位于 `storage` 目录下，备份时只需打包该目录：

```bash
cd /opt/pvesphere
tar czf pvesphere-backup-$(date +%F).tar.gz storage
```