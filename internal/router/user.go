package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitUserRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// No route group has permission
	noAuthRouter := r.Group("/")
	{
		noAuthRouter.POST("/register", deps.UserHandler.Register)
		noAuthRouter.POST("/login", deps.UserHandler.Login)
	}

	// Strict permission routing group (requires authentication)
	strictAuthRouter := r.Group("/").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		strictAuthRouter.GET("/user", deps.UserHandler.GetProfile)
		strictAuthRouter.PUT("/user", deps.UserHandler.UpdateProfile)
	}
}
