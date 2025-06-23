package middlewares

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func isOriginAllowed(origin string) bool {
	originsString := os.Getenv("CORS_ORIGINS")

	var allowedOrigins []string
	if originsString != "" {
		allowedOrigins = strings.Split(originsString, ",")
	}

	for _, allowedOrigin := range allowedOrigins {
		if origin == allowedOrigin {
			return true
		}
	}
	return false
}

func isHostAllowed(host string) bool {
	originsString := os.Getenv("CORS_ORIGINS")

	var allowedOrigins []string
	if originsString != "" {
		allowedOrigins = strings.Split(originsString, ",")
	}

	for _, allowedOrigin := range allowedOrigins {
		parsedURL, _ := url.Parse(allowedOrigin)
		fqdn := parsedURL.Host

		if host == fqdn {
			return true
		}
	}
	return false
}

func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Origin header from the request
		origin := c.Request.Header.Get("Origin")
		// expectedHost := "127.0.0.1:" + os.Getenv("PORT")

		// check if origin is allowed
		// if !isOriginAllowed(origin) && c.Request.Host != expectedHost {
		// 	utils.Logline("Request denied, ORIGIN is not allowed | Host is " + c.Request.Host + " | Origin is " + c.Request.Header.Get("origin"))
		// 	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid origin header"})
		// 	return
		// }

		// check if host is allowed
		// if !isHostAllowed(c.Request.Host) && c.Request.Host != expectedHost {
		// 	utils.Logline("Request denied, HOST is not Allowed | Host is " + c.Request.Host + " | Origin is " + c.Request.Header.Get("origin"))
		// 	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid host header"})
		// 	return
		// }

		// If the origin is allowed, set CORS headers in the response
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", os.Getenv("CORS_ALLOW_HEADERS"))
		c.Writer.Header().Set("Access-Control-Allow-Methods", os.Getenv("CORS_METHODS"))
		c.Writer.Header().Set("Access-Control-Max-Age", os.Getenv("CORS_MAX_AGE"))

		// Handle preflight OPTIONS requests by aborting with status 204
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "default-src 'self'; connect-src *; font-src *; script-src-elem * 'unsafe-inline'; img-src * data:; style-src * 'unsafe-inline';")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("Referrer-Policy", "strict-origin")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Permissions-Policy", "geolocation=(),midi=(),sync-xhr=(),microphone=(),camera=(),magnetometer=(),gyroscope=(),fullscreen=(self),payment=()")

		c.Next()
	}
}
