package handler

import (
	v1 "pvesphere/api/v1"
	"pvesphere/internal/handler"
	"pvesphere/internal/middleware"
	"pvesphere/test/mocks/service"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
)

func TestUserHandler_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	params := v1.RegisterRequest{
		Password: "123456",
		Email:    "xxx@gmail.com",
	}

	mockUserService := mock_service.NewMockUserService(ctrl)
	mockUserService.EXPECT().Register(gomock.Any(), &params).Return(nil)

	userHandler := handler.NewUserHandler(hdl, mockUserService)
	router.POST("/register", userHandler.Register)

	e := newHttpExcept(t, router)
	obj := e.POST("/register").
		WithHeader("Content-Type", "application/json").
		WithJSON(params).
		Expect().
		Status(http.StatusOK).
		JSON().
		Object()
	obj.Value("code").IsEqual(0)
	obj.Value("message").IsEqual("ok")
}

func TestUserHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	params := v1.LoginRequest{
		Email:    "xxx@gmail.com",
		Password: "123456",
	}

	tk := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySWQiOiJ4eHgiLCJleHAiOjE3MzgyMjA1MTQsIm5iZiI6MTczMDQ0NDUxNCwiaWF0IjoxNzMwNDQ0NTE0fQ.3D4YupmPBCkv16ESnYyWSV5Mxcdu0twzEUqx0K-UiWo"
	mockUserService := mock_service.NewMockUserService(ctrl)
	mockUserService.EXPECT().Login(gomock.Any(), &params).Return(tk, nil)

	userHandler := handler.NewUserHandler(hdl, mockUserService)
	router.POST("/login", userHandler.Login)

	obj := newHttpExcept(t, router).POST("/login").
		WithHeader("Content-Type", "application/json").
		WithJSON(params).
		Expect().
		Status(http.StatusOK).
		JSON().
		Object()
	obj.Value("code").IsEqual(0)
	obj.Value("message").IsEqual("ok")
	obj.Value("data").Object().Value("accessToken").IsEqual(tk)
}

func TestUserHandler_GetProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nickname := "xxxxx"
	mockUserService := mock_service.NewMockUserService(ctrl)
	mockUserService.EXPECT().GetProfile(gomock.Any(), userId).Return(&v1.GetProfileResponseData{
		UserId:   userId,
		Nickname: nickname,
	}, nil)

	userHandler := handler.NewUserHandler(hdl, mockUserService)
	router.Use(middleware.NoStrictAuth(jwt, logger))
	router.GET("/user", userHandler.GetProfile)

	obj := newHttpExcept(t, router).GET("/user").
		WithHeader("Authorization", "Bearer "+genToken(t)).
		Expect().
		Status(http.StatusOK).
		JSON().
		Object()
	obj.Value("code").IsEqual(0)
	obj.Value("message").IsEqual("ok")
	objData := obj.Value("data").Object()
	objData.Value("userId").IsEqual(userId)
	objData.Value("nickname").IsEqual(nickname)
}

func TestUserHandler_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	params := v1.UpdateProfileRequest{
		Nickname: "alan",
		Email:    "alan@gmail.com",
	}

	mockUserService := mock_service.NewMockUserService(ctrl)
	mockUserService.EXPECT().UpdateProfile(gomock.Any(), userId, &params).Return(nil)

	userHandler := handler.NewUserHandler(hdl, mockUserService)
	router.Use(middleware.StrictAuth(jwt, logger))
	router.PUT("/user", userHandler.UpdateProfile)

	obj := newHttpExcept(t, router).PUT("/user").
		WithHeader("Content-Type", "application/json").
		WithHeader("Authorization", "Bearer "+genToken(t)).
		WithJSON(params).
		Expect().
		Status(http.StatusOK).
		JSON().
		Object()
	obj.Value("code").IsEqual(0)
	obj.Value("message").IsEqual("ok")
}
