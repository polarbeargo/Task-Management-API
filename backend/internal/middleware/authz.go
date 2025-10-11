package middleware

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthzConfig struct {
	Role        string
	Permissions []string
}

func AuthzMiddleware(config AuthzConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "missing_token",
				"message": "Authorization header is required",
			})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_token_format",
				"message": "Authorization header must use Bearer token",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "default_secret_change_in_production"
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_token",
				"message": "Token validation failed",
			})
			return
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "expired_token",
				"message": "Token has expired",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_claims",
				"message": "Token claims are invalid",
			})
			return
		}

		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":   "expired_token",
					"message": "Token has expired",
				})
				return
			}
		}

		if iss, ok := claims["iss"].(string); ok && iss != "taskify-backend" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_issuer",
				"message": "Token issuer is invalid",
			})
			return
		}

		if config.Role != "" {
			role, _ := claims["role"].(string)
			if role != config.Role && role != "admin" { 
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "insufficient_role",
					"message": "User role does not have access to this resource",
				})
				return
			}
		}

		if len(config.Permissions) > 0 {
			perms, _ := claims["permissions"].([]interface{})
			userPerms := map[string]bool{}
			for _, p := range perms {
				if ps, ok := p.(string); ok {
					userPerms[ps] = true
				}
			}

			for _, required := range config.Permissions {
				if !userPerms[required] {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error":   "missing_permission",
						"message": "User does not have required permission: " + required,
					})
					return
				}
			}
		}

		c.Set("user_id", claims["user_id"])
		c.Set("user_role", claims["role"])
		c.Set("user_permissions", claims["permissions"])

		c.Next()
	}
}
