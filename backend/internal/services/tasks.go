package services

import (
	"strconv"
	"task-manager/backend/internal/models"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type TaskService interface {
	CreateTask(db *gorm.DB, task models.Task) error
	GetTaskByID(db *gorm.DB, id uuid.UUID) (models.Task, error)
	GetTasks(db *gorm.DB) ([]models.Task, error)
	UpdateTask(db *gorm.DB, id uuid.UUID, updated models.Task) error
	DeleteTask(db *gorm.DB, id uuid.UUID) error
	GetTasksPaginated(db *gorm.DB, sortBy, order, page, pageSize string) ([]models.Task, int64, error)
}

type TaskServiceImpl struct{}

func NewTaskService() *TaskServiceImpl {
	return &TaskServiceImpl{}
}

func (s *TaskServiceImpl) CreateTask(db *gorm.DB, task models.Task) error {
	return db.Create(&task).Error
}

func (s *TaskServiceImpl) GetTaskByID(db *gorm.DB, id uuid.UUID) (models.Task, error) {
	var task models.Task
	result := db.Where("id = ?", id).First(&task)
	return task, result.Error
}

func (s *TaskServiceImpl) GetTasks(db *gorm.DB) ([]models.Task, error) {
	var tasks []models.Task
	result := db.Find(&tasks)
	return tasks, result.Error
}

func (s *TaskServiceImpl) GetTasksPaginated(db *gorm.DB, sortBy, order, page, pageSize string) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64

	allowedSort := map[string]bool{"created_at": true, "updated_at": true, "title": true}
	if !allowedSort[sortBy] {
		sortBy = "created_at"
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	p := 1
	ps := 10
	if v, err := strconv.Atoi(page); err == nil && v > 0 {
		p = v
	}
	if v, err := strconv.Atoi(pageSize); err == nil && v > 0 && v <= 100 {
		ps = v
	}
	offset := (p - 1) * ps
	db.Model(&models.Task{}).Count(&total)
	result := db.Order(sortBy + " " + order).Offset(offset).Limit(ps).Find(&tasks)
	return tasks, total, result.Error
}

func (s *TaskServiceImpl) UpdateTask(db *gorm.DB, id uuid.UUID, updated models.Task) error {
	return db.Model(&models.Task{}).Where("id = ?", id).Updates(updated).Error
}

func (s *TaskServiceImpl) DeleteTask(db *gorm.DB, id uuid.UUID) error {
	return db.Delete(&models.Task{}, "id = ?", id).Error
}
