package middleware

import (
	"net/http"
	"strings"

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
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Use your JWT secret here
			return []byte("default_secret"), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}

		if config.Role != "" {
			role, _ := claims["role"].(string)
			if role != config.Role {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
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
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing permission: " + required})
					return
				}
			}
		}

		c.Next()
	}
}
