package main

import (
	"context"
	"flag"
	"fmt"

	"pvesphere/cmd/server/wire"
	"pvesphere/pkg/config"
	"pvesphere/pkg/log"

	"go.uber.org/zap"
)

// @title           PveSphere API
// @version         1.0.0
// @description     PveSphere is a comprehensive web-based management platform for Proxmox VE (PVE) clusters.
// @termsOfService  http://swagger.io/terms/
// @contact.name   PveSphere Support
// @contact.url    https://github.com/pvesphere/pvesphere
// @contact.email  support@pvesphere.io
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
// @host      localhost:8000
// @securityDefinitions.apiKey Bearer
// @in header
// @name Authorization
// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	var envConf = flag.String("conf", "config/local.yml", "config path, eg: -conf ./config/local.yml")
	flag.Parse()
	conf := config.NewConfig(*envConf)

	logger := log.NewLog(conf)

	app, cleanup, err := wire.NewWire(conf, logger)
	defer cleanup()
	if err != nil {
		panic(err)
	}
	logger.Info("server start", zap.String("host", fmt.Sprintf("http://%s:%d", conf.GetString("http.host"), conf.GetInt("http.port"))))
	logger.Info("docs addr", zap.String("addr", fmt.Sprintf("http://%s:%d/swagger/index.html", conf.GetString("http.host"), conf.GetInt("http.port"))))
	if err = app.Run(context.Background()); err != nil {
		panic(err)
	}
}
