package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type JobType string

const (
	JobTypeEmailNotification JobType = "email_notification"
	JobTypeTaskReminder      JobType = "task_reminder"
	JobTypeDataExport        JobType = "data_export"
	JobTypeCleanup           JobType = "cleanup"
)

type Job struct {
	ID        string                 `json:"id"`
	Type      JobType                `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Attempts  int                    `json:"attempts"`
	MaxTries  int                    `json:"max_tries"`
	CreatedAt time.Time              `json:"created_at"`
	ProcessAt time.Time              `json:"process_at"`
}

type JobHandler func(ctx context.Context, job *Job) error

type Worker struct {
	client   *redis.Client
	handlers map[JobType]JobHandler
	queues   []string
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

type WorkerConfig struct {
	RedisClient  *redis.Client
	Concurrency  int
	PollInterval time.Duration
	Queues       []string
}

func NewWorker(config WorkerConfig) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		client:   config.RedisClient,
		handlers: make(map[JobType]JobHandler),
		queues:   config.Queues,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (w *Worker) RegisterHandler(jobType JobType, handler JobHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[jobType] = handler
}

func (w *Worker) Start(concurrency int) {
	log.Printf("Starting worker with %d goroutines", concurrency)

	for i := 0; i < concurrency; i++ {
		w.wg.Add(1)
		go w.workerLoop()
	}
}

func (w *Worker) Stop() {
	log.Println("Stopping worker...")
	w.cancel()
	w.wg.Wait()
	log.Println("Worker stopped")
}

func (w *Worker) workerLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			if err := w.processNextJob(); err != nil {
				log.Printf("Error processing job: %v", err)
				time.Sleep(time.Second) 
			}
		}
	}
}

func (w *Worker) processNextJob() error {
	result, err := w.client.BLPop(w.ctx, 5*time.Second, w.queues...).Result()
	if err != nil {
		if err == redis.Nil {
			return nil 
		}
		return fmt.Errorf("failed to pop job: %w", err)
	}

	if len(result) < 2 {
		return fmt.Errorf("invalid job result")
	}

	queue := result[0]
	jobData := result[1]

	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	if time.Now().Before(job.ProcessAt) {
		return w.requeueJob(queue, &job)
	}

	return w.executeJob(&job)
}

func (w *Worker) executeJob(job *Job) error {
	w.mu.RLock()
	handler, exists := w.handlers[job.Type]
	w.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for job type: %s", job.Type)
	}

	log.Printf("Processing job %s of type %s", job.ID, job.Type)

	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	err := handler(ctx, job)
	if err != nil {
		job.Attempts++
		if job.Attempts < job.MaxTries {
			log.Printf("Job %s failed (attempt %d/%d), retrying: %v",
				job.ID, job.Attempts, job.MaxTries, err)
			return w.retryJob(job)
		}

		log.Printf("Job %s failed permanently after %d attempts: %v",
			job.ID, job.Attempts, err)
		return w.moveToDeadQueue(job, err)
	}

	log.Printf("Job %s completed successfully", job.ID)
	return nil
}

func (w *Worker) retryJob(job *Job) error {
	delay := time.Duration(1<<job.Attempts) * time.Minute
	job.ProcessAt = time.Now().Add(delay)

	return w.enqueueJob("retry_queue", job)
}

func (w *Worker) requeueJob(queue string, job *Job) error {
	return w.enqueueJob(queue, job)
}

func (w *Worker) enqueueJob(queue string, job *Job) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return w.client.RPush(w.ctx, queue, jobData).Err()
}

func (w *Worker) moveToDeadQueue(job *Job, jobErr error) error {
	deadJob := map[string]interface{}{
		"original_job": job,
		"error":        jobErr.Error(),
		"failed_at":    time.Now(),
	}

	deadJobData, err := json.Marshal(deadJob)
	if err != nil {
		return fmt.Errorf("failed to marshal dead job: %w", err)
	}

	return w.client.RPush(w.ctx, "dead_queue", deadJobData).Err()
}

type JobQueue struct {
	client *redis.Client
}

func NewJobQueue(client *redis.Client) *JobQueue {
	return &JobQueue{client: client}
}

func (q *JobQueue) Enqueue(queue string, jobType JobType, payload map[string]interface{}) error {
	return q.EnqueueAt(queue, jobType, payload, time.Now())
}

func (q *JobQueue) EnqueueAt(queue string, jobType JobType, payload map[string]interface{}, processAt time.Time) error {
	job := &Job{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      jobType,
		Payload:   payload,
		Attempts:  0,
		MaxTries:  3,
		CreatedAt: time.Now(),
		ProcessAt: processAt,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return q.client.RPush(ctx, queue, jobData).Err()
}

func (q *JobQueue) GetQueueSize(queue string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return q.client.LLen(ctx, queue).Result()
}
