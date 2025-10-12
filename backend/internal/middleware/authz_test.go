package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"task-manager/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func createTestToken(role string, permissions []string) (string, error) {
	claims := jwt.MapClaims{
		"role":        role,
		"permissions": permissions,
		"exp":         time.Now().Add(time.Hour).Unix(),
		"iss":         "taskify-backend",
		"user_id":     "test-user-123",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("default_secret_change_in_production"))
}

func TestAuthzMiddleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.AuthzMiddleware(middleware.AuthzConfig{Role: "admin"}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthzMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.AuthzMiddleware(middleware.AuthzConfig{Role: "admin"}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthzMiddleware_ValidTokenCorrectRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := createTestToken("admin", []string{"read", "write"})
	if err != nil {
		t.Fatal("Failed to create test token:", err)
	}

	router := gin.New()
	router.Use(middleware.AuthzMiddleware(middleware.AuthzConfig{Role: "admin"}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthzMiddleware_ValidTokenWrongRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := createTestToken("user", []string{"read"})
	if err != nil {
		t.Fatal("Failed to create test token:", err)
	}

	router := gin.New()
	router.Use(middleware.AuthzMiddleware(middleware.AuthzConfig{Role: "admin"}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAuthzMiddleware_ValidTokenCorrectPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := createTestToken("user", []string{"tasks:read", "tasks:write"})
	if err != nil {
		t.Fatal("Failed to create test token:", err)
	}

	router := gin.New()
	router.Use(middleware.AuthzMiddleware(middleware.AuthzConfig{
		Permissions: []string{"tasks:read"},
	}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthzMiddleware_ValidTokenMissingPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := createTestToken("user", []string{"tasks:read"})
	if err != nil {
		t.Fatal("Failed to create test token:", err)
	}

	router := gin.New()
	router.Use(middleware.AuthzMiddleware(middleware.AuthzConfig{
		Permissions: []string{"tasks:write"},
	}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAuthzMiddleware_NoRequirements(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := createTestToken("user", []string{})
	if err != nil {
		t.Fatal("Failed to create test token:", err)
	}

	router := gin.New()
	router.Use(middleware.AuthzMiddleware(middleware.AuthzConfig{}))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}
