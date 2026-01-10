# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0-rc01] - 2026-01-10

### 🧪 Release Candidate 1

PveSphere v1.0.0-rc01 是第一个候选发布版本，提供了完整的 Proxmox VE 多集群管理功能。

⚠️ **注意**: 这是候选版本，建议先在测试环境使用，欢迎反馈问题。经过充分测试后将发布正式的 v1.0.0 版本。

#### ✨ 核心功能

- **多集群管理**：统一管理多个 PVE 集群，无需在不同界面间切换
- **仪表板**：实时监控所有集群、节点、虚拟机和存储资源
- **虚拟机管理**：完整的 VM 生命周期管理（创建、启动、停止、迁移、备份、恢复）
- **模板管理**：跨节点导入和同步 VM 模板，支持共享存储和本地存储
- **存储管理**：监控存储使用情况和管理存储池
- **节点管理**：监控节点状态和资源使用情况
- **控制台访问**：VNC/NoVNC 远程访问虚拟机控制台
- **资源监控**：实时查看 CPU、内存、存储等资源指标

#### 🔧 技术特性

- 基于 Go 1.23+ 和 Nunu 框架构建
- RESTful API 设计
- 支持 Docker 和 Docker Compose 快速部署
- 完整的 Swagger API 文档
- SQLite/MySQL/PostgreSQL 数据库支持

#### 📦 部署方式

- Docker Compose 一键部署
- 独立 Docker 镜像部署
- 源码编译部署

---

[1.0.0-rc01]: https://github.com/pvesphere/pvesphere/releases/tag/v1.0.0-rc01
