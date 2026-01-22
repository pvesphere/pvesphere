package server

import (
	"context"
	"os"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"
	"pvesphere/pkg/sid"

	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MigrateServer struct {
	db         *gorm.DB
	log        *log.Logger
	userRepo   repository.UserRepository
	sid        *sid.Sid
}

func NewMigrateServer(db *gorm.DB, log *log.Logger, userRepo repository.UserRepository, sid *sid.Sid) *MigrateServer {
	return &MigrateServer{
		db:       db,
		log:      log,
		userRepo: userRepo,
		sid:      sid,
	}
}
func (m *MigrateServer) Start(ctx context.Context) error {
	if err := m.db.AutoMigrate(
		&model.User{},
		// PVE 相关表
		&model.PveCluster{},
		&model.PveNode{},
		&model.PveVM{},
		&model.PveStorage{},
		&model.VMIPAddress{},
		&model.VmTemplate{},
		// 模板管理相关表
		&model.TemplateUpload{},
		&model.TemplateInstance{},
		&model.TemplateSyncTask{},
	); err != nil {
		m.log.Error("migrate error", zap.Error(err))
		return err
	}
	m.log.Info("AutoMigrate success")

	// 创建默认用户
	if err := m.createDefaultUser(ctx); err != nil {
		m.log.Error("create default user error", zap.Error(err))
		return err
	}

	os.Exit(0)
	return nil
}

// createDefaultUser 创建默认管理员用户
func (m *MigrateServer) createDefaultUser(ctx context.Context) error {
	defaultUsername := "admin"
	defaultEmail := "pvesphere@gmail.com"
	defaultPassword := "Ab123456"
	defaultNickname := "PveSphere Admin"

	// 检查用户是否已存在（通过邮箱或用户名）
	existingUser, err := m.userRepo.GetByEmail(ctx, defaultEmail)
	if err != nil {
		m.log.Error("check default user error", zap.Error(err))
		return err
	}
	if existingUser != nil {
		m.log.Info("default user already exists", zap.String("email", defaultEmail))
		return nil
	}

	// 检查用户名是否已存在
	existingUser, err = m.userRepo.GetByUsername(ctx, defaultUsername)
	if err != nil {
		m.log.Error("check default username error", zap.Error(err))
		return err
	}
	if existingUser != nil {
		m.log.Info("default username already exists", zap.String("username", defaultUsername))
		return nil
	}

	// 生成用户 ID
	userId, err := m.sid.GenString()
	if err != nil {
		m.log.Error("generate user id error", zap.Error(err))
		return err
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		m.log.Error("hash password error", zap.Error(err))
		return err
	}

	// 创建用户
	user := &model.User{
		UserId:   userId,
		Username: defaultUsername,
		Email:    defaultEmail,
		Password: string(hashedPassword),
		Nickname: defaultNickname,
	}

	if err := m.userRepo.Create(ctx, user); err != nil {
		m.log.Error("create default user error", zap.Error(err))
		return err
	}

	m.log.Info("default user created successfully", 
		zap.String("username", defaultUsername),
		zap.String("email", defaultEmail),
		zap.String("userId", userId),
		zap.String("nickname", defaultNickname))
	return nil
}
func (m *MigrateServer) Stop(ctx context.Context) error {
	m.log.Info("AutoMigrate stop")
	return nil
}
