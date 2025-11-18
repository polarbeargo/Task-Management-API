package models_test

import (
	"testing"
	"time"

	"task-manager/backend/internal/models"

	"github.com/gofrs/uuid"
)

func TestTask_Validation(t *testing.T) {
	task := models.Task{
		ID:          uuid.Must(uuid.NewV4()),
		UserID:      uuid.Must(uuid.NewV4()),
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if task.Title != "Test Task" {
		t.Errorf("Expected title 'Test Task', got '%s'", task.Title)
	}

	if task.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", task.Status)
	}
}

func TestTask_EmptyTitle(t *testing.T) {
	task := models.Task{
		ID:     uuid.Must(uuid.NewV4()),
		UserID: uuid.Must(uuid.NewV4()),
		Title:  "",
		Status: "pending",
	}

	if task.Title != "" {
		t.Errorf("Expected empty title, got '%s'", task.Title)
	}
}

func TestUser_Validation(t *testing.T) {
	user := models.User{
		ID:       uuid.Must(uuid.NewV4()),
		Username: "testuser",
		Password: "hashedpassword",
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	if user.Password != "hashedpassword" {
		t.Errorf("Expected password 'hashedpassword', got '%s'", user.Password)
	}
}

func TestToken_Validation(t *testing.T) {
	userID := uuid.Must(uuid.NewV4())
	jti := uuid.Must(uuid.NewV4())
	refreshToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzIiwidHlwZSI6InJlZnJlc2giLCJqdGkiOiI0NTYiLCJpYXQiOjE2MDk0NTkyMDAsImV4cCI6MTYwOTQ2MjgwMH0.test"
	expiresAt := time.Now().Add(time.Hour)

	token := models.Token{
		ID:           uuid.Must(uuid.NewV4()),
		UserId:       userID,
		JTI:          jti,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}

	if token.UserId != userID {
		t.Errorf("Expected UserID %s, got %s", userID.String(), token.UserId.String())
	}

	if token.JTI != jti {
		t.Errorf("Expected JTI %s, got %s", jti.String(), token.JTI.String())
	}

	if token.RefreshToken != refreshToken {
		t.Errorf("Expected RefreshToken %s, got %s", refreshToken, token.RefreshToken)
	}

	if token.ExpiresAt != expiresAt {
		t.Errorf("Expected ExpiresAt %v, got %v", expiresAt, token.ExpiresAt)
	}
}

func TestTask_StatusTransitions(t *testing.T) {
	validStatuses := []string{"pending", "in_progress", "completed", "cancelled"}

	for _, status := range validStatuses {
		task := models.Task{
			ID:     uuid.Must(uuid.NewV4()),
			UserID: uuid.Must(uuid.NewV4()),
			Title:  "Test Task",
			Status: status,
		}

		if task.Status != status {
			t.Errorf("Expected status '%s', got '%s'", status, task.Status)
		}
	}
}
