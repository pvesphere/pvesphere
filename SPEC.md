# PveSphere v1.1.0 主机维护功能完善计划

## 一、当前代码问题汇总

### 后端问题 (P0-P2)

| 优先级 | 问题 | 文件 | 修复建议 |
|--------|------|------|----------|
| P0 | 事务处理缺失 | internal/service/pve_node_maintenance.go | 迁移过程使用数据库事务 |
| P0 | 并发计数逻辑错误 | internal/service/pve_node_maintenance.go | 修复 atomic.AddInt32(&successCount, -1) 错误 |
| P0 | 缺少异步任务支持 | internal/service/pve_node_maintenance.go | 添加任务队列支持 |
| P1 | API字段未赋值 | api/v1/pve_node_maintenance.go | CPUCores/VMCount等字段补充赋值 |
| P1 | 缺少状态查询接口 | api/v1/pve_node_maintenance.go | 添加维护状态查询 |
| P2 | 并发数无上限 | internal/service/pve_node_maintenance.go | 添加最大并发限制 |
| P2 | 参数验证不完整 | internal/handler/pve_node_maintenance.go | 添加参数校验 |

### 前端问题 (P0-P2)

| 优先级 | 问题 | 文件 | 修复建议 |
|--------|------|------|----------|
| P0 | 翻译键未定义 | locales/*.yaml | 添加完整的 node.maintenance 命名空间 |
| P0 | 缺少进度轮询 | NodeMaintenance.vue | 添加定时轮询维护进度 |
| P0 | 错误处理缺失 | NodeMaintenance.vue | catch 添加 ElMessage.error |
| P1 | 手动模式目标选择缺失 | NodeMaintenance.vue | 添加 el-table 多选目标节点 |
| P1 | 维护状态显示不完整 | NodeMaintenance.vue | 添加 failed 状态处理 |
| P2 | 缺少结束维护按钮 | NodeMaintenance.vue | 维护完成后显示退出按钮 |
| P2 | API类型不一致 | src/api/pve.ts | 统一 vm_id 类型 |

## 二、开发任务清单

### 阶段1: 后端完善

- [ ] 1.1 修复并发计数错误 (atomic 操作)
- [ ] 1.2 添加数据库事务支持
- [ ] 1.3 完善 API 字段赋值 (CPUCores, VMCount)
- [ ] 1.4 添加维护状态查询接口
- [ ] 1.5 添加参数验证
- [ ] 1.6 编译验证

### 阶段2: 前端完善

- [ ] 2.1 添加国际化翻译
- [ ] 2.2 完善手动模式目标选择
- [ ] 2.3 添加进度轮询
- [ ] 2.4 完善错误处理
- [ ] 2.5 添加结束维护按钮
- [ ] 2.6 编译验证

### 阶段3: 部署测试

- [ ] 3.1 停止当前运行版本
- [ ] 3.2 提交代码到 feature/v1.1.0 分支
- [ ] 3.3 构建 Docker 镜像
- [ ] 3.4 部署到 /opt/pvesphere
- [ ] 3.5 功能验证

## 三、开发原则

1. 本地开发测试
2. 每次提交前确认用户
3. 容器测试通过后再 push
