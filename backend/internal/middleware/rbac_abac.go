package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"task-manager/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
)

func RBACMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, role, err := extractUserFromToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		if len(requiredRoles) > 0 {
			hasRole := false
			for _, requiredRole := range requiredRoles {
				if role == requiredRole {
					hasRole = true
					break
				}
			}

			if !hasRole {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":          "Insufficient permissions",
					"required_roles": requiredRoles,
					"user_role":      role,
				})
				return
			}
		}

		c.Set("user_id", userID)
		c.Set("user_role", role)
		c.Next()
	}
}

func PermissionMiddleware(authService services.AuthorizationService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _, err := extractUserFromToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		ctx := context.Background()
		hasPermission, err := authService.HasPermission(ctx, userID, resource, action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Permission check failed"})
			return
		}

		if !hasPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":               "Insufficient permissions",
				"required_permission": resource + ":" + action,
			})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func ABACMiddleware(authService services.AuthorizationService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _, err := extractUserFromToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		request := services.AuthorizationRequest{
			UserID:    userID,
			Resource:  resource,
			Action:    action,
			Context:   buildRequestContext(c),
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			RequestID: c.GetHeader("X-Request-ID"),
		}

		if resourceIDStr := c.Param("id"); resourceIDStr != "" {
			if resourceID, err := uuid.FromString(resourceIDStr); err == nil {
				request.ResourceID = &resourceID
			}
		}
		if resourceIDStr := c.Param("user_id"); resourceIDStr != "" {
			if resourceID, err := uuid.FromString(resourceIDStr); err == nil {
				request.ResourceID = &resourceID
			}
		}
		if resourceIDStr := c.Param("task_id"); resourceIDStr != "" {
			if resourceID, err := uuid.FromString(resourceIDStr); err == nil {
				request.ResourceID = &resourceID
			}
		}

		ctx := context.Background()
		decision, err := authService.IsAuthorized(ctx, request)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
			return
		}

		go func() {
			authService.LogAuthorizationDecision(context.Background(), *decision)
		}()

		if decision.Decision != "allowed" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":       "Access denied",
				"reason":      decision.Reason,
				"policy_type": decision.PolicyType,
			})
			return
		}

		c.Set("user_id", userID)
		c.Set("auth_decision", decision)
		c.Next()
	}
}

func ResourceOwnershipMiddleware(resourceType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		userID, ok := userIDInterface.(uuid.UUID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			return
		}

		userRole, _ := c.Get("user_role")
		if role, ok := userRole.(string); ok && role == "admin" {
			c.Next()
			return
		}

		var resourceID uuid.UUID
		var err error

		if resourceIDStr := c.Param("user_id"); resourceIDStr != "" && resourceType == "user" {
			resourceID, err = uuid.FromString(resourceIDStr)
		} else if resourceIDStr := c.Param("id"); resourceIDStr != "" {
			resourceID, err = uuid.FromString(resourceIDStr)
		}

		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
			return
		}

		if resourceType == "user" || resourceType == "profile" {
			if resourceID != userID {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Can only access own profile"})
				return
			}
		}

		c.Next()
	}
}

func AdminOnlyMiddleware() gin.HandlerFunc {
	return RBACMiddleware("admin")
}

func UserOrAdminMiddleware() gin.HandlerFunc {
	return RBACMiddleware("user", "admin")
}

func extractUserFromToken(c *gin.Context) (uuid.UUID, string, error) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return uuid.Nil, "", jwt.ErrTokenMalformed
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte("default_secret"), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", jwt.ErrInvalidKey
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, "", jwt.ErrInvalidKey
	}

	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		return uuid.Nil, "", err
	}

	role, _ := claims["role"].(string)

	return userID, role, nil
}

func buildRequestContext(c *gin.Context) map[string]interface{} {
	context := make(map[string]interface{})

	context["http_method"] = c.Request.Method
	context["http_path"] = c.Request.URL.Path

	if len(c.Request.URL.Query()) > 0 {
		queryParams := make(map[string]string)
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				queryParams[k] = v[0]
			}
		}
		context["query_params"] = queryParams
	}

	headers := make(map[string]string)
	headers["content_type"] = c.GetHeader("Content-Type")
	headers["accept"] = c.GetHeader("Accept")
	context["headers"] = headers
	context["request_time"] = c.Request.Header.Get("Date")
	context["server_time"] = strconv.FormatInt(c.Request.Context().Value("request_start_time").(int64), 10)

	return context
}
