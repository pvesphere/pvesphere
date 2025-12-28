//go:build wireinject
// +build wireinject

package wire

import (
	"pvesphere/internal/handler"
	"pvesphere/internal/job"
	"pvesphere/internal/repository"
	"pvesphere/internal/router"
	"pvesphere/internal/server"
	"pvesphere/internal/service"
	"pvesphere/pkg/app"
	"pvesphere/pkg/jwt"
	"pvesphere/pkg/log"
	"pvesphere/pkg/server/http"
	"pvesphere/pkg/sid"

	"github.com/google/wire"
	"github.com/spf13/viper"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
	//repository.NewRedis,
	//repository.NewMongo,
	repository.NewRepository,
	repository.NewTransaction,
	repository.NewUserRepository,
	repository.NewPveClusterRepository,
	repository.NewPveNodeRepository,
	repository.NewPveVMRepository,
	repository.NewPveStorageRepository,
	repository.NewVmTemplateRepository,
	repository.NewVMIPAddressRepository,
	repository.NewPveTemplateRepository,
	repository.NewTemplateUploadRepository,
	repository.NewTemplateInstanceRepository,
	repository.NewTemplateSyncTaskRepository,
)

var serviceSet = wire.NewSet(
	service.NewService,
	service.NewUserService,
	service.NewPveClusterService,
	service.NewPveNodeService,
	service.NewPveVMService,
	service.NewPveStorageService,
	service.NewPveTemplateService,
	service.NewPveTaskService,
	service.NewDashboardService,
	service.NewTemplateManagementService,
)

var handlerSet = wire.NewSet(
	handler.NewHandler,
	handler.NewPveAuthHandler,
	handler.NewUserHandler,
	handler.NewPveClusterHandler,
	handler.NewPveNodeHandler,
	handler.NewPveVMHandler,
	handler.NewPveStorageHandler,
	handler.NewPveTemplateHandler,
	handler.NewTemplateManagementHandler,
	handler.NewPveTaskHandler,
	handler.NewDashboardHandler,
)

var jobSet = wire.NewSet(
	job.NewJob,
	job.NewUserJob,
)
var serverSet = wire.NewSet(
	server.NewHTTPServer,
	server.NewJobServer,
)

// build App
func newApp(
	httpServer *http.Server,
	jobServer *server.JobServer,
	// task *server.Task,
) *app.App {
	return app.NewApp(
		app.WithServer(httpServer, jobServer),
		app.WithName("demo-server"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		serviceSet,
		handlerSet,
		jobSet,
		serverSet,
		wire.Struct(new(router.RouterDeps), "*"),
		sid.NewSid,
		jwt.NewJwt,
		newApp,
	))
}
