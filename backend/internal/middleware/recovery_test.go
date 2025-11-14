package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"task-manager/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestRecoveryWithLog_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.RecoveryWithLog())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRecoveryWithLog_WithPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.RecoveryWithLog())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req, _ := http.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	expectedError := `{"error":"internal server error"}`
	if w.Body.String() != expectedError {
		t.Errorf("Expected error message %s, got %s", expectedError, w.Body.String())
	}
}
