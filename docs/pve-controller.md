# PVE Controller 功能文档

## 概述

PVE Controller 是一个基于类 Kubernetes Informer 机制的 Proxmox VE 资源同步控制器服务。它能够自动发现并实时同步 Proxmox 集群中的节点（Node）、虚拟机（VM）和存储（Storage）资源到数据库中。

## 架构设计

### 核心组件

#### 1. Informer 框架

实现了类似 Kubernetes Informer 的机制，包含以下核心组件：

- **ListWatcher**: 负责列出和监听资源变化
  - `NodeListWatcher`: 监听节点资源
  - `VMListWatcher`: 监听虚拟机资源
  - `StorageListWatcher`: 监听存储资源

- **Reflector**: 定期从 API 获取资源列表，检测变化
- **DeltaFIFO**: 增量队列，记录资源的变化（Added/Updated/Deleted）
- **Local Cache**: 本地缓存，存储当前资源状态
- **EventHandler**: 事件处理器，处理资源变化并同步到数据库

#### 2. 数据模型

- **PveCluster**: PVE 集群配置信息
- **PveNode**: PVE 节点信息
- **PveVM**: 虚拟机信息
- **PveStorage**: 存储信息
- **VMIPAddress**: 虚拟机 IP 地址信息
- **VmTemplate**: 虚拟机模板信息

#### 3. 服务组件

- **PveController**: 主控制器，管理多个集群的 Informer
- **ControllerServer**: 服务器封装，实现 server.Server 接口

## 功能特性

### 1. 自动发现集群

- 从数据库加载所有 `is_enabled = 1` 的集群
- 每 30 秒检查一次新集群
- 自动启动新集群的 Informer
- 只在集群被禁用或删除时才停止 Informer

**重要说明**：
- `is_enabled` 字段控制数据自动上报功能（`1` = 启用，`0` = 禁用）
- `is_schedulable` 字段只用于控制虚拟机创建时的调度决策，**不影响数据同步**

### 2. 实时资源同步

- **节点同步**: 自动同步集群中的所有节点
- **虚拟机同步**: 为每个节点同步其上的虚拟机
- **存储同步**: 为每个节点同步其存储信息

### 3. 轮询机制

- **监听间隔**: 5 秒轮询一次，检测资源变化
- **全量同步**: 每 5 分钟进行一次全量同步
- **版本检测**: 使用 MD5 哈希检测资源变化

### 4. 事件处理

支持三种资源变化事件：
- **OnAdd**: 资源新增
- **OnUpdate**: 资源更新
- **OnDelete**: 资源删除

### 5. 运行时动态加载

- 新增集群后，控制器会在最多 30 秒内自动加载
- 无需重启服务即可开始数据上报
- 支持动态启用/禁用集群的数据上报

## 使用方法

### 1. 数据库迁移

运行数据库迁移创建所有必要的表：

```bash
nunu run cmd/migration/main.go
```

### 2. 添加 PVE 集群

在数据库中插入集群配置记录：

```sql
INSERT INTO pve_cluster (
    cluster_name,
    cluster_name_alias,
    env,
    api_url,
    user_id,
    user_token,
    is_enabled,
    is_schedulable,
    creator
) VALUES (
    'my-cluster',
    '生产集群',
    'prod',
    'https://10.7.64.206:8006',
    'api-user@pve',
    'your-token-here',
    1,  -- is_enabled: 启用数据自动上报
    1,  -- is_schedulable: 允许虚拟机调度
    'admin'
);
```

**重要参数说明**：

- `is_enabled`: 
  - `1`: 启用数据自动上报功能
  - `0`: 禁用数据自动上报功能
  - **注意**: 此字段**控制数据同步**，设置为 `0` 时控制器会停止同步该集群的数据
  
- `is_schedulable`: 
  - `1`: 允许在此集群创建虚拟机（用于调度决策）
  - `0`: 禁止在此集群创建虚拟机
  - **注意**: 此字段**不影响数据自动上报功能**，只影响虚拟机创建时的调度

- `api_url`: Proxmox API 地址，格式：`https://host:port`
- `user_id`: API Token 的用户名，格式：`username@realm`
- `user_token`: API Token 的密钥值

### 3. 启动控制器

```bash
nunu run cmd/controller/main.go
```

### 4. 查看日志

控制器启动后会：
- 自动发现所有启用的集群（`is_enabled = 1`）
- 为每个集群启动 Informer
- 定期同步资源到数据库

日志示例：
```
starting PVE controller
cluster informer started cluster=my-cluster id=1
reflector initial list completed name=node-my-cluster count=3
node added node=pve-node-1 cluster_id=1
vm added vm=vm-100 vmid=100
storage added storage=local node=pve-node-1
```

### 5. 运行时管理集群

#### 启用数据上报
```sql
UPDATE pve_cluster SET is_enabled = 1 WHERE cluster_name = 'my-cluster';
```
控制器会在最多 30 秒内自动加载并开始同步数据。

#### 禁用数据上报
```sql
UPDATE pve_cluster SET is_enabled = 0 WHERE cluster_name = 'my-cluster';
```
控制器会在最多 30 秒内停止同步该集群的数据。

#### 禁用虚拟机调度（但保持数据上报）
```sql
UPDATE pve_cluster SET is_schedulable = 0 WHERE cluster_name = 'my-cluster';
```
数据上报会继续，但该集群不会被用于创建新虚拟机。

## 项目结构

```
pvesphere/
├── cmd/
│   └── controller/              # 控制器主程序
│       ├── main.go
│       └── wire/
│           ├── wire.go
│           └── wire_gen.go
├── internal/
│   ├── controller/              # 控制器核心逻辑
│   │   ├── handler.go          # 事件处理器
│   │   ├── pve_controller.go   # 主控制器
│   │   └── informer/           # Informer 框架
│   │       ├── cache.go
│   │       ├── delta_fifo.go
│   │       ├── informer.go
│   │       ├── list_watcher.go
│   │       ├── reflector.go
│   │       └── types.go
│   ├── model/                   # 数据模型
│   │   ├── pve_cluster.go
│   │   ├── pve_node.go
│   │   ├── pve_vm.go
│   │   ├── pve_storage.go
│   │   ├── vm_ipaddress.go
│   │   └── vm_template.go
│   ├── repository/              # 数据访问层
│   │   ├── pve_cluster.go
│   │   ├── pve_node.go
│   │   ├── pve_vm.go
│   │   ├── pve_storage.go
│   │   ├── vm_ipaddress.go
│   │   └── vm_template.go
│   └── server/
│       └── controller.go        # 控制器服务器
└── pkg/
    └── proxmox/                 # Proxmox API 客户端
        ├── client.go
        └── types.go
```

## API 接口说明

### Proxmox API 端点

控制器使用以下 Proxmox API 端点：

- `GET /api2/json/nodes` - 获取节点列表
- `GET /api2/json/nodes/{node}/qemu` - 获取节点上的虚拟机列表
- `GET /api2/json/nodes/{node}/storage` - 获取节点的存储列表

### 认证方式

使用 API Token 认证：
```
Authorization: PVEAPIToken=username@realm=token-value
```

## 配置说明

### Resync 周期

默认配置：
- **轮询间隔**: 5 秒
- **全量同步周期**: 5 分钟

可以在 `cmd/controller/wire/wire.go` 中修改：

```go
wire.Value(time.Minute*5) // resyncPeriod
```

### 集群检查间隔

默认每 30 秒检查一次新集群，可在 `pve_controller.go` 中修改：

```go
ticker := time.NewTicker(30 * time.Second)
```

## 字段说明

### is_enabled vs is_schedulable

| 字段 | 用途 | 影响范围 |
|------|------|----------|
| `is_enabled` | 控制数据自动上报 | 控制器是否会同步该集群的数据 |
| `is_schedulable` | 控制虚拟机调度 | 虚拟机创建时是否可以选择该集群 |

**使用场景示例**：

1. **维护模式**：禁用调度但保持数据上报
   ```sql
   UPDATE pve_cluster SET is_schedulable = 0, is_enabled = 1 WHERE id = ?;
   ```
   结果：不再创建新虚拟机，但数据继续同步

2. **暂停监控**：禁用数据上报但保持调度
   ```sql
   UPDATE pve_cluster SET is_enabled = 0, is_schedulable = 1 WHERE id = ?;
   ```
   结果：不再同步数据，但仍可用于创建虚拟机（虽然不推荐）

3. **完全禁用**：同时禁用调度和数据上报
   ```sql
   UPDATE pve_cluster SET is_enabled = 0, is_schedulable = 0 WHERE id = ?;
   ```

## 故障处理

### 常见问题

1. **集群没有被管理**
   - 检查集群的 `is_enabled` 字段是否为 `1`
   - 检查 API 连接配置是否正确
   - 查看日志了解具体错误

2. **资源同步失败**
   - 检查 API 连接（URL、Token）
   - 查看日志了解具体错误
   - 确认数据库表已创建

3. **运行时添加集群未自动加载**
   - 控制器每 30 秒检查一次新集群
   - 确保 `is_enabled = 1`
   - 等待最多 30 秒后会自动加载

4. **调度状态会影响数据同步吗？**
   - **不会**。`is_schedulable` 字段只用于控制虚拟机创建时的调度决策，不影响数据自动上报功能。
   - 数据同步由 `is_enabled` 字段控制。

## 性能优化

1. **并发控制**: 使用 goroutine 并发处理多个集群
2. **版本检测**: 使用 MD5 哈希避免不必要的数据库操作
3. **批量操作**: Repository 层使用 Upsert 减少数据库操作
4. **连接池**: 数据库连接池配置了合理的连接数
5. **锁优化**: 在锁外执行耗时操作，避免阻塞

## 扩展开发

### 添加新的资源类型

1. 创建对应的 ListWatcher（实现 `ListWatcher` 接口）
2. 创建对应的 EventHandler（实现 `EventHandler` 接口）
3. 在 `pve_controller.go` 中注册 Informer

### 自定义事件处理

可以在 EventHandler 中添加自定义逻辑，如：
- 发送通知
- 触发其他操作
- 数据转换

## 注意事项

1. **API 限制**: Proxmox API 可能有速率限制，轮询间隔不宜过短
2. **数据库性能**: 大量资源时注意数据库性能
3. **资源清理**: 禁用或删除的集群会停止同步，但历史数据不会自动清理
4. **安全性**: API Token 存储在数据库中，注意数据库安全

## 版本历史

- **v1.0.0**: 初始版本
  - 支持节点、虚拟机、存储的资源同步
  - 实现类 Kubernetes Informer 机制
  - 支持多集群管理
  - 支持运行时动态加载集群
  - 区分调度状态和数据上报状态

