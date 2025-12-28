# 模板管理系统设计文档

## 一、核心概念

### 1.1 核心实体

- **Template**：模板的逻辑定义，存储在 `vm_template` 表中
- **TemplateUpload**：模板上传记录，记录文件上传和导入过程
- **TemplateInstance**：模板在特定节点上的实例状态
- **TemplateSyncTask**：模板同步任务，用于追踪 local 存储的同步进度

### 1.2 存储类型分类

```
storage.shared = 1  → 共享存储 (ceph, nfs, iscsi)
storage.shared = 0  → 本地存储 (local)
```

## 二、决策矩阵

### 2.1 Shared 存储（共享存储）

| 行为 | 决策 |
|------|------|
| 文件同步 | ❌ 不需要 |
| 导入节点 | 任意 1 个节点 |
| TemplateInstance | ✅ 创建（逻辑） |
| Instance 数量 | = 该存储可见的节点数 |
| Instance 状态来源 | storage/node 可见性 |

**流程说明**：
1. 用户上传模板文件到共享存储
2. 系统选择任意一个可以访问该存储的节点
3. 在该节点上导入模板（转换为 VM Template）
4. 为所有可以访问该存储的节点创建 TemplateInstance
5. 所有 Instance 状态为 `available`（因为共享存储天然可见）

### 2.2 Local 存储（本地存储）

| 行为 | 决策 |
|------|------|
| 文件同步 | ✅ 需要 |
| 同步范围 | 用户选择 |
| TemplateInstance | ✅ 创建（实体） |
| Instance 数量 | = 用户选择的节点数 |
| Instance 状态来源 | 同步结果 |

**流程说明**：
1. 用户上传模板文件到某个节点的本地存储
2. 系统在该节点上导入模板
3. 为该节点创建 TemplateInstance（状态：`available`）
4. 用户选择需要同步的其他节点
5. 创建同步任务（TemplateSyncTask）
6. 执行文件同步（通过 SSH/SCP）
7. 在目标节点上导入模板
8. 为目标节点创建 TemplateInstance
9. 更新同步任务状态

## 三、数据模型设计

### 3.1 模板上传记录表（template_upload）

```sql
CREATE TABLE IF NOT EXISTS template_upload (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    template_id BIGINT NOT NULL COMMENT '关联的模板ID',
    cluster_id BIGINT NOT NULL COMMENT '集群ID',
    storage_id BIGINT NOT NULL COMMENT '存储ID',
    storage_name VARCHAR(100) NOT NULL COMMENT '存储名称',
    storage_type VARCHAR(50) NOT NULL COMMENT '存储类型',
    is_shared TINYINT NOT NULL DEFAULT 0 COMMENT '是否共享存储：1=共享，0=本地',
    
    upload_node_id BIGINT NOT NULL COMMENT '上传/导入节点ID',
    upload_node_name VARCHAR(100) NOT NULL COMMENT '上传/导入节点名称',
    
    file_name VARCHAR(255) NOT NULL COMMENT '文件名',
    file_path VARCHAR(500) NOT NULL COMMENT '文件路径',
    file_size BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小（字节）',
    file_format VARCHAR(50) NOT NULL COMMENT '文件格式：qcow2, vmdk, raw等',
    
    status VARCHAR(50) NOT NULL DEFAULT 'uploading' COMMENT '状态：uploading, uploaded, importing, imported, failed',
    import_progress INT DEFAULT 0 COMMENT '导入进度：0-100',
    error_message TEXT COMMENT '错误信息',
    
    creator VARCHAR(100) COMMENT '创建人',
    modifier VARCHAR(100) COMMENT '修改人',
    gmt_create DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    gmt_modified DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_template_id (template_id),
    INDEX idx_cluster_id (cluster_id),
    INDEX idx_storage_id (storage_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='模板上传记录表';
```

### 3.2 模板实例表（template_instance）

```sql
CREATE TABLE IF NOT EXISTS template_instance (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    template_id BIGINT NOT NULL COMMENT '关联的模板ID',
    upload_id BIGINT NOT NULL COMMENT '关联的上传记录ID',
    cluster_id BIGINT NOT NULL COMMENT '集群ID',
    
    node_id BIGINT NOT NULL COMMENT '节点ID',
    node_name VARCHAR(100) NOT NULL COMMENT '节点名称',
    
    storage_id BIGINT NOT NULL COMMENT '存储ID',
    storage_name VARCHAR(100) NOT NULL COMMENT '存储名称',
    is_shared TINYINT NOT NULL DEFAULT 0 COMMENT '是否共享存储：1=共享，0=本地',
    
    vmid INT NOT NULL COMMENT 'PVE虚拟机ID',
    volume_id VARCHAR(255) COMMENT '存储卷ID（如：local:100/vm-100-disk-0.qcow2）',
    
    status VARCHAR(50) NOT NULL DEFAULT 'pending' COMMENT '状态：pending, syncing, available, failed, deleted',
    sync_task_id BIGINT COMMENT '关联的同步任务ID（仅local存储）',
    
    is_primary TINYINT DEFAULT 0 COMMENT '是否为主实例：1=主（导入节点），0=从（同步节点）',
    
    creator VARCHAR(100) COMMENT '创建人',
    modifier VARCHAR(100) COMMENT '修改人',
    gmt_create DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    gmt_modified DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    UNIQUE KEY uk_template_node (template_id, node_id),
    INDEX idx_template_id (template_id),
    INDEX idx_upload_id (upload_id),
    INDEX idx_node_id (node_id),
    INDEX idx_storage_id (storage_id),
    INDEX idx_status (status),
    INDEX idx_sync_task_id (sync_task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='模板实例表';
```

### 3.3 模板同步任务表（template_sync_task）

```sql
CREATE TABLE IF NOT EXISTS template_sync_task (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    template_id BIGINT NOT NULL COMMENT '模板ID',
    upload_id BIGINT NOT NULL COMMENT '上传记录ID',
    cluster_id BIGINT NOT NULL COMMENT '集群ID',
    
    source_node_id BIGINT NOT NULL COMMENT '源节点ID',
    source_node_name VARCHAR(100) NOT NULL COMMENT '源节点名称',
    target_node_id BIGINT NOT NULL COMMENT '目标节点ID',
    target_node_name VARCHAR(100) NOT NULL COMMENT '目标节点名称',
    
    storage_name VARCHAR(100) NOT NULL COMMENT '存储名称',
    file_path VARCHAR(500) NOT NULL COMMENT '文件路径',
    file_size BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
    
    status VARCHAR(50) NOT NULL DEFAULT 'pending' COMMENT '状态：pending, syncing, importing, completed, failed',
    progress INT DEFAULT 0 COMMENT '同步进度：0-100',
    
    sync_start_time DATETIME COMMENT '同步开始时间',
    sync_end_time DATETIME COMMENT '同步结束时间',
    
    error_message TEXT COMMENT '错误信息',
    
    creator VARCHAR(100) COMMENT '创建人',
    gmt_create DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    gmt_modified DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_template_id (template_id),
    INDEX idx_upload_id (upload_id),
    INDEX idx_source_node (source_node_id),
    INDEX idx_target_node (target_node_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='模板同步任务表';
```

## 四、业务流程设计

### 4.1 完整流程图（文字版）

```
┌─────────────────────────────────────────────┐
│           用户上传模板文件                      │
│     (选择：集群、存储、模板名称、文件)             │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────┐
│  1. 创建 Template 记录                         │
│  2. 创建 TemplateUpload 记录                   │
│  3. 上传文件到指定存储                          │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
         判断 storage.shared
                   │
        ┌──────────┴──────────┐
        │                     │
        ▼                     ▼
┌───────────────┐    ┌───────────────┐
│ shared = 1    │    │ shared = 0    │
│ （共享存储）   │    │ （本地存储）   │
└───────┬───────┘    └───────┬───────┘
        │                     │
        ▼                     ▼
┌──────────────────────┐  ┌──────────────────────┐
│ 选择任意可见节点       │  │ 在上传节点导入模板     │
│ 导入模板              │  │                      │
└──────────┬───────────┘  └──────────┬───────────┘
           │                         │
           ▼                         ▼
┌──────────────────────┐  ┌──────────────────────┐
│ 为所有可见节点创建     │  │ 为上传节点创建        │
│ TemplateInstance      │  │ TemplateInstance      │
│ status = available    │  │ status = available    │
│ is_primary = 1 (首个) │  │ is_primary = 1        │
│ is_primary = 0 (其他) │  └──────────┬───────────┘
└───────────────────────┘             │
                                      ▼
                            ┌──────────────────────┐
                            │ 用户选择同步策略       │
                            │ - 选择目标节点列表     │
                            └──────────┬───────────┘
                                       │
                                       ▼
                            ┌──────────────────────┐
                            │ 创建同步任务           │
                            │ (TemplateSyncTask)    │
                            │ 为每个目标节点创建     │
                            │ status = pending      │
                            └──────────┬───────────┘
                                       │
                                       ▼
                            ┌──────────────────────┐
                            │ 异步执行同步任务       │
                            │ 1. 传输文件           │
                            │ 2. 导入模板           │
                            │ 3. 创建 Instance      │
                            │ 4. 更新任务状态       │
                            └───────────────────────┘
```

### 4.2 API 接口设计

#### 4.2.1 上传并导入模板

```
POST /api/v1/templates/upload

Request:
{
  "template_name": "centos7-template",
  "cluster_id": 1,
  "storage_id": 10,
  "description": "CentOS 7 模板",
  "file": <multipart file>,
  "auto_sync": false,              // local存储时是否自动同步到所有节点
  "sync_node_ids": [2, 3]          // local存储时，指定要同步的节点ID列表
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "template_id": 1,
    "upload_id": 1,
    "storage_type": "local",
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
      }
    ]
  }
}
```

#### 4.2.2 查询模板详情（包含实例状态）

```
GET /api/v1/templates/{id}/detail

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "template_name": "centos7-template",
    "cluster_id": 1,
    "cluster_name": "prod-cluster",
    "description": "CentOS 7 模板",
    "upload_info": {
      "upload_id": 1,
      "storage_name": "local",
      "is_shared": false,
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
        "status": "available",
        "is_primary": true
      },
      {
        "instance_id": 2,
        "node_id": 2,
        "node_name": "pve-node2",
        "vmid": 9000,
        "status": "syncing",
        "is_primary": false,
        "sync_progress": 45
      }
    ]
  }
}
```

#### 4.2.3 手动同步模板到其他节点

```
POST /api/v1/templates/{id}/sync

Request:
{
  "target_node_ids": [3, 4]
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "sync_tasks": [
      {
        "task_id": 2,
        "target_node_id": 3,
        "target_node_name": "pve-node3",
        "status": "pending"
      },
      {
        "task_id": 3,
        "target_node_id": 4,
        "target_node_name": "pve-node4",
        "status": "pending"
      }
    ]
  }
}
```

#### 4.2.4 查询同步任务状态

```
GET /api/v1/templates/sync-tasks/{task_id}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "task_id": 1,
    "template_id": 1,
    "template_name": "centos7-template",
    "source_node": {
      "node_id": 1,
      "node_name": "pve-node1"
    },
    "target_node": {
      "node_id": 2,
      "node_name": "pve-node2"
    },
    "status": "syncing",
    "progress": 45,
    "sync_start_time": "2025-12-24T10:00:00Z",
    "error_message": null
  }
}
```

#### 4.2.5 列出模板的所有实例

```
GET /api/v1/templates/{id}/instances

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "template_id": 1,
    "template_name": "centos7-template",
    "total": 3,
    "instances": [
      {
        "instance_id": 1,
        "node_id": 1,
        "node_name": "pve-node1",
        "vmid": 9000,
        "storage_name": "local",
        "status": "available",
        "is_primary": true
      },
      {
        "instance_id": 2,
        "node_id": 2,
        "node_name": "pve-node2",
        "vmid": 9000,
        "storage_name": "local",
        "status": "available",
        "is_primary": false
      }
    ]
  }
}
```

## 五、核心业务逻辑

### 5.1 存储类型判断

```go
// 判断存储是否为共享存储
func IsSharedStorage(storage *model.PveStorage) bool {
    return storage.Shared == 1
}

// 获取存储可见的所有节点
func GetStorageVisibleNodes(ctx context.Context, clusterID int64, storageName string) ([]*model.PveNode, error) {
    // 查询该存储在哪些节点上可见
    // shared 存储：所有节点都可见
    // local 存储：仅所属节点可见
}
```

### 5.2 模板导入流程

```go
func ImportTemplate(ctx context.Context, req *UploadTemplateRequest) error {
    // 1. 验证存储是否存在
    storage := GetStorageByID(req.StorageID)
    
    // 2. 上传文件到存储
    uploadNode := SelectUploadNode(storage)
    filePath := UploadFileToStorage(file, uploadNode, storage)
    
    // 3. 在节点上导入模板
    vmid := ImportTemplateToNode(uploadNode, storage, filePath)
    
    // 4. 创建模板记录
    template := CreateTemplate(req)
    
    // 5. 创建上传记录
    upload := CreateTemplateUpload(template, storage, uploadNode, filePath)
    
    // 6. 根据存储类型创建实例
    if IsSharedStorage(storage) {
        // 共享存储：为所有可见节点创建逻辑实例
        nodes := GetStorageVisibleNodes(template.ClusterID, storage.StorageName)
        for i, node := range nodes {
            CreateTemplateInstance(template, upload, node, vmid, i == 0)
        }
    } else {
        // 本地存储：仅为上传节点创建实例
        CreateTemplateInstance(template, upload, uploadNode, vmid, true)
        
        // 如果指定了同步节点，创建同步任务
        if len(req.SyncNodeIDs) > 0 {
            CreateSyncTasks(template, upload, uploadNode, req.SyncNodeIDs)
        }
    }
    
    return nil
}
```

### 5.3 模板同步流程（Local 存储）

```go
func SyncTemplateToNodes(ctx context.Context, templateID int64, targetNodeIDs []int64) error {
    // 1. 获取模板信息
    template := GetTemplate(templateID)
    upload := GetTemplateUpload(templateID)
    
    // 2. 验证是否为本地存储
    if upload.IsShared {
        return errors.New("shared storage does not need sync")
    }
    
    // 3. 获取主实例（源节点）
    primaryInstance := GetPrimaryInstance(templateID)
    sourceNode := GetNode(primaryInstance.NodeID)
    
    // 4. 为每个目标节点创建同步任务
    for _, targetNodeID := range targetNodeIDs {
        // 检查是否已存在实例
        if InstanceExists(templateID, targetNodeID) {
            continue
        }
        
        targetNode := GetNode(targetNodeID)
        task := CreateSyncTask(template, upload, sourceNode, targetNode)
        
        // 异步执行同步
        go ExecuteSyncTask(task)
    }
    
    return nil
}

func ExecuteSyncTask(task *TemplateSyncTask) error {
    // 1. 更新任务状态为 syncing
    UpdateTaskStatus(task.ID, "syncing", 0)
    
    // 2. 传输文件（SSH/SCP）
    err := TransferFile(
        sourceNode, targetNode,
        task.FilePath,
        task.Storage,
    )
    if err != nil {
        UpdateTaskStatus(task.ID, "failed", 0, err.Error())
        return err
    }
    
    // 3. 在目标节点导入模板
    UpdateTaskStatus(task.ID, "importing", 80)
    vmid := ImportTemplateOnNode(targetNode, task.Storage, task.FilePath)
    
    // 4. 创建模板实例
    CreateTemplateInstance(task.TemplateID, task.UploadID, targetNode, vmid, false)
    
    // 5. 更新任务完成
    UpdateTaskStatus(task.ID, "completed", 100)
    
    return nil
}
```

## 六、关键问题处理

### 6.1 存储可见性判断

**问题**：如何确定一个共享存储对哪些节点可见？

**方案**：
1. 从 `pve_storage` 表查询该存储在哪些节点上注册
2. Shared=1 的存储，通常在所有节点上都有记录
3. 通过 PVE API 验证节点是否真的可以访问该存储

```go
// 获取存储可见的节点列表
func GetStorageVisibleNodes(clusterID int64, storageName string) ([]*model.PveNode, error) {
    // 1. 查询该存储的所有记录
    storages := GetStoragesByName(clusterID, storageName)
    
    // 2. 提取节点名称
    nodeNames := make([]string, 0)
    for _, s := range storages {
        nodeNames = append(nodeNames, s.NodeName)
    }
    
    // 3. 查询对应的节点信息
    nodes := GetNodesByNames(clusterID, nodeNames)
    
    return nodes, nil
}
```

### 6.2 VMID 冲突处理

**问题**：
- Shared 存储：所有节点共享同一个 VMID
- Local 存储：每个节点可以使用相同的 VMID（因为存储隔离）

**方案**：
```go
// Shared 存储：所有实例使用相同的 VMID
if isShared {
    vmid := AllocateVMID(clusterID)  // 全局唯一
    for _, node := range nodes {
        instance.VMID = vmid  // 相同的 VMID
    }
}

// Local 存储：每个节点独立分配 VMID
if !isShared {
    for _, node := range nodes {
        vmid := AllocateVMIDForNode(node.ID)  // 节点级唯一
        instance.VMID = vmid  // 不同的 VMID
    }
}
```

### 6.3 同步失败重试

**方案**：
1. 记录失败原因到 `error_message`
2. 提供重试 API
3. 支持手动删除失败的同步任务

```go
POST /api/v1/templates/sync-tasks/{task_id}/retry
DELETE /api/v1/templates/sync-tasks/{task_id}
```

### 6.4 模板删除策略

**问题**：删除模板时如何处理实例？

**方案**：
1. 软删除：仅标记模板为已删除，保留实例
2. 级联删除：删除模板和所有实例（需要调用 PVE API）

```go
DELETE /api/v1/templates/{id}?cascade=true
```

## 七、状态流转图

### 7.1 TemplateUpload 状态流转

```
uploading → uploaded → importing → imported
                  ↓
                failed
```

### 7.2 TemplateInstance 状态流转（Shared）

```
available (创建后直接可用)
    ↓
deleted (模板删除)
```

### 7.3 TemplateInstance 状态流转（Local）

```
pending → syncing → available
              ↓
            failed
              ↓
          (可重试)
```

### 7.4 TemplateSyncTask 状态流转

```
pending → syncing → importing → completed
              ↓
            failed
              ↓
          (可重试)
```

## 八、实现优先级

### P0（核心功能）
1. ✅ 数据模型设计
2. ✅ 基础表创建
3. ✅ 上传并导入模板（Shared 存储）
4. ✅ 上传并导入模板（Local 存储）
5. ✅ 查询模板详情和实例状态

### P1（扩展功能）
6. ✅ 模板同步（Local 存储）
7. ✅ 同步任务状态查询
8. ✅ 同步任务重试

### P2（增强功能）
9. 模板删除（级联删除实例）
10. 批量同步
11. 同步进度实时推送（WebSocket）

## 九、测试场景

### 9.1 Shared 存储场景
1. 上传模板到 Ceph 存储
2. 验证所有节点都创建了 TemplateInstance
3. 验证所有实例状态为 `available`
4. 在任意节点使用模板创建 VM

### 9.2 Local 存储场景
1. 上传模板到节点1的 local 存储
2. 验证仅节点1创建了实例
3. 选择节点2、3进行同步
4. 验证同步任务创建
5. 验证文件传输成功
6. 验证节点2、3的实例创建
7. 在节点2使用模板创建 VM

### 9.3 异常场景
1. 上传失败
2. 导入失败
3. 同步传输失败
4. 目标节点离线
5. 存储空间不足

## 十、性能优化

### 10.1 文件传输优化
- 使用多线程并发传输
- 支持断点续传
- 压缩传输

### 10.2 数据库优化
- 合理使用索引
- 避免 N+1 查询
- 使用缓存（Redis）

### 10.3 异步处理
- 同步任务使用消息队列
- 进度更新使用 WebSocket
- 定时清理过期任务

