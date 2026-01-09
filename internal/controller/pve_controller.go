package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pvesphere/internal/controller/informer"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"
	"pvesphere/pkg/proxmox"

	"go.uber.org/zap"
)

type PveController struct {
	clusterRepo  repository.PveClusterRepository
	nodeRepo     repository.PveNodeRepository
	vmRepo       repository.PveVMRepository
	storageRepo  repository.PveStorageRepository
	logger       *log.Logger
	informers    map[int64]*ClusterInformer
	lock         sync.RWMutex
	resyncPeriod time.Duration
}

type ClusterInformer struct {
	Cluster          *model.PveCluster
	Client           *proxmox.ProxmoxClient
	NodeInformer     informer.Informer
	VMInformers      map[string]informer.Informer
	StorageInformers map[string]informer.Informer
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewPveController(
	clusterRepo repository.PveClusterRepository,
	nodeRepo repository.PveNodeRepository,
	vmRepo repository.PveVMRepository,
	storageRepo repository.PveStorageRepository,
	logger *log.Logger,
	resyncPeriod time.Duration,
) *PveController {
	return &PveController{
		clusterRepo:  clusterRepo,
		nodeRepo:     nodeRepo,
		vmRepo:       vmRepo,
		storageRepo:  storageRepo,
		logger:       logger,
		informers:    make(map[int64]*ClusterInformer),
		resyncPeriod: resyncPeriod,
	}
}

func (c *PveController) Start(ctx context.Context) error {
	c.logger.Info("starting PVE controller")

	// 加载所有启用的集群（is_enabled = 1 用于数据自动上报）
	clusters, err := c.clusterRepo.GetAllEnabled(ctx)
	if err != nil {
		return err
	}

	for _, cluster := range clusters {
		if err := c.startClusterInformer(ctx, cluster); err != nil {
			c.logger.Error("failed to start cluster informer", zap.Error(err), zap.String("cluster", cluster.ClusterName))
			continue
		}
	}

	// 定期检查新集群
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			c.syncClusters(ctx)
		}
	}
}

func (c *PveController) Stop(ctx context.Context) error {
	c.logger.Info("stopping PVE controller")

	c.lock.Lock()
	defer c.lock.Unlock()

	for _, inf := range c.informers {
		inf.cancel()
		if inf.NodeInformer != nil {
			inf.NodeInformer.Stop()
		}
		for _, vmInf := range inf.VMInformers {
			vmInf.Stop()
		}
		for _, stInf := range inf.StorageInformers {
			stInf.Stop()
		}
	}

	c.informers = make(map[int64]*ClusterInformer)
	return nil
}

func (c *PveController) syncClusters(ctx context.Context) {
	// 加载所有启用的集群（is_enabled = 1 用于数据自动上报）
	clusters, err := c.clusterRepo.GetAllEnabled(ctx)
	if err != nil {
		c.logger.Error("failed to list enabled clusters", zap.Error(err))
		return
	}

	clusterMap := make(map[int64]*model.PveCluster)
	for _, cluster := range clusters {
		clusterMap[cluster.Id] = cluster
	}

	c.lock.Lock()

	// 启动新启用的集群 informer（在锁内检查，在锁外启动避免阻塞）
	var toStart []*model.PveCluster
	for _, cluster := range clusters {
		if _, exists := c.informers[cluster.Id]; !exists {
			toStart = append(toStart, cluster)
		}
	}

	// 停止不再启用的集群 informer（收集需要停止的，在锁外执行）
	var toStop []*ClusterInformer
	var toStopIDs []int64
	for id, inf := range c.informers {
		if _, exists := clusterMap[id]; !exists {
			c.logger.Info("stopping cluster informer (disabled)", zap.Int64("cluster_id", id), zap.String("cluster", inf.Cluster.ClusterName))
			toStopIDs = append(toStopIDs, id)
			toStop = append(toStop, inf)
			// 先从 map 中删除，避免重复操作
			delete(c.informers, id)
		}
	}
	c.lock.Unlock()

	// 在锁外启动新集群的 informer
	for _, cluster := range toStart {
		if err := c.startClusterInformer(ctx, cluster); err != nil {
			c.logger.Error("failed to start cluster informer", zap.Error(err), zap.String("cluster", cluster.ClusterName))
		}
	}

	// 在锁外停止 informer 并取消 context
	for i, id := range toStopIDs {
		inf := toStop[i]
		inf.cancel()
		if inf.NodeInformer != nil {
			inf.NodeInformer.Stop()
		}
		for _, vmInf := range inf.VMInformers {
			vmInf.Stop()
		}
		for _, stInf := range inf.StorageInformers {
			stInf.Stop()
		}
		c.logger.Debug("cluster informer stopped", zap.Int64("cluster_id", id))
	}
}

func (c *PveController) startClusterInformer(ctx context.Context, cluster *model.PveCluster) error {
	// 先检查是否已存在（避免重复启动）
	c.lock.RLock()
	if _, exists := c.informers[cluster.Id]; exists {
		c.lock.RUnlock()
		return nil
	}
	c.lock.RUnlock()

	// 在锁外创建客户端和 context（避免阻塞）
	client, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		return fmt.Errorf("failed to create proxmox client: %w", err)
	}

	clusterCtx, cancel := context.WithCancel(ctx)
	inf := &ClusterInformer{
		Cluster:          cluster,
		Client:           client,
		VMInformers:      make(map[string]informer.Informer),
		StorageInformers: make(map[string]informer.Informer),
		ctx:              clusterCtx,
		cancel:           cancel,
	}

	// 启动节点 informer
	if err := c.startNodeInformer(inf); err != nil {
		cancel()
		return fmt.Errorf("failed to start node informer: %w", err)
	}

	// 再次检查并添加到 map（双重检查，避免并发问题）
	c.lock.Lock()
	if _, exists := c.informers[cluster.Id]; exists {
		c.lock.Unlock()
		// 如果已经存在，清理刚创建的资源
		cancel()
		return nil
	}
	c.informers[cluster.Id] = inf
	c.lock.Unlock()

	c.logger.Info("cluster informer started", zap.String("cluster", cluster.ClusterName), zap.Int64("id", cluster.Id))

	return nil
}

func (c *PveController) startNodeInformer(inf *ClusterInformer) error {
	nodeKeyFunc := func(obj interface{}) (string, error) {
		node := obj.(*model.PveNode)
		return fmt.Sprintf("%s-%d", node.NodeName, inf.Cluster.Id), nil
	}

	nodeWatcher := informer.NewNodeListWatcher(inf.Client, inf.Cluster.Id, inf.Cluster.ClusterName)
	nodeInf := informer.NewInformer(
		"node-"+inf.Cluster.ClusterName,
		nodeWatcher,
		nodeKeyFunc,
		c.logger,
		c.resyncPeriod,
	)

	nodeHandler := NewNodeEventHandler(c.nodeRepo, c.logger, inf.Cluster.Id, inf.Cluster.Env)
	nodeInf.AddEventHandler(nodeHandler)

	inf.NodeInformer = nodeInf
	nodeInf.Run(inf.ctx)

	// 等待节点列表同步后，启动每个节点的 VM 和 Storage informer
	go func() {
		// 等待节点列表同步，同时检查 context 是否已取消
		select {
		case <-inf.ctx.Done():
			// 如果集群已被禁用，直接返回，不启动 VM 和 Storage informers
			c.logger.Debug("cluster context cancelled, skipping VM/Storage informers startup",
				zap.Int64("cluster_id", inf.Cluster.Id),
				zap.String("cluster_name", inf.Cluster.ClusterName))
			return
		case <-time.After(5 * time.Second):
			// 5 秒后检查 context 是否已取消
			if inf.ctx.Err() != nil {
				c.logger.Debug("cluster context cancelled during wait, skipping VM/Storage informers startup",
					zap.Int64("cluster_id", inf.Cluster.Id),
					zap.String("cluster_name", inf.Cluster.ClusterName))
				return
			}
			// 启动 VM 和 Storage informers
			c.startVMAndStorageInformers(inf)
		}
	}()

	return nil
}

func (c *PveController) startVMAndStorageInformers(inf *ClusterInformer) {
	// 检查 context 是否已取消（集群可能已被禁用）
	if inf.ctx.Err() != nil {
		c.logger.Debug("cluster context cancelled, skipping VM/Storage informers startup",
			zap.Int64("cluster_id", inf.Cluster.Id),
			zap.String("cluster_name", inf.Cluster.ClusterName))
		return
	}

	// 使用集群的 context 而不是 Background，这样可以在集群被禁用时取消操作
	nodes, err := c.nodeRepo.GetByClusterID(inf.ctx, inf.Cluster.Id)
	if err != nil {
		c.logger.Error("failed to get nodes", zap.Error(err))
		return
	}

	for _, node := range nodes {
		// 在启动每个 informer 之前检查 context 是否已取消
		if inf.ctx.Err() != nil {
			c.logger.Debug("cluster context cancelled during VM/Storage informers startup",
				zap.Int64("cluster_id", inf.Cluster.Id),
				zap.String("cluster_name", inf.Cluster.ClusterName))
			return
		}
		// 启动 VM informer
		vmKeyFunc := func(obj interface{}) (string, error) {
			vm := obj.(*model.PveVM)
			return fmt.Sprintf("%s-%d-%d", vm.NodeName, vm.VMID, inf.Cluster.Id), nil
		}

		vmWatcher := informer.NewVMListWatcher(inf.Client, inf.Cluster.Id, inf.Cluster.ClusterName, node.NodeName)
		vmInf := informer.NewInformer(
			fmt.Sprintf("vm-%s-%s", inf.Cluster.ClusterName, node.NodeName),
			vmWatcher,
			vmKeyFunc,
			c.logger,
			c.resyncPeriod,
		)

		vmHandler := NewVMEventHandler(c.vmRepo, c.nodeRepo, c.logger, inf.Cluster.Id, inf.Cluster.ClusterName)
		vmInf.AddEventHandler(vmHandler)

		inf.VMInformers[node.NodeName] = vmInf
		vmInf.Run(inf.ctx)

		// 启动 Storage informer
		storageKeyFunc := func(obj interface{}) (string, error) {
			storage := obj.(*model.PveStorage)
			return fmt.Sprintf("%s-%s-%d", storage.NodeName, storage.StorageName, inf.Cluster.Id), nil
		}

		storageWatcher := informer.NewStorageListWatcher(inf.Client, inf.Cluster.Id, node.NodeName)
		storageInf := informer.NewInformer(
			fmt.Sprintf("storage-%s-%s", inf.Cluster.ClusterName, node.NodeName),
			storageWatcher,
			storageKeyFunc,
			c.logger,
			c.resyncPeriod,
		)

		storageHandler := NewStorageEventHandler(c.storageRepo, c.logger, inf.Cluster.Id)
		storageInf.AddEventHandler(storageHandler)

		inf.StorageInformers[node.NodeName] = storageInf
		storageInf.Run(inf.ctx)
	}
}

func (c *PveController) stopClusterInformer(clusterID int64) {
	inf, exists := c.informers[clusterID]
	if !exists {
		return
	}

	if inf.NodeInformer != nil {
		inf.NodeInformer.Stop()
	}
	for _, vmInf := range inf.VMInformers {
		vmInf.Stop()
	}
	for _, stInf := range inf.StorageInformers {
		stInf.Stop()
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.informers, clusterID)
}
