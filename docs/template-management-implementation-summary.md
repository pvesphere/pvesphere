# 模板管理系统实现总结

## 📊 实现概览

本文档总结了 PVEsphere 模板管理系统的完整实现过程和架构设计。

## ✅ 已完成的工作

### 1. 数据模型设计 ✓

创建了三个核心数据模型：

#### TemplateUpload（模板上传记录）
- 文件路径：`internal/model/template_upload.go`
- 功能：记录模板文件的上传和导入过程
- 关键字段：
  - `template_id`: 关联模板
  - `storage_id`: 关联存储
  - `upload_node_id`: 上传节点
  - `file_path`: 文件路径
  - `status`: 上传状态（uploading → uploaded → importing → imported → failed）

#### TemplateInstance（模板实例）
- 文件路径：`internal/model/template_instance.go`
- 功能：追踪模板在各个节点上的状态
- 关键字段：
  - `template_id`: 关联模板
  - `node_id`: 所在节点
  - `vmid`: PVE 虚拟机 ID
  - `is_primary`: 是否为主实例（导入节点）
  - `status`: 实例状态（pending → syncing → available → failed）

#### TemplateSyncTask（同步任务）
- 文件路径：`internal/model/template_sync_task.go`
- 功能：管理本地存储模板的跨节点同步
- 关键字段：
  - `source_node_id`: 源节点
  - `target_node_id`: 目标节点
  - `progress`: 同步进度（0-100）
  - `status`: 任务状态（pending → syncing → importing → completed → failed）

### 2. 数据库迁移脚本 ✓

- 文件路径：`scripts/migration_template_management.sql`
- 包含三张表的完整创建语句
- 已添加必要的索引和约束
- 已集成到 AutoMigrate（`internal/server/migration.go`）

### 3. API 接口定义 ✓

- 文件路径：`api/v1/pve_template.go`
- 定义了完整的请求和响应结构
- 接口列表：
  - `POST /api/v1/templates/upload` - 上传并导入模板
  - `GET /api/v1/templates/{id}/detail` - 查询模板详情（含实例）
  - `POST /api/v1/templates/{id}/sync` - 同步模板到其他节点
  - `GET /api/v1/templates/sync-tasks` - 列出同步任务
  - `GET /api/v1/templates/sync-tasks/{task_id}` - 查询同步任务
  - `POST /api/v1/templates/sync-tasks/{task_id}/retry` - 重试同步任务
  - `GET /api/v1/templates/{id}/instances` - 列出模板实例

### 4. Repository 层实现 ✓

创建了三个 Repository：

#### TemplateUploadRepository
- 文件路径：`internal/repository/template_upload.go`
- 方法：Create, Update, Delete, GetByID, GetByTemplateID, UpdateStatus

#### TemplateInstanceRepository
- 文件路径：`internal/repository/template_instance.go`
- 方法：Create, Update, Delete, GetByID, GetByTemplateAndNode, ListByTemplateID, GetPrimaryInstance, UpdateStatus

#### TemplateSyncTaskRepository
- 文件路径：`internal/repository/template_sync_task.go`
- 方法：Create, Update, Delete, GetByID, ListByTemplateID, ListWithPagination, UpdateStatus, GetPendingTasks

#### 扩展现有 Repository

在 `PveStorageRepository` 中添加了 `ListByStorageName` 方法，用于查询共享存储在所有节点上的记录。

### 5. Service 层实现 ✓

- 文件路径：`internal/service/template_management.go`
- 核心服务：`TemplateManagementService`
- 关键方法：

#### UploadAndImportTemplate
上传并导入模板的核心逻辑：
1. 验证存储是否存在
2. 判断存储类型（shared/local）
3. 选择上传节点
4. 创建模板记录
5. 上传文件到存储
6. 导入模板到 PVE
7. 根据存储类型创建实例：
   - **Shared**: 为所有可见节点创建逻辑实例
   - **Local**: 仅为上传节点创建实例，按需创建同步任务

#### GetTemplateDetailWithInstances
查询模板详情，包括：
- 模板基本信息
- 上传信息
- 所有实例状态
- 同步进度（如有）

#### SyncTemplateToNodes
同步模板到其他节点：
1. 验证模板和存储
2. 确认为本地存储
3. 获取主实例作为源
4. 为每个目标节点创建同步任务和实例

#### GetSyncTask / ListSyncTasks
查询同步任务状态和列表

#### RetrySyncTask
重试失败的同步任务

### 6. Handler 层实现 ✓

- 文件路径：`internal/handler/template_management.go`
- 适配了 Gin 框架
- 实现了所有 API 端点的 HTTP 处理
- 包含参数验证和错误处理

### 7. 路由配置 ✓

- 文件路径：`internal/router/pve_template.go`
- 添加了 `InitTemplateManagementRouter` 函数
- 已集成到 HTTP 服务器（`internal/server/http.go`）

### 8. 错误处理 ✓

在 `api/v1/errors.go` 中添加了新的错误定义：
- `ErrStorageNotFound` - 存储不存在
- `ErrNodeNotFound` - 节点不存在
- `ErrFileUploadFailed` - 文件上传失败
- `ErrTemplateImportFailed` - 模板导入失败
- `ErrSharedStorageNoSync` - 共享存储不需要同步
- `ErrInvalidOperation` - 无效操作

### 9. 文档编写 ✓

创建了三份完整文档：

#### 设计文档
- 文件：`docs/template-management-design.md`
- 内容：架构设计、决策矩阵、数据模型、API 设计、业务流程

#### 使用指南
- 文件：`docs/template-management-guide.md`
- 内容：快速开始、API 使用、典型场景、注意事项、FAQ

#### 实现总结
- 文件：`docs/template-management-implementation-summary.md`（本文档）
- 内容：实现概览、架构说明、后续工作

## 🏗️ 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                        用户请求                                │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                   Handler 层（HTTP）                          │
│  - UploadTemplate                                            │
│  - GetTemplateDetail                                         │
│  - SyncTemplate                                              │
│  - GetSyncTask / ListSyncTasks                               │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│               Service 层（业务逻辑）                           │
│  - TemplateManagementService                                 │
│    ├─ UploadAndImportTemplate                                │
│    ├─ GetTemplateDetailWithInstances                         │
│    ├─ SyncTemplateToNodes                                    │
│    ├─ GetSyncTask / ListSyncTasks                            │
│    └─ RetrySyncTask                                          │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              Repository 层（数据访问）                         │
│  - TemplateUploadRepository                                  │
│  - TemplateInstanceRepository                                │
│  - TemplateSyncTaskRepository                                │
│  - PveStorageRepository                                      │
│  - PveNodeRepository                                         │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                   数据库（MySQL）                             │
│  - template_upload                                           │
│  - template_instance                                         │
│  - template_sync_task                                        │
│  - vm_template                                               │
│  - pve_storage                                               │
│  - pve_node                                                  │
└─────────────────────────────────────────────────────────────┘
```

## 🎯 核心决策逻辑

### 存储类型判断

```go
if storage.Shared == 1 {
    // 共享存储逻辑
    // 1. 任意节点上传和导入
    // 2. 为所有可见节点创建逻辑实例
    // 3. 所有实例状态：available
} else {
    // 本地存储逻辑
    // 1. 在指定节点上传和导入
    // 2. 仅为该节点创建实例
    // 3. 按需创建同步任务
}
```

### 实例创建策略

#### 共享存储

```
Template (1)
    │
    ├─ TemplateUpload (1)
    │
    └─ TemplateInstance (N) ← 节点数量
        ├─ Node1: available, is_primary=1
        ├─ Node2: available, is_primary=0
        └─ Node3: available, is_primary=0
```

#### 本地存储

```
Template (1)
    │
    ├─ TemplateUpload (1)
    │
    ├─ TemplateInstance (主实例)
    │   └─ Node1: available, is_primary=1
    │
    └─ TemplateSyncTask (M) ← 同步节点数量
        ├─ Task1: Node1 → Node2
        │   └─ TemplateInstance: pending/syncing/available
        │
        └─ Task2: Node1 → Node3
            └─ TemplateInstance: pending/syncing/available
```

## 🔄 状态流转

### TemplateUpload 状态

```
uploading → uploaded → importing → imported
              ↓
            failed
```

### TemplateInstance 状态（共享存储）

```
available (直接可用)
    ↓
  deleted
```

### TemplateInstance 状态（本地存储）

```
pending → syncing → available
             ↓
           failed (可重试)
```

### TemplateSyncTask 状态

```
pending → syncing → importing → completed
             ↓
           failed (可重试)
```

## ⚠️ 待实现的功能（TODO）

### 1. 文件上传实现

当前 `uploadFileToStorage` 方法是占位实现，需要：
- 集成 Proxmox Client 的文件上传 API
- 处理大文件上传（分片、断点续传）
- 添加上传进度回调
- 实现上传超时和重试

**文件位置**: `internal/service/template_management.go:215-242`

### 2. 模板导入实现

当前 `importTemplateToPVE` 方法是占位实现，需要：
- 调用 PVE API 分配 VMID
- 执行 `qm importdisk` 导入磁盘
- 执行 `qm set` 配置虚拟机
- 执行 `qm template` 转换为模板
- 处理导入错误和回滚

**文件位置**: `internal/service/template_management.go:244-275`

### 3. 异步同步任务执行器

需要实现后台任务处理器，用于执行同步任务：

```go
// 伪代码
type SyncTaskExecutor struct {
    syncTaskRepo     repository.TemplateSyncTaskRepository
    instanceRepo     repository.TemplateInstanceRepository
    proxmoxClientMgr *ProxmoxClientManager
}

func (e *SyncTaskExecutor) Start(ctx context.Context) error {
    // 1. 定时扫描 pending 任务
    // 2. 并发执行同步（限制并发数）
    // 3. 更新任务状态
    // 4. 处理失败重试
}

func (e *SyncTaskExecutor) ExecuteSyncTask(ctx context.Context, task *model.TemplateSyncTask) error {
    // 1. 更新状态为 syncing
    // 2. SSH/SCP 传输文件到目标节点
    // 3. 在目标节点导入模板
    // 4. 创建 TemplateInstance
    // 5. 更新状态为 completed
}
```

**实现建议**：
- 使用 Goroutine Pool 控制并发
- 使用 Context 支持优雅关闭
- 记录详细日志
- 支持进度回调

### 4. WebSocket 实时推送

为了提升用户体验，可以添加 WebSocket 支持：

```go
// 客户端订阅
ws://localhost:8080/api/v1/templates/sync-tasks/{task_id}/ws

// 服务端推送进度
{
  "task_id": 1,
  "status": "syncing",
  "progress": 45,
  "message": "Transferring file..."
}
```

### 5. 依赖注入（Wire）

需要在 Wire 配置中添加新的依赖：

**文件位置**: `cmd/server/wire/wire.go`

```go
// 添加 Repository
wire.Build(
    // ... 现有的 ...
    repository.NewTemplateUploadRepository,
    repository.NewTemplateInstanceRepository,
    repository.NewTemplateSyncTaskRepository,
)

// 添加 Service
wire.Build(
    // ... 现有的 ...
    service.NewTemplateManagementService,
)

// 添加 Handler
wire.Build(
    // ... 现有的 ...
    handler.NewTemplateManagementHandler,
)
```

**执行编译**:

```bash
nunu wire cmd/server/main.go
```

### 6. 文件传输实现

需要实现节点间的文件传输功能：

```go
type FileTransferService interface {
    Transfer(ctx context.Context, sourceNode, targetNode *model.PveNode, filePath string) error
}

// 可选实现方式：
// 1. SSH + SCP
// 2. HTTP 直传
// 3. PVE 内置的 vzdump/qmrestore
```

### 7. 权限控制

添加模板管理的权限验证：
- 只有管理员可以上传模板
- 普通用户只能查看和使用模板
- 记录操作审计日志

### 8. 单元测试

为核心业务逻辑添加单元测试：
- Service 层测试（使用 mock repository）
- Repository 层测试（使用测试数据库）
- Handler 层测试（使用 httptest）

### 9. 性能优化

- 添加 Redis 缓存（模板列表、实例状态）
- 优化数据库查询（避免 N+1）
- 文件传输压缩
- 支持并发上传

### 10. 监控和告警

- 添加 Prometheus 指标（上传成功率、同步耗时）
- 同步失败自动告警
- 存储空间监控

## 📝 使用示例

### 完整的上传流程（本地存储）

```bash
# 1. 上传模板到节点1的本地存储，并同步到节点2、3
curl -X POST http://localhost:8080/api/v1/templates/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "template_name=centos7" \
  -F "cluster_id=1" \
  -F "storage_id=25" \
  -F "sync_node_ids=2,3" \
  -F "file=@centos7.qcow2"

# 响应
{
  "template_id": 1,
  "upload_id": 1,
  "sync_tasks": [
    {"task_id": 1, "target_node_id": 2, "status": "pending"},
    {"task_id": 2, "target_node_id": 3, "status": "pending"}
  ]
}

# 2. 查询同步进度
curl -X GET http://localhost:8080/api/v1/templates/sync-tasks/1 \
  -H "Authorization: Bearer $TOKEN"

# 响应
{
  "task_id": 1,
  "status": "syncing",
  "progress": 45
}

# 3. 查询模板实例
curl -X GET http://localhost:8080/api/v1/templates/1/instances \
  -H "Authorization: Bearer $TOKEN"

# 响应
{
  "instances": [
    {"node_id": 1, "status": "available", "is_primary": true},
    {"node_id": 2, "status": "syncing", "sync_progress": 45},
    {"node_id": 3, "status": "pending"}
  ]
}
```

## 🎓 设计亮点

### 1. 清晰的关注点分离

- **Model**: 纯数据定义，无业务逻辑
- **Repository**: 纯数据访问，无业务逻辑
- **Service**: 核心业务逻辑，不依赖 HTTP
- **Handler**: HTTP 适配层，参数验证和响应格式化

### 2. 存储类型自适应

系统根据存储类型自动选择处理策略，用户无需关心底层细节：
- 共享存储：自动为所有节点创建实例
- 本地存储：按需同步，灵活控制

### 3. 状态追踪

每个环节都有明确的状态：
- 上传状态：uploading → uploaded → importing → imported
- 实例状态：pending → syncing → available
- 任务状态：pending → syncing → completed

### 4. 主从实例标记

通过 `is_primary` 标记主实例：
- 主实例：文件上传和导入的节点
- 从实例：通过同步创建的节点
- 便于追溯和排查问题

### 5. 错误处理和重试

- 失败的任务可以重试
- 记录详细的错误信息
- 支持手动干预

### 6. 扩展性设计

- 接口化设计，便于替换实现
- 支持多种文件传输方式
- 支持多种存储类型

## 🚀 下一步行动

### 立即执行

1. ✅ 执行数据库迁移

```bash
nunu run cmd/migration/main.go
```

2. ✅ 配置 Wire 依赖注入

```bash
nunu wire cmd/server/main.go
```

3. ⏳ 实现文件上传逻辑（集成 Proxmox Client）

4. ⏳ 实现模板导入逻辑（调用 PVE API）

5. ⏳ 实现异步同步任务执行器

### 短期计划（1-2 周）

- 完成基础功能的实现
- 添加单元测试
- 完善错误处理
- 性能优化

### 中期计划（1 个月）

- 添加 WebSocket 实时推送
- 实现文件传输（SSH/SCP）
- 添加监控和告警
- 完善文档和示例

### 长期计划（3 个月）

- 支持模板版本管理
- 支持模板市场（分享和下载）
- 支持模板自动更新
- 集成 CI/CD（自动化模板构建）

## 📚 相关资源

- [设计文档](./template-management-design.md)
- [使用指南](./template-management-guide.md)
- [数据库迁移脚本](../scripts/migration_template_management.sql)
- [Proxmox VE API 文档](https://pve.proxmox.com/pve-docs/api-viewer/)

## 📌 总结

本次实现完成了模板管理系统的核心架构和基础功能，包括：

✅ 完整的数据模型设计
✅ 清晰的 API 接口定义
✅ 分层的代码架构
✅ 详细的文档说明

虽然还有一些核心功能待实现（文件上传、模板导入、异步同步），但整体架构已经搭建完成，后续工作可以按照设计逐步完善。

这套系统的设计充分考虑了 PVE 的特性（共享存储 vs 本地存储），提供了灵活且强大的模板管理能力，为用户提供了良好的体验。

