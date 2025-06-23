package repo

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func getUser(db models.ConnDb, fieldSearch string, value string) (*models.UserResponse, error) {
	var user models.UserResponse
	query := `SELECT id, nombre, direccion, telefono, correo, docid, COALESCE(passwd, '') as passwd, change_passwd
		FROM publico.cliente WHERE ` + fieldSearch + ` = $1 AND activo=true
		LIMIT 1`

	err := db.ConnPgsql.QueryRow(db.Ctx, query, value).Scan(&user.Profile.Id, &user.Profile.Nombre, &user.Profile.Direccion, &user.Profile.Telefono, &user.Profile.Correo,
		&user.Profile.Username, &user.Credentials.Passwd, &user.Credentials.ChangePasswd)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("recordDontExist")
		}
		utils.Logline("error on getting cliente", err)
		return nil, errors.New("errorGetData")
	}

	return &user, nil
}

func updateUser(db models.ConnDb, fieldUpdate string, value string, userId string) error {
	query := `UPDATE publico.cliente SET ` + fieldUpdate + ` = $1, change_passwd=false WHERE empresa_id=1 AND id=$2`

	_, err := db.ConnPgsql.Exec(db.Ctx, query, value, userId)
	if err != nil {
		utils.Logline("error updating user", err)
		return errors.New("errorGetData")
	}

	return nil
}

func Login(c *gin.Context, db models.ConnDb, userReq models.UserRequest) (*models.UserResponse, error) {
	// look up requested user
	user, err := getUser(db, "docid", userReq.Username)
	if err != nil {
		return nil, errors.New("invalidLogin")
	}

	// handle null value of passwd column
	passwordBytes := []byte{}
	if user.Credentials.Passwd.Valid {
		passwordBytes = []byte(user.Credentials.Passwd.String)
	}

	// compare passwd hash from db to user pass hash
	err = bcrypt.CompareHashAndPassword([]byte(passwordBytes), []byte(userReq.Password))
	if err != nil {
		return nil, errors.New("invalidLogin")
	}

	//check if remember is true or not
	remember, err := strconv.ParseBool(userReq.Remember)
	if err != nil {
		utils.Logline("error parsing boolean remember on login", err)
		return nil, errors.New("invalidLogin")
	}

	// generate authorization token
	userToken, err := GenerateAuthTokens(user.Profile.Id, remember, user.Credentials.ChangePasswd)
	if err != nil {
		utils.Logline("error creating tokens:", err)
		return nil, errors.New("invalidLogin")
	}
	user.Tokens = *userToken

	// create session and save refresh token
	session, err := saveSession(c,
		&sessionInternal{
			UserId:       user.Profile.Id,
			ChangePasswd: user.Credentials.ChangePasswd,
			RefreshToken: userToken.Refresh,
			ExpiresAt:    userToken.RefreshExpiresAt,
			Remember:     remember,
		})
	if err != nil {
		utils.Logline("error saving the session", err)
		return nil, errors.New("invalidLogin")
	}

	// associate session with client_id and save also deviceInfo
	err = updateClienteOnSession(db, session.ID(), user.Profile.Id, &userReq.DeviceInfo)
	if err != nil {
		utils.Logline("error update the cliente data on the session", err)
		return nil, errors.New("invalidLogin")
	}

	// create cookie for authToken
	// secureCookie, _ := strconv.ParseBool(os.Getenv("SECURE_COOKIE"))
	// c.SetSameSite(http.SameSiteLaxMode)
	// c.SetCookie("auth_token", userToken.Auth, userToken.AuthMaxAge, "/", os.Getenv("DOMAIN"), secureCookie, false)

	return user, nil
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Options(sessions.Options{MaxAge: -1})
	session.Save()
}

func RefreshToken(c *gin.Context, db models.ConnDb) (*models.UserToken, int, error) {
	// validate if session exist and if not stop refresh process
	session := sessions.Default(c)
	refreshTokenSession := session.Get("refresh_token")
	if refreshTokenSession == nil {
		return nil, http.StatusUnauthorized, errors.New("tokenMissing")
	}

	// get the token from the header
	refreshTokenHeader := c.GetHeader("Authorization")
	if len(refreshTokenHeader) < 8 || refreshTokenHeader[:7] != "Bearer " {
		return nil, http.StatusUnauthorized, errors.New("tokenMissing")
	}
	refreshTokenHeader = refreshTokenHeader[7:]

	// validate if token header is the same as the session - CSRF prevention
	if refreshTokenHeader != refreshTokenSession {
		return nil, http.StatusUnauthorized, errors.New("tokenInvalid")
	}

	// validate if session has expires
	expiresAt, err := time.Parse(time.RFC3339, session.Get("exp").(string))
	if err != nil {
		utils.Logline("error on parsing exp from session data", err)
		return nil, http.StatusInternalServerError, errors.New("errorInternal")
	}

	if time.Now().After(expiresAt) {
		return nil, http.StatusUnauthorized, errors.New("tokenExpired")
	}

	// generate authorization token
	userId := session.Get("user_id")
	remember := session.Get("remember").(bool)
	changePasswd := session.Get("change_passwd").(bool)
	userToken, err := GenerateAuthTokens(userId.(string), remember, changePasswd)
	if err != nil {
		utils.Logline("error creating tokens:", err)
		return nil, http.StatusInternalServerError, errors.New("errorInternal")
	}

	// update session
	_, err = saveSession(c,
		&sessionInternal{
			UserId:       userId.(string),
			RefreshToken: userToken.Refresh,
			ExpiresAt:    userToken.RefreshExpiresAt,
			Remember:     remember,
			ChangePasswd: changePasswd,
		})
	if err != nil {
		utils.Logline("error updating session tokens:", err)
		return nil, http.StatusInternalServerError, errors.New("errorInternal")
	}

	return userToken, http.StatusOK, nil
}

func ForgotPasswordReq(c *gin.Context, db models.ConnDb, userReq models.ForgotPasswordRequest) (string, int, error) {
	// look up requested user
	user, err := getUser(db, "docid", userReq.Username)
	if err != nil {
		return "", http.StatusBadRequest, errors.New("errorGetData")
	}

	// get MaxAge for the authLink to work
	linkMaxAge, err := strconv.Atoi(os.Getenv("LINK_MAX_AGE"))
	if err != nil {
		utils.Logline("error parsing AUTH_MAX_AGE", err)
		return "", http.StatusInternalServerError, errors.New("errorInternal")
	}
	linkExpirationTime := time.Now().Add(time.Duration(linkMaxAge) * time.Second)

	// create claims for auth token
	claims := &models.Claims{
		UserId: userReq.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "micuenta",
			Subject:   userReq.Username,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(linkExpirationTime),
		},
	}
	// Sign and get the complete encoded authorization token as a string using the secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	TokenString, err := token.SignedString([]byte(os.Getenv("LINK_SECRET")))
	if err != nil {
		utils.Logline("error signing auth token", err)
		return "", http.StatusInternalServerError, errors.New("errorInternal")
	}

	// save token to user record
	query := `UPDATE publico.cliente SET info=jsonb_set(info,'{link_email_token}', '"` + TokenString + `"') WHERE id = $1`
	_, err = db.ConnPgsql.Exec(db.Ctx, query, user.Profile.Id)
	if err != nil {
		utils.Logline("error updating user", err)
		return "", http.StatusInternalServerError, errors.New("errorInternal")
	}

	linkTokenString := `?username=` + user.Profile.Username + `&uid=` + TokenString
	bodyEmail := `
		<!DOCTYPE html>
		<html lang="en">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>Reestablecer Contraseña</title>
				<style>
					body {font-family: Arial, sans-serif; background-color: #f6f8fa; margin: 0;padding: 0; }
					.container {width: 100%; max-width: 600px; margin: 0 auto; padding: 20px;}
					.header {text-align: center; padding: 20px 0;}
					.header img {width: 200px;}
					.content {padding: 20px; border: 1px solid #e1e4e8; border-radius: 5px;}
					.content h1 {font-size: 24px; color: #333333; text-align: center;}
					.content p {font-size: 16px; color: #333333;}
					.footer {text-align: center; padding: 20px; font-size: 12px; color: #666666;}
				</style>
			</head>
			<body>
				<div class="container">
					<div class="header">
						<img src="cid:image001" alt="Besser Solutions Logo">
						<h1>Restablecer tu contraseña</h1>
					</div>
					<div class="content">
						<b>Hola, </b>
						<p>Nos enteramos de que perdiste tu contraseña de MiCuenta. ¡Lo sentimos!</p>
						<p>Pero no te preocupes, puedes utilizar el siguiente botón para restablecer tu contraseña:</p>
						<a href="` + os.Getenv("LINK_CHANGE_PASSWORD") + linkTokenString + `" style="display: block; width: 200px; margin: 20px auto; padding: 10px 0; text-align: center; background-color: #28a745 !important; color: #ffffff !important; text-decoration: none !important; border-radius: 5px !important;">Re-establecer Contraseña</a>
						<p>Si no utilizas este enlace en 3 horas, caducará.</p>
						<p>Gracias,<br>El equipo de Besser Solutions</p>
					</div>
					<div class="footer">
						<p>Recibiste este correo electrónico porque solicitaste un restablecimiento de contraseña.</p>
						<p>Besser Solutions, C.A. • Santa Irene, Calle San Miguel, Edif. Asdrubal Jose PB • Punto Fijo, Falcon 4102</p>
					</div>
				</div>
			</body>
		</html>
	`
	// send email with hashLink
	err = utils.SendEmail(user.Profile.Correo, ginI18n.MustGetMessage(c, "titleChangePassword"), bodyEmail)
	if err != nil {
		utils.Logline("error sending the email", err)
		return "", http.StatusInternalServerError, errors.New("errorEmail")
	}

	return strings.Join(user.Profile.Correo, ","), http.StatusOK, nil
}

func ForgotPasswordSend(c *gin.Context, db models.ConnDb, userReq models.ForgotPasswordSend) (int, error) {
	// save token to user record
	query := `SELECT id, info->>'link_email_token' as  token FROM publico.cliente WHERE docid=$1 AND info->>'link_email_token'=$2`

	var userId string
	var tokenString string
	err := db.ConnPgsql.QueryRow(db.Ctx, query, userReq.Username, userReq.Token).Scan(&userId, &tokenString)
	if err != nil {
		if err == pgx.ErrNoRows {
			return http.StatusUnauthorized, errors.New("tokenInvalid")
		}
		utils.Logline("error getting user", err)
		return http.StatusInternalServerError, errors.New("errorInternal")
	}

	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("LINK_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return http.StatusUnauthorized, errors.New("tokenExpired")
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(userReq.Password), 10)
	query = `UPDATE publico.cliente SET change_passwd=false, info=jsonb_set(info,'{link_email_token}', '""'), passwd=$1 WHERE id=$2`
	_, err = db.ConnPgsql.Exec(db.Ctx, query, string(hash), userId)
	if err != nil {
		return http.StatusInternalServerError, errors.New("errorInternal")
	}

	return http.StatusOK, nil
}

type sessionInternal struct {
	UserId       string
	ChangePasswd bool
	RefreshToken string
	ExpiresAt    string
	Remember     bool
}

func saveSession(c *gin.Context, sessionData *sessionInternal) (sessions.Session, error) {
	session := sessions.Default(c)
	session.Set("user_id", sessionData.UserId)
	session.Set("change_passwd", sessionData.ChangePasswd)
	session.Set("refresh_token", sessionData.RefreshToken)
	session.Set("remember", sessionData.Remember)
	session.Set("exp", sessionData.ExpiresAt)
	if sessionData.Remember {
		// use this maxAge for refresh token and session if remember is true
		maxAgeRefreshRemember, err := strconv.Atoi(os.Getenv("REFRESH_MAX_AGE_REMEMBER"))
		if err != nil {
			utils.Logline("error gettin max age when remember is true", err)
			return nil, err
		}
		session.Options(sessions.Options{MaxAge: maxAgeRefreshRemember})
	}
	if err := session.Save(); err != nil {
		utils.Logline("error saving the session", err)
		return nil, errors.New("invalidLogin")
	}
	return session, nil
}

func ChangePassword(c *gin.Context, db models.ConnDb, userReq models.UserChangePassword) (int, error) {
	session := sessions.Default(c)
	userId := session.Get("user_id")
	if userId == nil {
		return http.StatusUnauthorized, errors.New("tokenMissing")
	}

	user, err := getUser(db, "id", userId.(string))
	if err != nil {
		return http.StatusInternalServerError, errors.New("errorInternal")
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(userReq.Password), 10)
	err = updateUser(db, "passwd", string(hash), user.Profile.Id)
	if err != nil {
		return http.StatusInternalServerError, errors.New("errorInternal")
	}

	session.Set("change_passwd", false)
	err = session.Save()
	if err != nil {
		utils.Logline("error saving the session", err)
	}

	return http.StatusOK, nil
}

func GenerateAuthTokens(userId string, remember bool, changePassword bool) (*models.UserToken, error) {
	// get max age for auth token
	authMaxAge, err := strconv.Atoi(os.Getenv("AUTH_MAX_AGE"))
	if err != nil {
		utils.Logline("error parsing AUTH_MAX_AGE", err)
		return nil, err
	}
	if changePassword {
		authMaxAge = 10
	}
	authExpirationTime := time.Now().Add(time.Duration(authMaxAge) * time.Second)

	// create claims for auth token
	claims := &models.Claims{
		UserId:       userId,
		ChangePasswd: changePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "micuenta",
			Subject:   userId,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(authExpirationTime),
		},
	}
	// Sign and get the complete encoded authorization token as a string using the secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	authTokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		utils.Logline("error signing auth token", err)
		return nil, err
	}

	// get max age for refresh token
	refreshMaxAge, err := strconv.Atoi(os.Getenv("REFRESH_MAX_AGE"))
	if err != nil {
		utils.Logline("error parsing REFRESH_MAX_AGE", err)
		return nil, err
	}
	if remember {
		// use this maxAge for refresh token and session if remember is true
		maxAgeRefreshRemember, err := strconv.Atoi(os.Getenv("REFRESH_MAX_AGE_REMEMBER"))
		if err != nil {
			utils.Logline("error gettin max age when remember is true", err)
			return nil, err
		}
		refreshMaxAge = maxAgeRefreshRemember
	}
	refreshExpirationTime := time.Now().Add(time.Duration(refreshMaxAge) * time.Second)

	// create claims for refresh token
	claims = &models.Claims{
		UserId:       userId,
		ChangePasswd: changePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "micuenta",
			Subject:   userId,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(refreshExpirationTime),
		},
	}
	// Sign and get the complete encoded authorization token as a string using the secret
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refreshTokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		utils.Logline("error signing refresh token", err)
		return nil, err
	}

	return &models.UserToken{
		Auth:             authTokenString,
		AuthMaxAge:       authMaxAge,
		AuthExpiresAt:    authExpirationTime.Format(time.RFC3339),
		Refresh:          refreshTokenString,
		RefreshMaxAge:    refreshMaxAge,
		RefreshExpiresAt: refreshExpirationTime.Format(time.RFC3339),
	}, err
}

func updateClienteOnSession(db models.ConnDb, sessionId string, clienteId string, deviceInfo *models.UserDeviceInfo) error {
	var deviceInfoString pgtype.Text
	if deviceInfo != nil {
		deviceInfoJSON, err := json.Marshal(deviceInfo)
		if err != nil {
			return err
		}
		deviceInfoString.String = string(deviceInfoJSON)
		deviceInfoString.Valid = true
	}

	_, err := db.ConnPgsql.Exec(db.Ctx, `UPDATE publico.cliente_session_store SET empresa_id=$2, cliente_id=$3, device_info=$4 WHERE key=$1 AND cliente_id IS NULL`,
		sessionId, 1, clienteId, deviceInfoString.String)
	return err
}
