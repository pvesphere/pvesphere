# 模板管理系统使用指南

## 📚 目录

1. [系统概述](#系统概述)
2. [快速开始](#快速开始)
3. [API 使用说明](#api-使用说明)
4. [典型场景](#典型场景)
5. [注意事项](#注意事项)

## 系统概述

### 核心功能

模板管理系统提供了完整的 PVE 虚拟机模板管理能力：

- ✅ **模板上传**：支持上传模板文件到 PVE 存储
- ✅ **自动导入**：自动将上传的文件转换为 PVE 模板
- ✅ **智能同步**：根据存储类型自动选择同步策略
- ✅ **实例管理**：追踪模板在各个节点上的状态
- ✅ **任务追踪**：实时查看同步任务进度

### 核心概念

#### 1. Template（模板）
逻辑上的模板定义，存储在 `vm_template` 表中。

#### 2. TemplateUpload（上传记录）
记录模板文件的上传和导入过程。

#### 3. TemplateInstance（模板实例）
模板在特定节点上的实例状态。
- **共享存储**：逻辑实例（文件共享，无需同步）
- **本地存储**：物理实例（需要文件同步）

#### 4. TemplateSyncTask（同步任务）
用于追踪本地存储模板的跨节点同步进度。

### 存储类型

系统根据存储类型采用不同的处理策略：

| 存储类型 | shared 值 | 示例 | 同步策略 |
|---------|-----------|------|---------|
| 共享存储 | 1 | Ceph, NFS, iSCSI | 无需同步，所有节点自动可见 |
| 本地存储 | 0 | local, dir | 需要手动/自动同步到其他节点 |

## 快速开始

### 1. 数据库迁移

首先执行数据库迁移脚本创建必要的表：

```bash
# 方式1：使用 nunu migration
nunu run cmd/migration/main.go

# 方式2：手动执行 SQL 脚本
mysql -u root -p pvesphere < scripts/migration_template_management.sql
```

迁移完成后，将创建以下三张表：
- `template_upload`
- `template_instance`
- `template_sync_task`

### 2. 启动服务

```bash
# 启动 API 服务
nunu run cmd/server/main.go

# 服务默认运行在 http://localhost:8080
```

### 3. 上传第一个模板

```bash
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "template_name=centos7-template" \
  -F "cluster_id=1" \
  -F "storage_id=10" \
  -F "description=CentOS 7 基础模板" \
  -F "file=@/path/to/centos7.qcow2"
```

## API 使用说明

### 1. 上传并导入模板

**接口**: `POST /api/v1/templates/upload`

**参数**:
- `template_name` (必填): 模板名称
- `cluster_id` (必填): 集群 ID
- `storage_id` (必填): 存储 ID
- `description` (可选): 模板描述
- `auto_sync` (可选): 是否自动同步（仅 local 存储）
- `sync_node_ids` (可选): 同步节点 ID 列表，逗号分隔
- `file` (必填): 模板文件

**示例 - 共享存储**:

```bash
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "template_name=centos7-ceph" \
  -F "cluster_id=1" \
  -F "storage_id=10" \
  -F "description=CentOS 7 on Ceph" \
  -F "file=@centos7.qcow2"
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "template_id": 1,
    "upload_id": 1,
    "storage_type": "cephfs",
    "is_shared": true,
    "import_node": {
      "node_id": 1,
      "node_name": "pve-node1"
    },
    "sync_tasks": []
  }
}
```

**示例 - 本地存储（指定同步节点）**:

```bash
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "template_name=centos7-local" \
  -F "cluster_id=1" \
  -F "storage_id=20" \
  -F "description=CentOS 7 on local" \
  -F "sync_node_ids=2,3" \
  -F "file=@centos7.qcow2"
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "template_id": 2,
    "upload_id": 2,
    "storage_type": "dir",
    "is_shared": false,
    "import_node": {
      "node_id": 1,
      "node_name": "pve-node1"
    },
    "sync_tasks": [
      {
        "task_id": 1,
        "target_node_id": 2,
        "target_node_name": "pve-node2",
        "status": "pending"
      },
      {
        "task_id": 2,
        "target_node_id": 3,
        "target_node_name": "pve-node3",
        "status": "pending"
      }
    ]
  }
}
```

### 2. 查询模板详情

**接口**: `GET /api/v1/templates/{id}/detail`

**参数**:
- `id` (路径参数): 模板 ID
- `include_instances` (查询参数): 是否包含实例信息（true/false）

**示例**:

```bash
curl -X GET "http://localhost:8080/api/v1/templates/1/detail?include_instances=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "template_name": "centos7-template",
    "cluster_id": 1,
    "cluster_name": "prod-cluster",
    "description": "CentOS 7 基础模板",
    "upload_info": {
      "upload_id": 1,
      "storage_name": "ceph-storage",
      "is_shared": true,
      "file_name": "centos7.qcow2",
      "file_size": 1073741824,
      "status": "imported"
    },
    "instances": [
      {
        "instance_id": 1,
        "node_id": 1,
        "node_name": "pve-node1",
        "vmid": 9000,
        "storage_name": "ceph-storage",
        "status": "available",
        "is_primary": true
      },
      {
        "instance_id": 2,
        "node_id": 2,
        "node_name": "pve-node2",
        "vmid": 9000,
        "storage_name": "ceph-storage",
        "status": "available",
        "is_primary": false
      }
    ]
  }
}
```

### 3. 手动同步模板

**接口**: `POST /api/v1/templates/{id}/sync`

**参数**:
- `id` (路径参数): 模板 ID
- `target_node_ids` (请求体): 目标节点 ID 数组

**示例**:

```bash
curl -X POST http://localhost:8080/api/v1/templates/2/sync \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_node_ids": [4, 5]
  }'
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "sync_tasks": [
      {
        "task_id": 3,
        "target_node_id": 4,
        "target_node_name": "pve-node4",
        "status": "pending"
      },
      {
        "task_id": 4,
        "target_node_id": 5,
        "target_node_name": "pve-node5",
        "status": "pending"
      }
    ]
  }
}
```

### 4. 查询同步任务

**接口**: `GET /api/v1/templates/sync-tasks/{task_id}`

**示例**:

```bash
curl -X GET http://localhost:8080/api/v1/templates/sync-tasks/1 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "task_id": 1,
    "template_id": 2,
    "template_name": "centos7-local",
    "source_node": {
      "node_id": 1,
      "node_name": "pve-node1"
    },
    "target_node": {
      "node_id": 2,
      "node_name": "pve-node2"
    },
    "storage_name": "local",
    "status": "syncing",
    "progress": 45,
    "sync_start_time": "2025-12-24T10:00:00Z",
    "error_message": null
  }
}
```

### 5. 列出同步任务

**接口**: `GET /api/v1/templates/sync-tasks`

**参数**:
- `page` (可选): 页码，默认 1
- `page_size` (可选): 每页数量，默认 10
- `template_id` (可选): 按模板 ID 过滤
- `status` (可选): 按状态过滤（pending/syncing/completed/failed）

**示例**:

```bash
curl -X GET "http://localhost:8080/api/v1/templates/sync-tasks?template_id=2&status=syncing&page=1&page_size=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 6. 重试同步任务

**接口**: `POST /api/v1/templates/sync-tasks/{task_id}/retry`

**示例**:

```bash
curl -X POST http://localhost:8080/api/v1/templates/sync-tasks/1/retry \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 7. 列出模板实例

**接口**: `GET /api/v1/templates/{id}/instances`

**示例**:

```bash
curl -X GET http://localhost:8080/api/v1/templates/1/instances \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 典型场景

### 场景 1：使用共享存储部署模板

**背景**：
- 集群使用 Ceph 共享存储
- 需要在所有节点上使用同一个模板

**步骤**：

1. 上传模板到 Ceph 存储

```bash
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -F "template_name=ubuntu-22.04" \
  -F "cluster_id=1" \
  -F "storage_id=15" \
  -F "description=Ubuntu 22.04 LTS" \
  -F "file=@ubuntu-22.04-cloud.qcow2"
```

2. 系统自动完成：
   - 选择任意节点上传文件
   - 在该节点导入模板
   - 为所有节点创建模板实例（状态：available）

3. 验证所有节点都可用

```bash
curl -X GET "http://localhost:8080/api/v1/templates/1/instances"
```

4. 在任意节点使用模板创建虚拟机

**优点**：
- ✅ 一次上传，所有节点可用
- ✅ 无需同步，立即可用
- ✅ 存储空间利用率高

### 场景 2：使用本地存储部署模板

**背景**：
- 集群使用本地存储（性能更好）
- 需要在多个节点上使用模板
- 需要手动控制同步范围

**步骤**：

1. 上传模板到本地存储（仅在节点1）

```bash
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -F "template_name=debian-12" \
  -F "cluster_id=1" \
  -F "storage_id=25" \
  -F "description=Debian 12" \
  -F "file=@debian-12.qcow2"
```

2. 系统完成：
   - 在节点1上传并导入模板
   - 为节点1创建实例（状态：available）

3. 手动同步到节点 2 和 3

```bash
curl -X POST http://localhost:8080/api/v1/templates/1/sync \
  -H "Content-Type: application/json" \
  -d '{"target_node_ids": [2, 3]}'
```

4. 查询同步进度

```bash
curl -X GET "http://localhost:8080/api/v1/templates/sync-tasks?template_id=1"
```

5. 等待同步完成后，在各节点使用模板

**优点**：
- ✅ 灵活控制同步范围
- ✅ 本地存储性能更好
- ✅ 支持按需同步

### 场景 3：批量部署模板

**背景**：
- 需要部署多个不同的模板
- 希望自动同步到所有节点

**步骤**：

1. 批量上传模板（使用自动同步）

```bash
# 上传 CentOS 7
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -F "template_name=centos7" \
  -F "cluster_id=1" \
  -F "storage_id=25" \
  -F "sync_node_ids=2,3,4,5" \
  -F "file=@centos7.qcow2"

# 上传 Ubuntu 22.04
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -F "template_name=ubuntu2204" \
  -F "cluster_id=1" \
  -F "storage_id=25" \
  -F "sync_node_ids=2,3,4,5" \
  -F "file=@ubuntu2204.qcow2"

# 上传 Debian 12
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -F "template_name=debian12" \
  -F "cluster_id=1" \
  -F "storage_id=25" \
  -F "sync_node_ids=2,3,4,5" \
  -F "file=@debian12.qcow2"
```

2. 系统自动创建所有同步任务

3. 监控所有同步任务

```bash
curl -X GET "http://localhost:8080/api/v1/templates/sync-tasks?status=syncing"
```

4. 同步完成后，所有节点都可使用这些模板

## 注意事项

### 1. 存储类型判断

系统根据 `pve_storage.shared` 字段判断存储类型：
- `shared = 1`：共享存储，不需要同步
- `shared = 0`：本地存储，需要同步

**重要**：确保 PVE 存储配置正确，否则会导致判断错误。

### 2. VMID 管理

- **共享存储**：所有节点的实例使用相同的 VMID
- **本地存储**：每个节点的实例使用独立的 VMID（可以相同，因为存储隔离）

### 3. 同步失败处理

如果同步失败：

1. 查看错误信息

```bash
curl -X GET http://localhost:8080/api/v1/templates/sync-tasks/{task_id}
```

2. 修复问题（网络、存储空间等）

3. 重试任务

```bash
curl -X POST http://localhost:8080/api/v1/templates/sync-tasks/{task_id}/retry
```

### 4. 性能优化建议

- **文件格式**：推荐使用 qcow2 格式（支持压缩和增量）
- **文件大小**：建议单个模板不超过 50GB
- **并发同步**：避免同时同步大量模板到同一节点
- **网络带宽**：同步时会占用网络带宽，建议在业务低峰期进行

### 5. 安全建议

- ✅ 上传接口需要严格的权限控制
- ✅ 验证上传文件的格式和大小
- ✅ 限制上传速率，防止 DoS
- ✅ 定期清理失败的上传和同步任务

### 6. 常见问题

**Q: 共享存储的模板是否需要同步？**

A: 不需要。共享存储的特点是所有节点都可以访问同一份文件，因此上传一次后，所有节点自动可用。

**Q: 如何删除模板？**

A: 目前支持软删除（标记为已删除）和级联删除（删除模板和所有实例）。使用 `DELETE /api/v1/templates/{id}?cascade=true` 进行级联删除。

**Q: 同步任务可以取消吗？**

A: 当前版本不支持取消进行中的任务，但可以删除 pending 状态的任务。

**Q: 模板文件存储在哪里？**

A: 取决于选择的存储类型：
- Ceph: 存储在 Ceph 池中
- NFS: 存储在 NFS 共享目录
- Local: 存储在节点本地目录（通常是 `/var/lib/vz/template/`）

**Q: 如何监控同步进度？**

A: 通过轮询 `/api/v1/templates/sync-tasks/{task_id}` 接口，查看 `progress` 字段（0-100）。

**Q: 同步速度慢怎么办？**

A: 可能的原因和解决方案：
1. 网络带宽不足 → 升级网络
2. 磁盘 I/O 慢 → 使用 SSD
3. 文件过大 → 压缩模板文件
4. 多任务并发 → 限制并发数量

## 下一步

- [ ] 实现异步同步任务执行器
- [ ] 添加 WebSocket 支持实时进度推送
- [ ] 支持断点续传
- [ ] 添加模板版本管理
- [ ] 支持模板克隆和快照

## 相关文档

- [设计文档](./template-management-design.md)
- [API 文档](./swagger.yaml)
- [数据库迁移脚本](../scripts/migration_template_management.sql)

