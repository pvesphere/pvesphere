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
		// 获取节点上的 VM 列表
		maintenanceRouter.GET("/vms", maintenanceHandler.GetNodeVMs)
		// 开始维护（同步阻塞模式，保留向后兼容）
		maintenanceRouter.POST("/start", maintenanceHandler.StartMaintenance)
		// 结束维护
		maintenanceRouter.POST("/end", maintenanceHandler.EndMaintenance)
	}

	// 维护计划管理路由
	planRouter := r.Group("/nodes/maintenance/plans").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		// 创建维护计划（异步执行）
		planRouter.POST("", maintenanceHandler.CreateMaintenancePlan)
		// 获取维护计划详情
		planRouter.GET("/:plan_id", maintenanceHandler.GetMaintenancePlan)
		// 获取维护计划任务列表
		planRouter.GET("/:plan_id/tasks", maintenanceHandler.GetMaintenancePlanTasks)
		// 获取维护计划进度
		planRouter.GET("/:plan_id/progress", maintenanceHandler.GetMaintenancePlanProgress)
		// 取消维护计划
		planRouter.POST("/:plan_id/cancel", maintenanceHandler.CancelMaintenancePlan)
	}

	// 维护历史路由
	historyRouter := r.Group("/nodes/maintenance/history").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		// 获取维护历史
		historyRouter.GET("", maintenanceHandler.GetMaintenanceHistory)
	}
}
