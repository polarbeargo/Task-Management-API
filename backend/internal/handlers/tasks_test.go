package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-manager/backend/internal/handlers"
	"task-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type MockTaskService struct {
	shouldReturnError bool
	tasks             []models.Task
	returnNotFound    bool
}

func (m *MockTaskService) CreateTask(db *gorm.DB, task models.Task) error {
	if m.shouldReturnError {
		return gorm.ErrInvalidData
	}
	m.tasks = append(m.tasks, task)
	return nil
}

func (m *MockTaskService) GetTaskByID(db *gorm.DB, id uuid.UUID) (models.Task, error) {
	if m.shouldReturnError {
		return models.Task{}, gorm.ErrInvalidData
	}
	if m.returnNotFound {
		return models.Task{}, gorm.ErrRecordNotFound
	}

	for _, task := range m.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return models.Task{ID: id, Title: "Test Task", Status: "pending"}, nil
}

func (m *MockTaskService) GetTasks(db *gorm.DB) ([]models.Task, error) {
	if m.shouldReturnError {
		return nil, gorm.ErrInvalidData
	}
	return m.tasks, nil
}

func (m *MockTaskService) GetTasksPaginated(db *gorm.DB, sortBy, order, page, pageSize string) ([]models.Task, int64, error) {
	if m.shouldReturnError {
		return nil, 0, gorm.ErrInvalidData
	}
	return m.tasks, int64(len(m.tasks)), nil
}

func (m *MockTaskService) UpdateTask(db *gorm.DB, id uuid.UUID, updated models.Task) error {
	if m.shouldReturnError {
		return gorm.ErrInvalidData
	}
	return nil
}

func (m *MockTaskService) DeleteTask(db *gorm.DB, id uuid.UUID) error {
	if m.shouldReturnError {
		return gorm.ErrInvalidData
	}
	return nil
}

func setupTaskHandler() (*handlers.TaskHandler, *MockTaskService, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	mockService := &MockTaskService{}
	handler := handlers.NewTaskHandler(nil, mockService)
	router := gin.New()

	// Add mock authentication middleware
	router.Use(func(c *gin.Context) {
		// Mock user_id in context
		c.Set("user_id", uuid.Must(uuid.NewV4()).String())
		c.Next()
	})

	return handler, mockService, router
}

func TestCreateTask(t *testing.T) {
	handler, _, router := setupTaskHandler()

	router.POST("/tasks", handler.CreateTask)

	task := models.Task{
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "pending",
	}

	taskJSON, _ := json.Marshal(task)
	req, _ := http.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestCreateTaskInvalidJSON(t *testing.T) {
	handler, _, router := setupTaskHandler()

	router.POST("/tasks", handler.CreateTask)

	req, _ := http.NewRequest("POST", "/tasks", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetTaskByID(t *testing.T) {
	handler, _, router := setupTaskHandler()

	router.GET("/tasks/:id", handler.GetTaskByID)

	taskID := uuid.Must(uuid.NewV4())

	req, _ := http.NewRequest("GET", "/tasks/"+taskID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var responseTask models.Task
	err := json.Unmarshal(w.Body.Bytes(), &responseTask)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if responseTask.Title != "Test Task" {
		t.Errorf("Expected title 'Test Task', got '%s'", responseTask.Title)
	}
}

func TestGetTaskByIDNotFound(t *testing.T) {
	handler, mockService, router := setupTaskHandler()

	router.GET("/tasks/:id", handler.GetTaskByID)

	mockService.returnNotFound = true

	taskID := uuid.Must(uuid.NewV4())

	req, _ := http.NewRequest("GET", "/tasks/"+taskID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetTasksPaginated(t *testing.T) {
	handler, mockService, router := setupTaskHandler()

	router.GET("/tasks", handler.GetTasks)

	mockService.tasks = []models.Task{
		{Title: "Task 1", Status: "pending"},
		{Title: "Task 2", Status: "completed"},
	}

	req, _ := http.NewRequest("GET", "/tasks?sortBy=created_at&order=desc&page=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["total"] != float64(2) {
		t.Errorf("Expected total 2, got %v", response["total"])
	}
}

func TestUpdateTask(t *testing.T) {
	handler, _, router := setupTaskHandler()

	router.PUT("/tasks/:id", handler.UpdateTask)

	taskID := uuid.Must(uuid.NewV4())
	updateData := models.Task{
		Title:       "Updated Task",
		Description: "Updated Description",
		Status:      "completed",
	}

	updateJSON, _ := json.Marshal(updateData)
	req, _ := http.NewRequest("PUT", "/tasks/"+taskID.String(), bytes.NewBuffer(updateJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDeleteTask(t *testing.T) {
	handler, _, router := setupTaskHandler()

	router.DELETE("/tasks/:id", handler.DeleteTask)

	taskID := uuid.Must(uuid.NewV4())

	req, _ := http.NewRequest("DELETE", "/tasks/"+taskID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}
