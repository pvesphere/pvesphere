package router

import (
	"pvesphere/internal/handler"
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitNodeMaintenanceRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	maintenanceHandler := handler.NewNodeMaintenanceHandler(
		handler.NewHandler(deps.Logger),
		deps.NodeMaintenanceService,
	)

	// 节点维护相关路由
	maintenanceRouter := r.Group("/nodes/maintenance").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		// 获取可迁移的目标节点
		maintenanceRouter.GET("/candidates", maintenanceHandler.GetMigrateCandidates)
		// 获取节点上的VM列表
		maintenanceRouter.GET("/vms", maintenanceHandler.GetNodeVMs)
		// 开始维护
		maintenanceRouter.POST("/start", maintenanceHandler.StartMaintenance)
		// 结束维护
		maintenanceRouter.POST("/end", maintenanceHandler.EndMaintenance)
	}
}
