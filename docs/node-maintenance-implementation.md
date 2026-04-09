# 主机维护功能实现总结

## 实现概述

本次实现完成了主机维护功能的持久化、异步执行、进度查询和历史记录等核心功能。

## 实现内容

### 1. 数据模型 (internal/model/node_maintenance.go)

新增两个数据表：

**node_maintenance_plan (维护计划表)**
- `id`: 主键
- `node_id`, `node_name`: 节点信息
- `status`: pending/running/completed/failed/cancelled
- `mode`: auto/manual
- `concurrency`: 并发数
- `total_vms`, `migrated_vms`, `failed_vms`: VM 统计
- `start_time`, `end_time`: 时间记录
- `failed_vm_ids`: 失败 VM ID 列表
- `error_message`: 错误信息

**node_maintenance_task (维护任务表)**
- `id`: 主键
- `plan_id`: 关联维护计划
- `vm_id`, `vm_name`: VM 信息
- `source_node_id`, `source_node`: 源节点
- `target_node_id`, `target_node`: 目标节点
- `status`: pending/running/completed/failed
- `error_message`: 错误信息
- `started_at`, `completed_at`: 时间记录

### 2. 数据库迁移 (internal/server/migration.go)

添加了 `NodeMaintenancePlan` 和 `NodeMaintenanceTask` 的 AutoMigrate。

### 3. 服务层 (internal/service/pve_node_maintenance.go)

**新增方法：**
- `CreateMaintenancePlan`: 创建维护计划（异步）
- `executeMaintenancePlan`: 后台执行迁移任务
- `GetMaintenancePlan`: 获取计划详情
- `GetMaintenancePlanTasks`: 获取任务列表（分页）
- `GetMaintenancePlanProgress`: 获取进度
- `CancelMaintenancePlan`: 取消计划
- `GetMaintenanceHistory`: 获取历史记录

**修改方法：**
- `EndMaintenance`: 增加校验，有任务运行时禁止结束

### 4. API 定义 (api/v1/pve_node_maintenance.go)

新增 API 结构体：
- `CreateMaintenancePlanRequest/Response`
- `MaintenancePlanDetail/Response`
- `MaintenancePlanItem/ListResponse`
- `MaintenanceTaskDetail/Item/ListResponse`
- `MaintenancePlanProgress/ProgressResponse`
- `CancelMaintenancePlanRequest/Response`
- `MaintenanceHistoryResponse`
- `HistoryRecord`

### 5. Handler (internal/handler/pve_node_maintenance.go)

新增处理函数：
- `CreateMaintenancePlan`: POST /api/v1/nodes/maintenance/plans
- `GetMaintenancePlan`: GET /api/v1/nodes/maintenance/plans/:plan_id
- `GetMaintenancePlanTasks`: GET /api/v1/nodes/maintenance/plans/:plan_id/tasks
- `GetMaintenancePlanProgress`: GET /api/v1/nodes/maintenance/plans/:plan_id/progress
- `CancelMaintenancePlan`: POST /api/v1/nodes/maintenance/plans/:plan_id/cancel
- `GetMaintenanceHistory`: GET /api/v1/nodes/maintenance/history

### 6. 路由 (internal/router/pve_node_maintenance.go)

```
/nodes/maintenance
  /candidates     [GET]  获取可迁移目标节点
  /vms            [GET]  获取节点 VM 列表
  /start          [POST] 开始维护（同步，向后兼容）
  /end            [POST] 结束维护（带校验）
  /plans
    /             [POST] 创建维护计划（异步）
    /:plan_id     [GET]  获取计划详情
    /:plan_id/tasks    [GET] 获取任务列表
    /:plan_id/progress [GET] 获取进度
    /:plan_id/cancel   [POST] 取消计划
  /history
    /             [GET] 获取历史记录
```

## 功能特性

### P0 - 已完成

1. **任务持久化**
   - 迁移开始前创建计划和任务记录
   - 迁移过程中实时更新状态
   - 服务重启后任务不丢失

2. **进度查询**
   - 支持查询计划整体进度
   - 支持查询单个任务状态
   - 支持分页查询任务列表

3. **EndMaintenance 校验**
   - 检查是否有运行中的任务
   - 有任务运行时返回错误提示

4. **维护历史**
   - 记录已完成的维护计划
   - 支持按节点筛选
   - 支持分页查询

### 保留功能

- `StartMaintenance` 同步版本保留，向后兼容

## API 使用示例

### 创建维护计划（异步）
```bash
POST /api/v1/nodes/maintenance/plans
{
  "node_id": "1",
  "mode": "auto",
  "concurrency": 3
}
```

响应：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "plan_id": 1,
    "node_id": 1,
    "node_name": "pve-node-1",
    "status": "pending",
    "total_vms": 5
  }
}
```

### 查询进度
```bash
GET /api/v1/nodes/maintenance/plans/1/progress
```

### 查询历史
```bash
GET /api/v1/nodes/maintenance/history?page=1&page_size=20
```

## 验证方法

1. 启动维护，关闭浏览器后再打开能看到进度
2. 维护过程中重启服务，任务不丢失
3. 有任务运行时，EndMaintenance 返回错误提示
4. 维护完成后，能在历史记录中查询

## 文件清单

- [x] `internal/model/node_maintenance.go` (新增)
- [x] `internal/server/migration.go` (修改)
- [x] `internal/service/pve_node_maintenance.go` (修改)
- [x] `internal/handler/pve_node_maintenance.go` (修改)
- [x] `internal/router/pve_node_maintenance.go` (修改)
- [x] `api/v1/pve_node_maintenance.go` (修改)
- [x] `docs/swagger.json` (更新)
