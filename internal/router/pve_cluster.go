package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitPveClusterRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// Strict permission routing group
	strictAuthRouter := r.Group("/clusters").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		strictAuthRouter.GET("", deps.PveClusterHandler.ListClusters)
		// 状态、资源和验证接口必须在 /:id 之前定义，避免路由冲突
		strictAuthRouter.GET("/status", deps.PveClusterHandler.GetClusterStatus)
		strictAuthRouter.GET("/resources", deps.PveClusterHandler.GetClusterResources)
		strictAuthRouter.GET("/verify", deps.PveClusterHandler.VerifyCluster)
		strictAuthRouter.GET("/:id", deps.PveClusterHandler.GetCluster)
		strictAuthRouter.POST("", deps.PveClusterHandler.CreateCluster)
		strictAuthRouter.PUT("/:id", deps.PveClusterHandler.UpdateCluster)
		strictAuthRouter.DELETE("/:id", deps.PveClusterHandler.DeleteCluster)
	}
}
