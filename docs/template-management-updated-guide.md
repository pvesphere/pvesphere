# 模板管理系统使用指南（更新版）

## 🎯 功能概述

PVEsphere 模板管理系统基于**已有虚拟机备份文件**创建和管理模板，无需手动上传文件。

### 核心特性

- ✅ **基于备份导入**：使用 PVE 已有的虚拟机备份文件作为模板源
- ✅ **智能同步**：根据存储类型自动选择同步策略
- ✅ **实例追踪**：实时追踪模板在各个节点上的状态
- ✅ **任务管理**：查看和管理同步任务，支持重试失败任务

## 📋 业务流程

### 原流程（已废弃）
```
用户上传文件 → 存储到节点 → 导入为模板 → 创建实例/同步
```

### 新流程
```
选择已有备份文件 → 指定导入节点 → 导入为模板 → 创建实例/同步
```

## 🚀 快速开始

### 1. 准备备份文件

首先确保你有可用的虚拟机备份文件，PVE 备份文件通常位于：

- **本地存储**：`/var/lib/vz/dump/`
- **共享存储**：`/mnt/pve/{storage_name}/dump/`

备份文件格式示例：
```
vzdump-qemu-100-2024_01_01-00_00_00.vma
vzdump-qemu-100-2024_01_01-00_00_00.vma.zst
vzdump-qemu-100-2024_01_01-00_00_00.vma.lzo
vzdump-qemu-100-2024_01_01-00_00_00.vma.gz
```

### 2. 导入模板（共享存储）

```bash
curl -X POST http://localhost:8080/api/v1/templates/import \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "centos7-template",
    "cluster_id": 1,
    "storage_id": 10,
    "node_id": 1,
    "backup_file": "vzdump-qemu-100-2024_01_01-00_00_00.vma.zst",
    "description": "CentOS 7 基础模板"
  }'
```

**响应**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "template_id": 1,
    "import_id": 1,
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

### 3. 导入模板（本地存储 + 同步）

```bash
curl -X POST http://localhost:8080/api/v1/templates/import \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "ubuntu2204-template",
    "cluster_id": 1,
    "storage_id": 20,
    "node_id": 1,
    "backup_file": "vzdump-qemu-200-2024_01_15-10_30_00.vma.zst",
    "description": "Ubuntu 22.04 LTS",
    "sync_node_ids": [2, 3]
  }'
```

**响应**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "template_id": 2,
    "import_id": 2,
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

## 📖 API 接口文档

### 1. 导入模板

**接口**: `POST /api/v1/templates/import`

**请求参数**:
```json
{
  "template_name": "centos7-template",    // 模板名称（必填）
  "cluster_id": 1,                        // 集群ID（必填）
  "storage_id": 10,                       // 存储ID（必填）
  "node_id": 1,                           // 导入节点ID（必填）
  "backup_file": "vzdump-qemu-100-2024_01_01-00_00_00.vma.zst",  // 备份文件名（必填）
  "description": "CentOS 7 基础模板",     // 描述（可选）
  "auto_sync": false,                     // 是否自动同步到所有节点（可选，仅 local 存储）
  "sync_node_ids": [2, 3]                 // 同步到的节点ID列表（可选，仅 local 存储）
}
```

**响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "template_id": 1,
    "import_id": 1,
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

### 2. 查询模板详情

**接口**: `GET /api/v1/templates/{id}/detail?include_instances=true`

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
      "file_name": "vzdump-qemu-100-2024_01_01-00_00_00.vma.zst",
      "file_size": 1073741824,
      "status": "imported"
    },
    "instances": [
      {
        "instance_id": 1,
        "node_id": 1,
        "node_name": "pve-node1",
        "vmid": 9000,
        "status": "available",
        "is_primary": true
      },
      {
        "instance_id": 2,
        "node_id": 2,
        "node_name": "pve-node2",
        "vmid": 9000,
        "status": "available",
        "is_primary": false
      }
    ]
  }
}
```

### 3. 手动同步模板

**接口**: `POST /api/v1/templates/{id}/sync`

**请求**:
```json
{
  "target_node_ids": [4, 5]
}
```

### 其他接口

其他接口（查询同步任务、列出实例等）保持不变，参考原文档。

## 🎯 典型使用场景

### 场景 1：基于共享存储备份创建模板

```bash
# 1. 列出备份文件（假设已有）
# ls /mnt/pve/ceph-storage/dump/
# vzdump-qemu-100-2024_01_01-00_00_00.vma.zst
# vzdump-qemu-200-2024_01_15-10_30_00.vma.zst

# 2. 导入备份为模板
curl -X POST http://localhost:8080/api/v1/templates/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "centos7-prod",
    "cluster_id": 1,
    "storage_id": 10,
    "node_id": 1,
    "backup_file": "vzdump-qemu-100-2024_01_01-00_00_00.vma.zst",
    "description": "生产环境 CentOS 7 模板"
  }'

# 3. 系统自动完成：
#    - 在节点1上从备份恢复虚拟机
#    - 转换为模板
#    - 为所有节点创建逻辑实例（状态：available）

# 4. 所有节点立即可用该模板创建虚拟机
```

### 场景 2：基于本地存储备份创建模板并同步

```bash
# 1. 确认备份文件存在
# ssh pve-node1 "ls /var/lib/vz/dump/"
# vzdump-qemu-300-2024_02_01-15_00_00.vma.zst

# 2. 导入备份为模板，并指定同步节点
curl -X POST http://localhost:8080/api/v1/templates/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "ubuntu2204-dev",
    "cluster_id": 1,
    "storage_id": 20,
    "node_id": 1,
    "backup_file": "vzdump-qemu-300-2024_02_01-15_00_00.vma.zst",
    "description": "开发环境 Ubuntu 22.04",
    "sync_node_ids": [2, 3, 4]
  }'

# 3. 系统执行：
#    - 在节点1上从备份恢复并转换为模板
#    - 创建节点1的实例（available）
#    - 创建节点2、3、4的同步任务（pending）

# 4. 查询同步进度
curl -X GET "http://localhost:8080/api/v1/templates/sync-tasks?template_id=2" \
  -H "Authorization: Bearer $TOKEN"
```

### 场景 3：查找可用的备份文件

由于系统需要知道备份文件名，你可以：

**方法1：通过 PVE Web UI 查看**
1. 登录 PVE Web UI
2. Datacenter → Storage → 选择存储 → Content
3. 查看 backup 类型的文件列表

**方法2：通过 PVE API 查询**
```bash
# 查询存储上的备份文件列表
curl -X GET "https://pve-node1:8006/api2/json/nodes/pve-node1/storage/local/content?content=backup" \
  -H "Authorization: PVEAPIToken=USER@REALM!TOKENID=UUID"
```

**方法3：通过 SSH 直接查看**
```bash
# 本地存储
ssh pve-node1 "ls -lh /var/lib/vz/dump/"

# 共享存储（以 Ceph 为例）
ssh pve-node1 "ls -lh /mnt/pve/ceph-storage/dump/"
```

## ⚙️ 技术实现细节

### 备份文件路径构建

系统会根据存储类型自动构建备份文件路径：

```go
// 本地存储
// 输入：backup_file = "vzdump-qemu-100-2024_01_01.vma.zst"
// 输出：local:backup/vzdump-qemu-100-2024_01_01.vma.zst

// 共享存储
// 输入：backup_file = "vzdump-qemu-100-2024_01_01.vma.zst"
// 输出：ceph-storage:backup/vzdump-qemu-100-2024_01_01.vma.zst
```

### 导入流程（PVE 命令）

系统在后台执行以下 PVE 命令：

```bash
# 1. 从备份恢复虚拟机
qmrestore /var/lib/vz/dump/vzdump-qemu-100-2024_01_01.vma.zst <new_vmid> \
  --storage <storage_name>

# 2. 重命名虚拟机（可选）
qm set <new_vmid> --name <template_name>

# 3. 转换为模板
qm template <new_vmid>
```

### 支持的备份格式

- `.vma` - 未压缩的 PVE 备份
- `.vma.zst` - Zstandard 压缩（推荐，速度快）
- `.vma.lzo` - LZO 压缩
- `.vma.gz` - Gzip 压缩

## 💡 最佳实践

### 1. 选择合适的备份文件

- ✅ 使用**最小化安装**的虚拟机备份（减少文件大小）
- ✅ 移除不必要的软件包和日志
- ✅ 清理临时文件和缓存
- ✅ 使用压缩格式（推荐 `.vma.zst`）

### 2. 备份文件命名规范

建议保持 PVE 的默认命名格式：
```
vzdump-qemu-{vmid}-{date}-{time}.{format}
```

示例：
```
vzdump-qemu-100-2024_01_01-00_00_00.vma.zst
```

### 3. 存储选择策略

**共享存储**：
- ✅ 适合需要在多个节点使用的模板
- ✅ 节省存储空间（只存一份）
- ✅ 立即可用，无需同步

**本地存储**：
- ✅ 适合高性能要求的场景
- ✅ 需要手动同步到其他节点
- ⚠️ 占用每个节点的本地存储空间

### 4. 同步策略

- 对于**生产环境**模板，建议同步到所有节点
- 对于**开发/测试**模板，按需同步
- 可以先导入，稍后再手动同步（避免并发压力）

## ⚠️ 注意事项

### 1. 备份文件必须存在

导入前请确保备份文件已经存在于指定的存储上。系统**不会自动上传**文件。

### 2. VMID 分配

系统会自动分配新的 VMID，不会与原始虚拟机冲突。

### 3. 节点访问权限

确保导入节点可以访问指定的存储。

### 4. 存储空间

导入过程会占用存储空间，请确保有足够的可用空间。

### 5. 网络带宽

同步到其他节点会占用网络带宽，建议在业务低峰期进行。

## 🆚 与原方案的对比

| 特性 | 原方案（上传） | 新方案（导入） |
|------|---------------|---------------|
| 文件来源 | 用户上传 | PVE 已有备份 |
| 网络传输 | 需要（用户→服务器） | 不需要 |
| 适用场景 | 外部导入 | 基于现有虚拟机 |
| 实现复杂度 | 高（需要处理文件上传） | 低（直接调用 PVE API） |
| 性能 | 受网络影响 | 本地操作，速度快 |

## 🔗 相关文档

- [设计文档](./template-management-design.md)
- [实现总结](./template-management-implementation-summary.md)
- [Proxmox VE Backup API](https://pve.proxmox.com/pve-docs/api-viewer/index.html#/nodes/{node}/storage/{storage}/content)

## 📝 更新日志

**2025-12-25**
- ✅ 移除文件上传功能
- ✅ 改为基于已有备份文件导入
- ✅ 简化 API 接口
- ✅ 更新所有文档和示例

---

**提示**：如果你需要从外部导入模板文件，请先通过 PVE Web UI 或 SSH 将文件上传到 PVE 存储的 backup 目录，然后使用本系统的导入功能。

