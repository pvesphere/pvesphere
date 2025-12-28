package router

import (
	"pvesphere/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitPveAuthRouter(
	deps RouterDeps,
	r *gin.RouterGroup,
) {
	// 高权限票据接口仍然走严格鉴权，避免被未授权客户端直接获取 root 级 ticket。
	strictAuthRouter := r.Group("/pve").Use(middleware.StrictAuth(deps.JWT, deps.Logger))
	{
		strictAuthRouter.POST("/access/ticket", deps.PveAuthHandler.GetAccessTicket)
	}
}
