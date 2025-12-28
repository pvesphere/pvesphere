# PVE Controller 快速开始

## 功能简介

PVE Controller 是一个独立的服务，用于自动同步 Proxmox VE 集群的资源信息（节点、虚拟机、存储）到数据库中。

## 快速启动

### 1. 创建数据库表

```bash
nunu run cmd/migration/main.go
```

这将创建以下表：
- `pve_cluster` - PVE 集群配置
- `pve_node` - PVE 节点信息
- `pve_vm` - 虚拟机信息
- `pve_storage` - 存储信息
- `vm_ipaddress` - 虚拟机 IP 地址
- `vm_template` - 虚拟机模板

### 2. 添加集群配置

在数据库中插入集群记录（示例）：

```sql
INSERT INTO pve_cluster (
    cluster_name, api_url, user_id, user_token, 
    is_enabled, is_schedulable, env
) VALUES (
    'my-cluster',
    'https://your-pve-host:8006',
    'your-user@pve',
    'your-api-token',
    1,  -- is_enabled: 启用数据自动上报
    1,  -- is_schedulable: 允许虚拟机调度
    'prod'
);
```

**关键字段说明**：
- `is_enabled` - **数据自动上报开关**（`1`=启用，`0`=禁用）
  - 设置为 `1` 时，控制器会同步该集群的资源数据
  - 设置为 `0` 时，控制器会停止同步该集群的数据
- `is_schedulable` - **虚拟机调度开关**（`1`=可调度，`0`=不可调度）
  - 仅用于控制虚拟机创建时的调度决策
  - **不影响数据自动上报功能**
- `api_url` - Proxmox API 地址
- `user_id` - API Token 用户名（格式：`username@realm`）
- `user_token` - API Token 密钥

### 3. 启动控制器

```bash
nunu run cmd/controller/main.go
```

控制器将自动：
- ✅ 发现**所有启用的集群**（`is_enabled = 1`）
- ✅ 每 30 秒检查新集群
- ✅ 每 5 秒轮询资源变化
- ✅ 每 5 分钟全量同步
- ✅ 自动同步节点、虚拟机、存储信息

## 字段说明

### is_enabled vs is_schedulable

| 字段 | 用途 | 控制器行为 |
|------|------|-----------|
| `is_enabled` | 控制数据自动上报 | `1`=同步数据，`0`=停止同步 |
| `is_schedulable` | 控制虚拟机调度 | `1`=可调度，`0`=不可调度 |

**重要**：
- 数据上报由 `is_enabled` 控制
- 虚拟机调度由 `is_schedulable` 控制
- 两者相互独立，互不影响

## 架构说明

### 核心机制

采用类 Kubernetes Informer 机制：

```
ListWatcher → Reflector → DeltaFIFO → EventHandler → Database
     ↓            ↓            ↓             ↓
  轮询 API    检测变化    生成事件      处理事件
```

### 工作流程

1. **集群发现**: 从数据库加载 `is_enabled = 1` 的集群
2. **创建 Informer**: 为每个集群创建 Node/VM/Storage Informer
3. **持续同步**: 
   - 轮询检测资源变化
   - 生成 Add/Update/Delete 事件
   - 同步到数据库

## 数据流

```
Proxmox API
    ↓
ListWatcher (轮询)
    ↓
Reflector (检测变化)
    ↓
DeltaFIFO (事件队列)
    ↓
EventHandler (处理事件)
    ↓
Repository (数据库操作)
    ↓
Database
```

## 运行时管理

### 启用集群数据上报

```sql
UPDATE pve_cluster SET is_enabled = 1 WHERE cluster_name = 'my-cluster';
```

控制器会在**最多 30 秒内**自动加载并开始同步数据，无需重启服务。

### 禁用集群数据上报

```sql
UPDATE pve_cluster SET is_enabled = 0 WHERE cluster_name = 'my-cluster';
```

控制器会在**最多 30 秒内**停止同步该集群的数据。

### 禁用虚拟机调度（保持数据上报）

```sql
UPDATE pve_cluster SET is_schedulable = 0 WHERE cluster_name = 'my-cluster';
```

数据上报会继续，但该集群不会被用于创建新虚拟机。

## 配置说明

### 同步间隔

- **资源轮询**: 5 秒（在 `list_watcher.go` 中配置）
- **全量同步**: 5 分钟（在 `wire.go` 中配置）
- **集群检查**: 30 秒（在 `pve_controller.go` 中配置）

### 日志

控制器会输出详细的日志信息：
- 集群发现和启动
- 资源同步事件
- 错误信息

## 常见问题

### Q: 集群没有被管理？

A: 检查以下几点：
1. 集群记录是否存在于数据库中
2. `is_enabled` 字段是否为 `1`
3. API 连接配置是否正确
4. 查看日志了解具体错误

### Q: 资源同步失败？

A: 
1. 检查 API 连接（URL、Token）
2. 查看日志了解具体错误
3. 确认数据库表已创建

### Q: 运行时添加集群未自动加载？

A: 控制器每 30 秒检查一次新集群，确保：
1. `is_enabled = 1`
2. 等待最多 30 秒后会自动加载

### Q: 调度状态会影响数据同步吗？

A: **不会**。`is_schedulable` 字段只用于控制虚拟机创建时的调度决策，不影响数据自动上报功能。数据同步由 `is_enabled` 字段控制。

### Q: 如何停止某个集群的数据同步？

A: 将集群的 `is_enabled` 设置为 `0`，控制器会在下次检查（30秒内）时停止该集群的 Informer。

## 详细文档

完整文档请参考：[pve-controller.md](./pve-controller.md)

