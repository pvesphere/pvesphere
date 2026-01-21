# PveSphere

[![license](https://img.shields.io/github/license/pvesphere/pvesphere-ui.svg)](LICENSE)

**English** | [中文](./README_zh.md)

---

## Introduction

PveSphere is a web-based multi-cluster management platform for Proxmox VE (PVE). It provides a unified interface to manage multiple PVE clusters, nodes, virtual machines, storage, and templates from a single dashboard.

<img src="./docs/pvesphere-review03.gif" width="100%" />

## Why PveSphere?

Proxmox VE is a powerful open-source virtualization platform, but managing multiple clusters can be challenging:

- **Multiple Cluster Management**: Need to switch between different PVE web interfaces when managing multiple clusters
- **Unified Monitoring**: Difficult to get a global view of resources across all clusters
- **Template Synchronization**: Manually syncing VM templates across nodes is time-consuming and error-prone
- **Operational Complexity**: Managing VMs, storage, and backups across clusters requires frequent context switching

PveSphere solves these problems by providing a **centralized management platform** that simplifies multi-cluster operations and improves operational efficiency.

## What Problems Does It Solve?

### 🎯 Core Pain Points

1. **Multi-Cluster Management**: Unified interface for managing multiple PVE clusters without switching between different web UIs
2. **Resource Visibility**: Real-time monitoring of all clusters, nodes, VMs, and storage from a single dashboard
3. **Template Management**: Automated template synchronization across nodes, supporting both shared and local storage
4. **Simplified Operations**: Streamlined workflows for VM lifecycle management, migrations, backups, and monitoring
5. **Better UX**: Modern, responsive web interface that's easier to use than the native PVE interface for many operations

### 🔍 Specific Use Cases

- Managing multiple PVE clusters across different locations or environments
- Centralized monitoring and alerting for PVE infrastructure
- Automated template distribution across nodes
- Simplified VM provisioning and management workflows
- Better resource visibility for capacity planning

## Who Is PveSphere For?

### ✅ Suitable For

- **Multi-cluster operators**: Teams managing multiple PVE clusters who need centralized management
- **Small to medium teams**: Small DevOps teams that want to simplify PVE operations
- **Template-heavy environments**: Users who frequently deploy VMs from templates and need automated synchronization
- **Resource monitoring focused**: Teams that need better visibility into resource usage across clusters
- **Open-source enthusiasts**: Users who prefer open-source solutions and want to avoid vendor lock-in

### ❌ Not Suitable For

- **Single cluster users**: If you only manage one PVE cluster, the native PVE web interface may be sufficient
- **Large-scale requirements**: Teams that already operate large-scale OpenStack environments
- **Complex HA/DR scenarios**: Advanced high-availability and disaster recovery requirements may need specialized tools
- **API-only users**: If you primarily interact with PVE via API/CLI, you may not need a web management interface

## Real-World Scenarios

### Scenario 1: Multi-Environment Management
You run development, staging, and production PVE clusters. Instead of logging into three different PVE web interfaces, you can manage all three from PveSphere's unified dashboard.

### Scenario 2: Template Distribution
You have VM templates stored on local storage and need to make them available on multiple nodes. PveSphere automates the synchronization process, saving hours of manual work.

### Scenario 3: Resource Monitoring
You need to monitor resource utilization across multiple clusters to identify hotspots and plan capacity. PveSphere provides a centralized view of CPU, memory, and storage usage.

### Scenario 4: Simplified VM Operations
Your team frequently creates, migrates, and manages VMs across clusters. PveSphere provides a streamlined interface that reduces the number of clicks and context switches needed.

## Features

### Core Functionality

- **Multi-Cluster Dashboard**: Real-time overview of all clusters, nodes, VMs, and storage resources
- **Cluster Management**: Add and manage multiple PVE clusters with centralized configuration
- **VM Lifecycle Management**: Create, start, stop, migrate, backup, and restore VMs
- **Template Management**: Import, synchronize, and manage VM templates across nodes
- **Storage Management**: Monitor storage usage and manage storage pools
- **Node Management**: Monitor nodes and access node consoles via terminal proxy
- **VM Console Access**: VNC/NoVNC console access for VMs
- **Resource Monitoring**: Real-time metrics and utilization tracking

For detailed documentation, please visit: [https://docs.pvesphere.com](https://docs.pvesphere.com)

## Quick Start

### Prerequisites

- Docker >= 20.10
- Docker Compose >= 2.0

### Quick Start with Docker

```bash
# Clone the repository
git clone https://github.com/pvesphere/pvesphere.git
cd pvesphere

# Build and start all services
make docker-compose-build

# View service logs
make docker-compose-logs

# Stop all services
make docker-compose-down
```

### All-in-One Deployment (Single Container)

If you just want to quickly try PveSphere or run it as a single container, you can use the prebuilt **All-in-One** image (backend + frontend in one container):

```bash
docker run -d \
  --name=pvesphere \
  --restart=always \
  -p 8080:8080 \
  pvesphere/pvesphere-aio:latest
```

Then open:

- `http://localhost:8080`

### Default Login

After first startup, use these credentials:
- **Email**: `pvesphere@gmail.com`
- **Password**: `Ab123456`

> ⚠️ Remember to change the default password after first login.

### Access Services

- **API Service**: http://localhost:8000
- **API Documentation**: http://localhost:8000/swagger/index.html

For detailed installation and configuration instructions, please visit: [https://docs.pvesphere.com](https://docs.pvesphere.com)

## Roadmap

PveSphere focuses on **stability and correctness over speed of features**. The following high-level roadmap outlines planned improvements:

### Core Improvements

- **Improved Template Lifecycle Tracking**: Enhanced visibility and management of template states throughout their lifecycle
- **Safer Multi-Cluster Orchestration**: More robust and reliable operations when managing resources across multiple clusters
- **Enhanced Automation Workflows**: Streamlined automation capabilities for common operational tasks
- **Better Observability and Auditability**: Comprehensive monitoring, logging, and audit trails for all operations

### Planned Features

- **Advanced Storage Management**: More comprehensive storage operations and management capabilities
- **Cluster-Level VM Scheduling**: Intelligent VM placement and scheduling mechanisms at the cluster level
- **DRS (Distributed Resource Scheduler)**: Secondary scheduling capabilities for dynamic resource rebalancing
- **Role-Based Access Control (RBAC)**: Fine-grained access control and permission management
- **Extended Monitoring & Logging**: Enhanced monitoring capabilities and comprehensive logging management

> **Note**: Roadmap items prioritize stability and correctness. Features will be released when they are thoroughly tested and production-ready, not based on arbitrary timelines.

## Documentation

For comprehensive documentation, including installation guides, API reference, and usage examples, please visit:

**📚 [https://docs.pvesphere.com](https://docs.pvesphere.com)**

## Contributing

Contributions are welcome! Whether it's bug fixes, feature enhancements, or documentation improvements, we appreciate your help in making PveSphere better.

Please feel free to submit issues and pull requests.

## License

[Apache License 2.0](LICENSE)

Copyright © 2025-present PveSphere Contributors

## Contact

- **Email**: pvesphere@gmail.com
- **Twitter**: [@PveSphere](https://x.com/PveSphere)
- **GitHub**: [https://github.com/pvesphere/pvesphere](https://github.com/pvesphere/pvesphere)

---

## Related Projects

- [Proxmox VE](https://www.proxmox.com/) - The underlying virtualization platform
- [vue-pure-admin](https://github.com/pure-admin/vue-pure-admin) - Frontend base template
- [Nunu](https://github.com/go-nunu/nunu) - Backend framework
