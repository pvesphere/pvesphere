package task

import (
	"context"
	"pvesphere/internal/repository"
)

type UserTask interface {
	CheckUser(ctx context.Context) error
}

func NewUserTask(
	task *Task,
	userRepo repository.UserRepository,
) UserTask {
	return &userTask{
		userRepo: userRepo,
		Task:     task,
	}
}

type userTask struct {
	userRepo repository.UserRepository
	*Task
}

func (t userTask) CheckUser(ctx context.Context) error {
	// do something
	t.logger.Info("CheckUser")
	return nil
}
