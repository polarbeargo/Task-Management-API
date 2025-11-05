package services

import (
	"context"
	"fmt"
	"time"

	"task-manager/backend/internal/cache"
	"task-manager/backend/internal/models"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type CachedTaskService struct {
	taskService   TaskService
	cache         *cache.MultiLevelCache
	warmingActive bool
}

func NewCachedTaskService(taskService TaskService, cacheInstance *cache.MultiLevelCache) *CachedTaskService {
	cts := &CachedTaskService{
		taskService:   taskService,
		cache:         cacheInstance,
		warmingActive: false,
	}

	cts.setupCacheWarming()

	return cts
}

func (s *CachedTaskService) CreateTask(db *gorm.DB, task models.Task) error {
	err := s.taskService.CreateTask(db, task)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("task:%s", task.ID.String())
	s.cache.Set(cacheKey, task, 30*time.Minute)

	userCachePattern := fmt.Sprintf("user_tasks:%s:*", task.UserID.String())
	s.cache.DeletePattern(userCachePattern)

	// Invalidate list caches to ensure new task appears in listings
	s.cache.DeletePattern("tasks_paginated:*")
	s.cache.Delete("all_tasks")

	return nil
}

func (s *CachedTaskService) GetTaskByID(db *gorm.DB, id uuid.UUID) (models.Task, error) {
	cacheKey := fmt.Sprintf("task:%s", id.String())

	var cachedTask models.Task
	err := s.cache.Get(cacheKey, &cachedTask)
	if err == nil {
		return cachedTask, nil
	}

	task, err := s.taskService.GetTaskByID(db, id)
	if err != nil {
		return task, err
	}

	s.cache.Set(cacheKey, task, 30*time.Minute)

	return task, nil
}

func (s *CachedTaskService) GetTasks(db *gorm.DB) ([]models.Task, error) {
	cacheKey := "all_tasks"

	var cachedTasks []models.Task
	err := s.cache.Get(cacheKey, &cachedTasks)
	if err == nil {
		return cachedTasks, nil
	}

	tasks, err := s.taskService.GetTasks(db)
	if err != nil {
		return tasks, err
	}

	s.cache.Set(cacheKey, tasks, 10*time.Minute)

	return tasks, nil
}

func (s *CachedTaskService) GetTasksPaginated(db *gorm.DB, sortBy, order, page, pageSize string) ([]models.Task, int64, error) {
	cacheKey := fmt.Sprintf("tasks_paginated:%s:%s:%s:%s", sortBy, order, page, pageSize)

	var cachedResult struct {
		Tasks []models.Task `json:"tasks"`
		Total int64         `json:"total"`
	}

	err := s.cache.Get(cacheKey, &cachedResult)
	if err == nil {
		return cachedResult.Tasks, cachedResult.Total, nil
	}

	tasks, total, err := s.taskService.GetTasksPaginated(db, sortBy, order, page, pageSize)
	if err != nil {
		return tasks, total, err
	}

	result := struct {
		Tasks []models.Task `json:"tasks"`
		Total int64         `json:"total"`
	}{
		Tasks: tasks,
		Total: total,
	}
	s.cache.Set(cacheKey, result, 5*time.Minute)

	return tasks, total, nil
}

func (s *CachedTaskService) UpdateTask(db *gorm.DB, id uuid.UUID, updated models.Task) error {
	err := s.taskService.UpdateTask(db, id, updated)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("task:%s", id.String())
	s.cache.Delete(cacheKey)

	task, getErr := s.taskService.GetTaskByID(db, id)
	if getErr == nil {
		userCachePattern := fmt.Sprintf("user_tasks:%s:*", task.UserID.String())
		s.cache.DeletePattern(userCachePattern)
	}

	s.cache.DeletePattern("tasks_paginated:*")
	s.cache.Delete("all_tasks")

	return nil
}

func (s *CachedTaskService) DeleteTask(db *gorm.DB, id uuid.UUID) error {
	task, getErr := s.taskService.GetTaskByID(db, id)

	err := s.taskService.DeleteTask(db, id)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("task:%s", id.String())
	s.cache.Delete(cacheKey)

	if getErr == nil {
		userCachePattern := fmt.Sprintf("user_tasks:%s:*", task.UserID.String())
		s.cache.DeletePattern(userCachePattern)
	}

	s.cache.DeletePattern("tasks_paginated:*")
	s.cache.Delete("all_tasks")

	return nil
}

func (s *CachedTaskService) GetTasksByUser(db *gorm.DB, userID uuid.UUID) ([]models.Task, error) {
	cacheKey := fmt.Sprintf("user_tasks:%s", userID.String())

	var cachedTasks []models.Task
	err := s.cache.Get(cacheKey, &cachedTasks)
	if err == nil {
		return cachedTasks, nil
	}

	var tasks []models.Task
	result := db.Where("user_id = ?", userID).Find(&tasks)
	if result.Error != nil {
		return tasks, result.Error
	}

	s.cache.Set(cacheKey, tasks, 15*time.Minute)

	return tasks, nil
}

func (s *CachedTaskService) GetCacheStats() map[string]interface{} {
	return s.cache.Stats()
}

func (s *CachedTaskService) setupCacheWarming() {
	if s.cache == nil {
		return
	}

	warmer := s.cache.GetWarmer()
	if warmer == nil {
		return
	}

	warmer.AddWarmupJob(cache.WarmupJob{
		Key:      "tasks_paginated:created_at:desc:1:10",
		Data:     nil,
		TTL:      5 * time.Minute,
		Priority: 100,
	})

	warmer.AddWarmupJob(cache.WarmupJob{
		Key:      "all_tasks",
		Data:     nil,
		TTL:      10 * time.Minute,
		Priority: 80,
	})
}

func (s *CachedTaskService) StartCacheWarming(ctx context.Context) {
	if s.cache == nil {
		return
	}

	warmer := s.cache.GetWarmer()
	if warmer != nil && !s.warmingActive {
		warmer.Start(ctx)
		s.warmingActive = true
	}
}

func (s *CachedTaskService) StopCacheWarming() {
	if s.cache == nil {
		return
	}

	warmer := s.cache.GetWarmer()
	if warmer != nil && s.warmingActive {
		warmer.Stop()
		s.warmingActive = false
	}
}

func (s *CachedTaskService) WarmCriticalData(ctx context.Context, db *gorm.DB) error {
	if s.cache == nil {
		return nil
	}

	go func() {
		if tasks, err := s.taskService.GetTasks(db); err == nil {
			s.cache.Set("all_tasks", tasks, 10*time.Minute)
		}

		if tasks, total, err := s.taskService.GetTasksPaginated(db, "created_at", "desc", "1", "10"); err == nil {
			result := struct {
				Tasks []models.Task `json:"tasks"`
				Total int64         `json:"total"`
			}{
				Tasks: tasks,
				Total: total,
			}
			s.cache.Set("tasks_paginated:created_at:desc:1:10", result, 5*time.Minute)
		}
	}()

	return nil
}
