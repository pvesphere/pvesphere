package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitPveNodeRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// Console WebSocket 需要同域连接，浏览器 WebSocket 无法方便地携带 Authorization header，
	// 因此这里采用 /api/v1/nodes/console 返回的短期 ws_token 鉴权，不走 StrictAuth。
	r.Group("/nodes").GET("/console/ws", deps.PveNodeHandler.NodeConsoleWS)

	// Strict permission routing group
	strictAuthRouter := r.Group("/nodes").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		strictAuthRouter.GET("", deps.PveNodeHandler.ListNodes)
		// 状态和服务接口必须在 /:id 之前定义，避免路由冲突
		strictAuthRouter.GET("/status", deps.PveNodeHandler.GetNodeStatus)
		strictAuthRouter.POST("/status", deps.PveNodeHandler.SetNodeStatus)
		strictAuthRouter.GET("/services", deps.PveNodeHandler.GetNodeServices)
		strictAuthRouter.POST("/services/start", deps.PveNodeHandler.StartNodeService)
		strictAuthRouter.POST("/services/stop", deps.PveNodeHandler.StopNodeService)
		strictAuthRouter.POST("/services/restart", deps.PveNodeHandler.RestartNodeService)
		// 网络管理路由必须在 /:id 之前定义，避免路由冲突
		strictAuthRouter.GET("/network", deps.PveNodeHandler.GetNodeNetworks)
		strictAuthRouter.POST("/network", deps.PveNodeHandler.CreateNodeNetwork)
		strictAuthRouter.PUT("/network", deps.PveNodeHandler.ReloadNodeNetwork)
		strictAuthRouter.DELETE("/network", deps.PveNodeHandler.RevertNodeNetwork)
		strictAuthRouter.GET("/rrd", deps.PveNodeHandler.GetNodeRRDData)
		// 控制台相关路由必须在 /:id 之前定义，避免路由冲突
		strictAuthRouter.POST("/console", deps.PveNodeHandler.GetNodeConsole)

		// 存储相关路由必须在 /:id 之前定义，避免路由冲突
		strictAuthRouter.GET("/storage/status", deps.PveNodeHandler.GetNodeStorageStatus)
		strictAuthRouter.GET("/storage/rrd", deps.PveNodeHandler.GetNodeStorageRRDData)
		strictAuthRouter.GET("/storage/content", deps.PveNodeHandler.GetNodeStorageContent)
		strictAuthRouter.GET("/storage/content/detail", deps.PveNodeHandler.GetNodeStorageVolume)
		strictAuthRouter.POST("/storage/upload", deps.PveNodeHandler.UploadNodeStorageContent)
		strictAuthRouter.DELETE("/storage/content", deps.PveNodeHandler.DeleteNodeStorageContent)

		// 磁盘管理路由必须在 /:id 之前定义，避免路由冲突
		strictAuthRouter.GET("/disks/list", deps.PveNodeHandler.GetNodeDisksList)
		strictAuthRouter.GET("/disks/directory", deps.PveNodeHandler.GetNodeDisksDirectory)
		strictAuthRouter.GET("/disks/lvm", deps.PveNodeHandler.GetNodeDisksLVM)
		strictAuthRouter.GET("/disks/lvmthin", deps.PveNodeHandler.GetNodeDisksLVMThin)
		strictAuthRouter.GET("/disks/zfs", deps.PveNodeHandler.GetNodeDisksZFS)
		strictAuthRouter.POST("/disks/initgpt", deps.PveNodeHandler.InitGPTDisk)
		strictAuthRouter.PUT("/disks/wipedisk", deps.PveNodeHandler.WipeDisk)

		strictAuthRouter.GET("/:id", deps.PveNodeHandler.GetNode)
		strictAuthRouter.POST("", deps.PveNodeHandler.CreateNode)
		strictAuthRouter.PUT("/:id", deps.PveNodeHandler.UpdateNode)
		strictAuthRouter.DELETE("/:id", deps.PveNodeHandler.DeleteNode)
	}
}
