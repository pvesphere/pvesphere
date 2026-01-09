package informer

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pvesphere/internal/model"
	"pvesphere/pkg/proxmox"
)

// NodeListWatcher 节点列表监听器
type NodeListWatcher struct {
	client       *proxmox.ProxmoxClient
	clusterID    int64
	clusterName  string
	lastVersion  string
	pollInterval time.Duration
}

func NewNodeListWatcher(client *proxmox.ProxmoxClient, clusterID int64, clusterName string) ListWatcher {
	return &NodeListWatcher{
		client:       client,
		clusterID:    clusterID,
		clusterName:  clusterName,
		pollInterval: 5 * time.Second,
	}
}

func (w *NodeListWatcher) List(ctx context.Context) ([]interface{}, error) {
	var nodes []struct {
		Node   string `json:"node"`
		Status string `json:"status"`
		Uptime int64  `json:"uptime"`
		Maxcpu int    `json:"maxcpu"`
		Maxmem int64  `json:"maxmem"`
	}

	if err := w.client.Get(ctx, "/nodes", &nodes); err != nil {
		return nil, err
	}

	// 从 /cluster/status 获取节点 IP 地址映射
	nodeIPMap := make(map[string]string)
	clusterStatus, err := w.client.GetClusterStatus(ctx)
	if err == nil {
		// 解析集群状态，提取节点 IP 信息
		for _, item := range clusterStatus {
			if name, ok := item["name"].(string); ok {
				if nodeType, ok := item["type"].(string); ok && nodeType == "node" {
					if ip, ok := item["ip"].(string); ok && ip != "" {
						nodeIPMap[name] = ip
					}
				}
			}
		}
	}

	result := make([]interface{}, 0, len(nodes))
	for _, n := range nodes {
		// 从集群状态中获取 IP 地址
		ipAddress := nodeIPMap[n.Node]

		node := &model.PveNode{
			NodeName:  n.Node,
			IPAddress: ipAddress,
			ClusterID: w.clusterID,
			Status:    n.Status,
			Env:       "", // 可以从集群配置获取
		}
		result = append(result, node)
	}

	return result, nil
}

func (w *NodeListWatcher) Watch(ctx context.Context, version string) (string, []interface{}, error) {
	items, err := w.List(ctx)
	if err != nil {
		return version, nil, err
	}

	// 计算当前版本（使用内容的 MD5）
	newVersion := w.calculateVersion(items)
	if newVersion == version {
		// 没有变化，等待后返回空
		time.Sleep(w.pollInterval)
		return version, nil, nil
	}

	return newVersion, items, nil
}

func (w *NodeListWatcher) GetResourceVersion(obj interface{}) string {
	return w.calculateVersion([]interface{}{obj})
}

func (w *NodeListWatcher) calculateVersion(items []interface{}) string {
	data, _ := json.Marshal(items)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// VMListWatcher VM 列表监听器
type VMListWatcher struct {
	client       *proxmox.ProxmoxClient
	clusterID    int64
	clusterName  string
	nodeName     string
	pollInterval time.Duration
}

func NewVMListWatcher(client *proxmox.ProxmoxClient, clusterID int64, clusterName, nodeName string) ListWatcher {
	return &VMListWatcher{
		client:       client,
		clusterID:    clusterID,
		clusterName:  clusterName,
		nodeName:     nodeName,
		pollInterval: 5 * time.Second,
	}
}

// isStableVMStatus 判断虚拟机状态是否为稳定状态（可以安全上报）
// 过滤掉临时状态：locked, migrating, cloning, creating 等
func isStableVMStatus(status string) bool {
	// Proxmox VM 状态：
	// - running: 运行中（稳定）
	// - stopped: 已停止（稳定）
	// - paused: 已暂停（稳定）
	// - locked: 锁定中（临时，如克隆、迁移、备份等操作中）
	// - migrating: 迁移中（临时）
	// - creating: 创建中（临时）
	// - unknown: 未知状态（可能临时）

	// 过滤掉临时状态
	unstableStatuses := map[string]bool{
		"locked":    true, // 锁定状态（克隆、迁移、备份等操作中）
		"migrating": true, // 迁移中
		"creating":  true, // 创建中
		"unknown":   true, // 未知状态
	}

	// 如果状态为空，也认为是临时状态
	if status == "" {
		return false
	}

	// 检查是否为不稳定状态
	if unstableStatuses[status] {
		return false
	}

	// 其他状态（running, stopped, paused 等）认为是稳定状态
	return true
}

func (w *VMListWatcher) List(ctx context.Context) ([]interface{}, error) {
	var vms []struct {
		Vmid     int    `json:"vmid"`
		Name     string `json:"name"`
		Status   string `json:"status"`
		Uptime   int64  `json:"uptime"`
		Maxmem   int64  `json:"maxmem"`
		Cpus     int    `json:"cpus"`
		Template *int   `json:"template,omitempty"` // 模板标识：0=否, 1=是（如果字段存在）
	}

	path := fmt.Sprintf("/nodes/%s/qemu", w.nodeName)
	if err := w.client.Get(ctx, path, &vms); err != nil {
		return nil, err
	}

	result := make([]interface{}, 0, len(vms))
	for _, v := range vms {
		// 过滤掉临时状态的虚拟机，避免在克隆、迁移等操作期间重复上报
		if !isStableVMStatus(v.Status) {
			// 跳过临时状态的虚拟机，不进行上报
			continue
		}

		// 过滤掉空虚拟机（名称为空且资源为0，可能是克隆过程中的临时状态）
		if v.Name == "" && v.Maxmem == 0 && v.Cpus == 0 {
			// 跳过空虚拟机，可能是克隆过程中的临时状态
			continue
		}

		// 过滤掉模板同步过程中的临时虚拟机（名称以 "sync-" 开头）
		// 这些是临时创建的虚拟机，会在迁移后转换为模板，不应该作为普通虚拟机上报
		if strings.HasPrefix(v.Name, "sync-") {
			// 跳过临时同步虚拟机，避免在同步过程中产生脏数据
			continue
		}

		// 检查是否为模板
		isTemplate := int8(0)
		if v.Template != nil && *v.Template == 1 {
			isTemplate = 1
		}

		vm := &model.PveVM{
			VMID:        uint32(v.Vmid),
			VmName:      v.Name,
			NodeName:    w.nodeName,
			ClusterName: w.clusterName,
			Status:      v.Status,
			CPUNum:      v.Cpus,
			MemorySize:  int(v.Maxmem / 1024 / 1024), // 转换为 MB
			IsTemplate:  isTemplate,
		}

		result = append(result, vm)
	}

	return result, nil
}

func (w *VMListWatcher) Watch(ctx context.Context, version string) (string, []interface{}, error) {
	items, err := w.List(ctx)
	if err != nil {
		return version, nil, err
	}

	newVersion := w.calculateVersion(items)
	if newVersion == version {
		time.Sleep(w.pollInterval)
		return version, nil, nil
	}

	return newVersion, items, nil
}

func (w *VMListWatcher) GetResourceVersion(obj interface{}) string {
	return w.calculateVersion([]interface{}{obj})
}

func (w *VMListWatcher) calculateVersion(items []interface{}) string {
	data, _ := json.Marshal(items)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// StorageListWatcher 存储列表监听器
type StorageListWatcher struct {
	client       *proxmox.ProxmoxClient
	clusterID    int64
	nodeName     string
	pollInterval time.Duration
}

func NewStorageListWatcher(client *proxmox.ProxmoxClient, clusterID int64, nodeName string) ListWatcher {
	return &StorageListWatcher{
		client:       client,
		clusterID:    clusterID,
		nodeName:     nodeName,
		pollInterval: 5 * time.Second,
	}
}

func (w *StorageListWatcher) List(ctx context.Context) ([]interface{}, error) {
	var storages []struct {
		Storage      string  `json:"storage"`
		Type         string  `json:"type"`
		Content      string  `json:"content"`
		Shared       int     `json:"shared"`
		Active       int     `json:"active"`
		Enabled      int     `json:"enabled"`
		Avail        int64   `json:"avail"`
		Used         int64   `json:"used"`
		Total        int64   `json:"total"`
		UsedFraction float64 `json:"used_fraction"`
	}

	path := fmt.Sprintf("/nodes/%s/storage", w.nodeName)
	if err := w.client.Get(ctx, path, &storages); err != nil {
		return nil, err
	}

	result := make([]interface{}, 0, len(storages))
	for _, s := range storages {
		storage := &model.PveStorage{
			NodeName:     w.nodeName,
			ClusterID:    w.clusterID,
			StorageName:  s.Storage,
			Type:         s.Type,
			Content:      s.Content,
			Shared:       s.Shared,
			Active:       s.Active,
			Enabled:      s.Enabled,
			Avail:        s.Avail,
			Used:         s.Used,
			Total:        s.Total,
			UsedFraction: s.UsedFraction,
		}
		result = append(result, storage)
	}

	return result, nil
}

func (w *StorageListWatcher) Watch(ctx context.Context, version string) (string, []interface{}, error) {
	items, err := w.List(ctx)
	if err != nil {
		return version, nil, err
	}

	newVersion := w.calculateVersion(items)
	if newVersion == version {
		time.Sleep(w.pollInterval)
		return version, nil, nil
	}

	return newVersion, items, nil
}

func (w *StorageListWatcher) GetResourceVersion(obj interface{}) string {
	return w.calculateVersion([]interface{}{obj})
}

func (w *StorageListWatcher) calculateVersion(items []interface{}) string {
	data, _ := json.Marshal(items)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}
