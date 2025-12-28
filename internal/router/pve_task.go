package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitPveTaskRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// Strict permission routing group
	strictAuthRouter := r.Group("/tasks").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		strictAuthRouter.GET("/cluster", deps.PveTaskHandler.ListClusterTasks)
		strictAuthRouter.GET("/node", deps.PveTaskHandler.ListNodeTasks)
		strictAuthRouter.GET("/log", deps.PveTaskHandler.GetTaskLog)
		strictAuthRouter.GET("/status", deps.PveTaskHandler.GetTaskStatus)
		strictAuthRouter.DELETE("/stop", deps.PveTaskHandler.StopTask)
	}
}
