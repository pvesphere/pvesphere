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

## 配置说明

### 修改端口

编辑 `docker-compose.yml`:

```yaml
services:
  api:
    ports:
      - "8001:8000"  # 修改主机端口
  frontend:
    ports:
      - "8081:8080"  # 修改主机端口
```

### 前端 API 配置

前端通过 Nginx 反向代理访问后端 API：
- 前端请求: `/api/v1/*`
- Nginx 代理到: `http://api:8000/api/v1/*`

前端代码中的 API 地址配置为相对路径（空字符串），通过 Nginx 代理。

## 故障排查

### 查看服务日志

```bash
docker-compose logs api
docker-compose logs controller
docker-compose logs frontend
```

### 端口被占用

```bash
# 查看端口占用
lsof -i :8000
lsof -i :8080
```

### 前端构建失败

1. **检查前端目录是否存在**：
```bash
ls -la /path/to/pvesphere-ui
```

2. **检查 package.json**：
```bash
cat /path/to/pvesphere-ui/package.json
```

3. **清理缓存后重新构建**：
```bash
./deploy/build.sh -c
./deploy/build.sh -f /path/to/pvesphere-ui
```

4. **查看详细构建日志**：
```bash
docker-compose build --progress=plain frontend
```

### 前端无法调用后端

1. 检查浏览器控制台（F12）是否有错误
2. 确认 API 服务正常运行：`curl http://localhost:8000/`
3. 清除浏览器缓存并刷新页面（Ctrl+Shift+R 或 Cmd+Shift+R）
4. 检查 Nginx 代理配置：`docker-compose exec frontend cat /etc/nginx/conf.d/default.conf`

### 重新构建

```bash
# 清理并重新构建前端
cd deploy
./build.sh -c
./build.sh

# 重新构建所有服务
cd docker-compose
docker-compose build --no-cache
docker-compose up -d
```

## 技术栈

- 后端: Go 1.23 + Nunu 框架
- 前端: Vue.js + Vite（使用 pnpm 构建，仅支持本地代码）
- 数据库: SQLite
- Web 服务器: Nginx（反向代理）

## 注意事项

1. **前端构建仅支持本地代码**：必须提供本地前端代码目录
2. **前端使用 pnpm 包管理器**：项目要求使用 pnpm
3. **所有服务运行在 `pvesphere-network` 桥接网络中**
4. **数据持久化在 `pvesphere-storage` volume 中**
5. **构建脚本会自动排除**：`node_modules`、`dist`、`.git` 等目录

## 目录结构

```
deploy/
├── build/
│   ├── Dockerfile              # 构建文件
│   └── .frontend-local/        # 前端代码缓存（自动生成）
├── build.sh                    # 构建脚本
├── docker-compose/
│   └── docker-compose.yml      # Docker Compose 配置
├── nginx/
│   └── nginx.conf              # Nginx 配置
└── README.md                   # 本文档
```

## 优势

- ✅ **无网络依赖**：前端构建不需要从 Git 下载
- ✅ **构建速度快**：使用本地代码，避免网络延迟
- ✅ **灵活配置**：支持指定任意前端目录
- ✅ **节省空间**：支持符号链接模式
- ✅ **易于调试**：本地代码便于修改和测试
