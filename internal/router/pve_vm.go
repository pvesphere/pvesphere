package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitPveVMRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// Console WebSocket 需要同域连接，浏览器 WebSocket 无法方便地携带 Authorization header，
	// 因此这里采用 /api/v1/vms/console 返回的短期 ws_token 鉴权，不走 StrictAuth。
	r.Group("/vms").GET("/console/ws", deps.PveVMHandler.VMConsoleWS)

	// Strict permission routing group
	strictAuthRouter := r.Group("/vms").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		strictAuthRouter.GET("", deps.PveVMHandler.ListVMs)
		strictAuthRouter.POST("", deps.PveVMHandler.CreateVM) // 仅创建数据库记录
		// 注意：所有具体路径必须在 /:id 之前定义，避免路由冲突
		strictAuthRouter.POST("/create", deps.PveVMHandler.CreateVMInProxmox) // 完整创建流程
		strictAuthRouter.POST("/:id/start", deps.PveVMHandler.StartVM)
		strictAuthRouter.POST("/:id/stop", deps.PveVMHandler.StopVM)
		// 配置相关路由必须在 /:id 之前定义
		strictAuthRouter.GET("/config", deps.PveVMHandler.GetVMCurrentConfig)
		strictAuthRouter.GET("/config/pending", deps.PveVMHandler.GetVMPendingConfig)
		strictAuthRouter.PUT("/config", deps.PveVMHandler.UpdateVMConfig)
		strictAuthRouter.GET("/status", deps.PveVMHandler.GetVMStatus)
		strictAuthRouter.POST("/console", deps.PveVMHandler.GetVMConsole)
		strictAuthRouter.GET("/rrd", deps.PveVMHandler.GetVMRRDData)
		// 迁移相关路由必须在 /:id 之前定义
		strictAuthRouter.POST("/migrate", deps.PveVMHandler.MigrateVM)
		strictAuthRouter.POST("/remote-migrate", deps.PveVMHandler.RemoteMigrateVM)
		// 备份相关路由必须在 /:id 之前定义
		strictAuthRouter.POST("/backup", deps.PveVMHandler.CreateBackup)
		strictAuthRouter.DELETE("/backup", deps.PveVMHandler.DeleteBackup)
		// CloudInit 相关路由必须在 /:id 之前定义
		strictAuthRouter.GET("/cloudinit", deps.PveVMHandler.GetVMCloudInit)
		strictAuthRouter.PUT("/cloudinit", deps.PveVMHandler.UpdateVMCloudInit)
		strictAuthRouter.GET("/:id", deps.PveVMHandler.GetVM)
		strictAuthRouter.PUT("/:id", deps.PveVMHandler.UpdateVM)
		strictAuthRouter.DELETE("/:id", deps.PveVMHandler.DeleteVM)
	}
}
