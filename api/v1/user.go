package v1

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20" example:"alice"`
	Email    string `json:"email" binding:"required,email" example:"1234@gmail.com"`
	Password string `json:"password" binding:"required,min=6" example:"123456"`
}

type LoginRequest struct {
	Account  string `json:"account" binding:"required" example:"alice"` // 支持用户名或邮箱登录
	Password string `json:"password" binding:"required" example:"123456"`
}
type LoginResponseData struct {
	AccessToken string `json:"accessToken"`
}
type LoginResponse struct {
	Response
	Data LoginResponseData
}

type UpdateProfileRequest struct {
	Nickname    string `json:"nickname" example:"alan"`
	OldPassword string `json:"oldPassword" example:"oldpassword"`
	NewPassword string `json:"newPassword" example:"newpassword"`
}
type GetProfileResponseData struct {
	UserId   string `json:"userId"`
	Username string `json:"username" example:"alice"`
	Email    string `json:"email" example:"pvesphere@gmail.com"`
	Nickname string `json:"nickname" example:"alan"`
}
type GetProfileResponse struct {
	Response
	Data GetProfileResponseData
}
