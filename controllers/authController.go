package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"ired.com/micuenta/app"
	"ired.com/micuenta/middlewares"
	"ired.com/micuenta/models"
	"ired.com/micuenta/repo"
)

func AuthRoutes(r *gin.Engine) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", authLogin)
		auth.GET("/logout", authLogout)
		auth.POST("/refresh", authRefresh)
		auth.POST("/forgot-password-req", authForgotPasswordReq)
		auth.POST("/forgot-password-send", authForgotPasswordSend)
		auth.POST("/change-password", middlewares.JwtPasswdAuth, authChangePassword)
	}
}

// @Summary        Authenticate a user
// @Description    Authenticates a user by validating credentials and returning a session token
// @Tags           Authentication
// @Accept         json
// @Produce        json
// @Param          x-access-token header string false "Access Token"
// @Param          credentials body models.UserRequest true "User Login Data"
// @Success 200    {object} models.SuccessResponse{record=models.UserResponse}
// @Failure 400    {object} models.ErrorResponse "Invalid Request or Already Logged In"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Router         /auth/login [post]
func authLogin(c *gin.Context) {
	// validate if session exist and if so stop login process
	session := sessions.Default(c)
	refreshTokenSession := session.Get("refresh_token")
	if refreshTokenSession != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "alreadyLogin")},
		)
		return
	}

	// validate if body exist
	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "errorFailedBody")},
		)
		return
	}

	// Bind and Validate the data and the struct
	var userReq models.UserRequest
	if err := c.ShouldBindJSON(&userReq); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	//set variables for handling dbs conns
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	// look up requested user
	userResponse, err := repo.Login(c, db, userReq)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		//if error clear seassion and auth cookies
		repo.Logout(c)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "loginSuccessful"),
			Record: userResponse,
		},
	)
}

// @Summary        Refresh authentication token
// @Description    Refreshes the user's session token if valid and returns a new one
// @Tags           Authentication
// @Accept         json
// @Produce        json
// @Param          x-access-token header string true "Access Token"
// @Success 200    {object} models.SuccessResponse{record=models.UserToken}
// @Failure 400    {object} models.ErrorResponse "Bad Request"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Router         /auth/refresh [post]
func authRefresh(c *gin.Context) {
	//set variables for handling dbs conns
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	// Verify and refresh the token
	tokenResponse, errType, err := repo.RefreshToken(c, db)
	if err != nil {
		c.AbortWithStatusJSON(
			errType,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Notice: "Token refreshed", Record: tokenResponse})
}

// @Summary        Logout a user
// @Description    Logs out the authenticated user by clearing session data
// @Tags           Authentication
// @Accept         json
// @Produce        json
// @Param          x-access-token header string true "Access Token"
// @Success 200    {object} models.SuccessResponse
// @Router         /auth/logout [get]
func authLogout(c *gin.Context) {
	repo.Logout(c)

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "logoutSuccessful")},
	)
}

// @Summary        Change user password
// @Description    Updates the user's password after validating the request body
// @Tags           Authentication
// @Accept         json
// @Produce        json
// @Param          x-access-token header string true "Access Token"
// @Param          user body models.UserChangePassword true "Password Change Request"
// @Success 200    {object} models.SuccessResponse
// @Failure 400    {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Router         /auth/change-password [post]
func authChangePassword(c *gin.Context) {
	// validate if body exist
	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "errorFailedBody")},
		)
		return
	}

	// Bind and Validate the data and the struct
	var userReq models.UserChangePassword
	if err := c.ShouldBindJSON(&userReq); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	// set variables for handling dbs conns
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	errType, err := repo.ChangePassword(c, db, userReq)
	if err != nil {
		c.JSON(
			errType,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notice": ginI18n.MustGetMessage(c, "changePasswdOk"),
	})
}

// @Summary        Request password reset
// @Description    Sends a password reset email to the user after validating the request
// @Tags           Authentication
// @Accept         json
// @Produce        json
// @Param          user body models.ForgotPasswordRequest true "User email for password reset"
// @Success 200    {object} models.SuccessResponse
// @Failure 400    {object} models.ErrorResponse "Invalid Request"
// @Failure 404    {object} models.ErrorResponse "User Not Found"
// @Router         /auth/forgot-password [post]
func authForgotPasswordReq(c *gin.Context) {
	// validate if body exist
	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "errorFailedBody")},
		)
		return
	}

	// Bind and Validate the data and the struct
	var userReq models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&userReq); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	// set variables for handling dbs conns
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	email, errType, err := repo.ForgotPasswordReq(c, db, userReq)
	if err != nil {
		c.JSON(
			errType,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "emailPasswordSent") + " " + email + " " + ginI18n.MustGetMessage(c, "emailInstuctions")},
	)
}

// @Summary        Send password reset instructions
// @Description    Processes a password reset request and sends recovery instructions to the user
// @Tags           Authentication
// @Accept         json
// @Produce        json
// @Param          user body models.ForgotPasswordSend true "User email for password reset"
// @Success 200    {object} models.SuccessResponse "Password reset instructions sent"
// @Failure 400    {object} models.ErrorResponse "Invalid Request"
// @Failure 404    {object} models.ErrorResponse "User Not Found"
// @Router         /auth/reset-password [post]
func authForgotPasswordSend(c *gin.Context) {
	// validate if body exist
	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "errorFailedBody")},
		)
		return
	}

	// Bind and Validate the data and the struct
	var userReq models.ForgotPasswordSend
	if err := c.ShouldBindJSON(&userReq); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	// set variables for handling dbs conns
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	errType, err := repo.ForgotPasswordSend(c, db, userReq)
	if err != nil {
		c.JSON(
			errType,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "changePasswdOk")},
	)
}
