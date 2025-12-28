package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitPveStorageRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// Strict permission routing group
	strictAuthRouter := r.Group("/storages").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		strictAuthRouter.GET("", deps.PveStorageHandler.ListStorages)
		strictAuthRouter.GET("/:id", deps.PveStorageHandler.GetStorage)
		strictAuthRouter.POST("", deps.PveStorageHandler.CreateStorage)
		strictAuthRouter.PUT("/:id", deps.PveStorageHandler.UpdateStorage)
		strictAuthRouter.DELETE("/:id", deps.PveStorageHandler.DeleteStorage)
	}
}
