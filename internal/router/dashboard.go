package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

// InitDashboardRouter 配置 Dashboard 路由
func InitDashboardRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// Dashboard 路由组，使用严格鉴权
	dashboardRouter := r.Group("/dashboard").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		// 获取可选集群列表
		dashboardRouter.GET("/scopes", deps.DashboardHandler.GetScopes)

		// 获取全局概览
		dashboardRouter.GET("/overview", deps.DashboardHandler.GetOverview)

		// 获取资源使用率
		dashboardRouter.GET("/resources", deps.DashboardHandler.GetResources)

		// 获取压力和风险焦点
		dashboardRouter.GET("/hotspots", deps.DashboardHandler.GetHotspots)

		// 获取运行中的操作
		dashboardRouter.GET("/operations", deps.DashboardHandler.GetOperations)
	}
}

