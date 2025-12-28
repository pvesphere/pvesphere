package server

import (
	apiV1 "pvesphere/api/v1"
	"pvesphere/docs"
	"pvesphere/internal/middleware"
	"pvesphere/internal/router"
	"pvesphere/pkg/server/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewHTTPServer(
	deps router.RouterDeps,
) *http.Server {
	if deps.Config.GetString("env") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	s := http.NewServer(
		gin.Default(),
		deps.Logger,
		http.WithServerHost(deps.Config.GetString("http.host")),
		http.WithServerPort(deps.Config.GetInt("http.port")),
	)

	// swagger doc
	docs.SwaggerInfo.BasePath = "/"
	s.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerfiles.Handler,
		//ginSwagger.URL(fmt.Sprintf("http://localhost:%d/swagger/doc.json", deps.Config.GetInt("app.http.port"))),
		ginSwagger.DefaultModelsExpandDepth(-1),
		ginSwagger.PersistAuthorization(true),
	))

	s.Use(
		middleware.CORSMiddleware(),
		middleware.ResponseLogMiddleware(deps.Logger),
		middleware.RequestLogMiddleware(deps.Logger),
		//middleware.SignMiddleware(log),
	)
	s.GET("/", func(ctx *gin.Context) {
		deps.Logger.WithContext(ctx).Info("hello")
		apiV1.HandleSuccess(ctx, map[string]interface{}{
			":)": "Thank you for using Pvesphere!",
		})
	})

	// v1 := s.Group("/v1")
	// router.InitUserRouter(deps, v1)

	apiV1 := s.Group("/api/v1")
	router.InitUserRouter(deps, apiV1)
	router.InitPveAuthRouter(deps, apiV1)
	router.InitPveClusterRouter(deps, apiV1)
	router.InitPveNodeRouter(deps, apiV1)
	router.InitPveVMRouter(deps, apiV1)
	router.InitPveStorageRouter(deps, apiV1)
	router.InitPveTemplateRouter(deps, apiV1)
	router.InitTemplateManagementRouter(deps, apiV1)
	router.InitPveTaskRouter(deps, apiV1)
	router.InitDashboardRouter(deps, apiV1)

	return s
}
