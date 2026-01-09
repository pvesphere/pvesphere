//go:build wireinject
// +build wireinject

package wire

import (
	"pvesphere/internal/repository"
	"pvesphere/internal/server"
	"pvesphere/pkg/app"
	"pvesphere/pkg/log"
	"pvesphere/pkg/sid"
	"github.com/google/wire"
	"github.com/spf13/viper"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
	//repository.NewRedis,
	repository.NewRepository,
	repository.NewUserRepository,
)
var serverSet = wire.NewSet(
	server.NewMigrateServer,
)
var sidSet = wire.NewSet(
	sid.NewSid,
)

// build App
func newApp(
	migrateServer *server.MigrateServer,
) *app.App {
	return app.NewApp(
		app.WithServer(migrateServer),
		app.WithName("demo-migrate"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		sidSet,
		serverSet,
		newApp,
	))
}
