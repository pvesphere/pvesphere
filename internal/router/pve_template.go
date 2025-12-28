package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitPveTemplateRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// Strict permission routing group
	strictAuthRouter := r.Group("/templates").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		// 基础模板 CRUD
		strictAuthRouter.GET("", deps.PveTemplateHandler.ListTemplates)
		strictAuthRouter.GET("/:id", deps.PveTemplateHandler.GetTemplate)
		strictAuthRouter.POST("", deps.PveTemplateHandler.CreateTemplate)
		strictAuthRouter.PUT("/:id", deps.PveTemplateHandler.UpdateTemplate)
		strictAuthRouter.DELETE("/:id", deps.PveTemplateHandler.DeleteTemplate)
	}
}

func InitTemplateManagementRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// 模板管理路由（导入、同步等高级功能）
	strictAuthRouter := r.Group("/templates").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		// 模板导入（从备份文件）
		strictAuthRouter.POST("/import", deps.TemplateManagementHandler.ImportTemplate)
		
		// 模板详情（包含实例）
		strictAuthRouter.GET("/:id/detail", deps.TemplateManagementHandler.GetTemplateDetail)
		
		// 模板同步
		strictAuthRouter.POST("/:id/sync", deps.TemplateManagementHandler.SyncTemplate)
		
		// 模板实例
		strictAuthRouter.GET("/:id/instances", deps.TemplateManagementHandler.ListTemplateInstances)
	}
	
	// 同步任务路由
	syncTaskRouter := r.Group("/templates/sync-tasks").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		syncTaskRouter.GET("", deps.TemplateManagementHandler.ListSyncTasks)
		syncTaskRouter.GET("/:task_id", deps.TemplateManagementHandler.GetSyncTask)
		syncTaskRouter.POST("/:task_id/retry", deps.TemplateManagementHandler.RetrySyncTask)
	}
}
