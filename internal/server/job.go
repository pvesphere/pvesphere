package server

import (
	"context"
	"pvesphere/internal/job"
	"pvesphere/pkg/log"
)

type JobServer struct {
	log     *log.Logger
	userJob job.UserJob
}

func NewJobServer(
	log *log.Logger,
	userJob job.UserJob,
) *JobServer {
	return &JobServer{
		log:     log,
		userJob: userJob,
	}
}

func (j *JobServer) Start(ctx context.Context) error {
	// Tips: If you want job to start as a separate process, just refer to the task implementation and adjust the code accordingly.

	// eg: kafka consumer
	err := j.userJob.KafkaConsumer(ctx)
	return err
}
func (j *JobServer) Stop(ctx context.Context) error {
	return nil
}
