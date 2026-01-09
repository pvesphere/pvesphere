//go:build wireinject
// +build wireinject

package wire

import (
	"time"

	"pvesphere/internal/controller"
	"pvesphere/internal/repository"
	"pvesphere/internal/server"
	"pvesphere/pkg/app"
	"pvesphere/pkg/log"
	"github.com/google/wire"
	"github.com/spf13/viper"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
	repository.NewRepository,
	repository.NewTransaction,
	repository.NewPveClusterRepository,
	repository.NewPveNodeRepository,
	repository.NewPveVMRepository,
	repository.NewPveStorageRepository,
	repository.NewVMIPAddressRepository,
	repository.NewVmTemplateRepository,
)

var controllerSet = wire.NewSet(
	controller.NewPveController,
)

var serverSet = wire.NewSet(
	server.NewControllerServer,
)

func newApp(
	controllerServer *server.ControllerServer,
) *app.App {
	return app.NewApp(
		app.WithServer(controllerServer),
		app.WithName("pve-controller"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		controllerSet,
		serverSet,
		newApp,
		wire.Value(time.Minute*5), // resyncPeriod
	))
}

