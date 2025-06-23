package models

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// user request for handle login
type UserRequest struct {
	Username   string         `json:"username" binding:"required,min=7,max=30"`
	Password   string         `json:"password" binding:"required,min=7,max=30"`
	Remember   string         `json:"remember" binding:"required,boolean"`
	DeviceInfo UserDeviceInfo `json:"device_info"`
}

type UserDeviceInfo struct {
	Model        string `json:"model" binding:"omitempty,min=7,max=30"`
	System       string `json:"system" binding:"omitempty,min=7,max=30"`
	Manufacturer string `json:"manufacturer" binding:"omitempty,min=7,max=30"`
	Platform     string `json:"platform" binding:"omitempty,min=6,max=30"`
	UserAgent    string `json:"user_agent" binding:"omitempty,min=7,max=200"`
	ScreenSize   string `json:"screen_size" binding:"omitempty,min=7,max=30"`
}

// user response to handle login
type UserResponse struct {
	Profile     UserProfile     `json:"profile"`
	Credentials UserCredentials `json:"credentials"`
	Tokens      UserToken       `json:"tokens"`
}

type UserProfile struct {
	Id        string   `json:"id"`
	Username  string   `json:"username"`
	Nombre    string   `json:"nombre"`
	Direccion string   `json:"direccion"`
	Telefono  []string `json:"telefono"`
	Correo    []string `json:"correo"`
}

type UserCredentials struct {
	Passwd       pgtype.Text `json:"-"`
	ChangePasswd bool        `json:"change_passwd"`
}

type UserToken struct {
	Auth             string `json:"auth"`
	AuthMaxAge       int    `json:"-"`
	AuthExpiresAt    string `json:"auth_expires_at"`
	Refresh          string `json:"refresh"`
	RefreshMaxAge    int    `json:"-"`
	RefreshExpiresAt string `json:"refresh_expires_at"`
}

// user request for handle change password
type UserChangePassword struct {
	Username string `json:"username" binding:"required,min=7,max=30"`
	Password string `json:"password" binding:"required,passwd_strenght,min=7,max=30"`
}

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("passwd_strenght", passwdStrenght)
	}
}

// claims on auth and refresh token
type Claims struct {
	UserId       string
	ChangePasswd bool
	jwt.RegisteredClaims
}

// user request for forgotPassword
type ForgotPasswordRequest struct {
	Username string `json:"username" binding:"required,min=7,max=30"`
}

// user request for forgotPassword
type ForgotPasswordSend struct {
	Username string `json:"username" binding:"required,min=7,max=30"`
	Password string `json:"password" binding:"required,passwd_strenght,min=7,max=30"`
	Token    string `json:"token" binding:"required"`
}
