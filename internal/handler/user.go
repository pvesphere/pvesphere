package handler

import (
	"net/http"
	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserHandler struct {
	*Handler
	userService service.UserService
}

func NewUserHandler(handler *Handler, userService service.UserService) *UserHandler {
	return &UserHandler{
		Handler:     handler,
		userService: userService,
	}
}

// Register godoc
// @Summary 用户注册
// @Schemes
// @Description 目前只支持邮箱登录
// @Tags 用户模块
// @Accept json
// @Produce json
// @Param request body v1.RegisterRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/register [post]
func (h *UserHandler) Register(ctx *gin.Context) {
	req := new(v1.RegisterRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.userService.Register(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("userService.Register error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// Login godoc
// @Summary 账号登录
// @Schemes
// @Description
// @Tags 用户模块
// @Accept json
// @Produce json
// @Param request body v1.LoginRequest true "params"
// @Success 200 {object} v1.LoginResponse
// @Router /api/v1/login [post]
func (h *UserHandler) Login(ctx *gin.Context) {
	var req v1.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	token, err := h.userService.Login(ctx, &req)
	if err != nil {
		v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		return
	}
	v1.HandleSuccess(ctx, v1.LoginResponseData{
		AccessToken: token,
	})
}

// GetProfile godoc
// @Summary 获取用户信息
// @Schemes
// @Description
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} v1.GetProfileResponse
// @Router /api/v1/user [get]
func (h *UserHandler) GetProfile(ctx *gin.Context) {
	userId := GetUserIdFromCtx(ctx)
	if userId == "" {
		v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		return
	}

	user, err := h.userService.GetProfile(ctx, userId)
	if err != nil {
		h.logger.WithContext(ctx).Error("userService.GetProfile error", zap.Error(err))
		if err == v1.ErrUnauthorized {
			v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		} else if err == v1.ErrNotFound {
			v1.HandleError(ctx, http.StatusNotFound, v1.ErrNotFound, nil)
		} else {
			v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
		}
		return
	}

	v1.HandleSuccess(ctx, user)
}

// UpdateProfile godoc
// @Summary 修改用户信息
// @Schemes
// @Description 支持修改昵称和密码，不允许修改邮箱。修改密码需要提供旧密码和新密码。
// @Tags 用户模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.UpdateProfileRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/user [put]
func (h *UserHandler) UpdateProfile(ctx *gin.Context) {
	userId := GetUserIdFromCtx(ctx)
	if userId == "" {
		v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		return
	}

	var req v1.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 验证：如果修改密码，必须同时提供旧密码和新密码
	if (req.OldPassword != "" && req.NewPassword == "") || (req.OldPassword == "" && req.NewPassword != "") {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.userService.UpdateProfile(ctx, userId, &req); err != nil {
		h.logger.WithContext(ctx).Error("userService.UpdateProfile error", zap.Error(err))
		if err == v1.ErrUnauthorized {
			v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		} else {
			v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
		}
		return
	}

	v1.HandleSuccess(ctx, nil)
}
