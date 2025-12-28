package router

import (
	"pvesphere/internal/handler"
	"pvesphere/pkg/jwt"
	"pvesphere/pkg/log"

	"github.com/spf13/viper"
)

type RouterDeps struct {
	Logger                     *log.Logger
	Config                     *viper.Viper
	JWT                        *jwt.JWT
	PveAuthHandler             *handler.PveAuthHandler
	UserHandler                *handler.UserHandler
	PveClusterHandler          *handler.PveClusterHandler
	PveNodeHandler             *handler.PveNodeHandler
	PveVMHandler               *handler.PveVMHandler
	PveStorageHandler          *handler.PveStorageHandler
	PveTemplateHandler         *handler.PveTemplateHandler
	TemplateManagementHandler  *handler.TemplateManagementHandler
	PveTaskHandler             *handler.PveTaskHandler
	DashboardHandler           *handler.DashboardHandler
}
