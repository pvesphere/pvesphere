package server

import (
	"context"
	"pvesphere/internal/controller"
	"pvesphere/pkg/log"
)

type ControllerServer struct {
	controller *controller.PveController
	log        *log.Logger
}

func NewControllerServer(
	log *log.Logger,
	pveController *controller.PveController,
) *ControllerServer {
	return &ControllerServer{
		controller: pveController,
		log:        log,
	}
}

func (s *ControllerServer) Start(ctx context.Context) error {
	s.log.Info("starting controller server")
	return s.controller.Start(ctx)
}

func (s *ControllerServer) Stop(ctx context.Context) error {
	s.log.Info("stopping controller server")
	return s.controller.Stop(ctx)
}
