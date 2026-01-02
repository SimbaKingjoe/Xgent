package orchestrator

import (
	"fmt"

	"github.com/xcode-ai/xgent-go/internal/executor"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

// Config contains orchestrator configuration
type Config struct {
	Workers      int
	QueueSize    int
	WorkspaceDir string
}

// Orchestrator manages task execution
type Orchestrator struct {
	config   *Config
	storage  *storage.Storage
	logger   *zap.Logger
	queue    *TaskQueue
	executor *executor.AgnoExecutor
}

// New creates a new orchestrator
func New(cfg *Config, storage *storage.Storage, logger *zap.Logger) *Orchestrator {
	return &Orchestrator{
		config:   cfg,
		storage:  storage,
		logger:   logger,
		queue:    NewTaskQueue(cfg.Workers),
		executor: executor.NewAgnoExecutor(storage, logger),
	}
}

// Start starts the orchestrator
func (o *Orchestrator) Start() error {
	o.logger.Info("Starting orchestrator", zap.Int("workers", o.config.Workers))
	o.queue.Start(o.executor)
	return nil
}

// Stop stops the orchestrator
func (o *Orchestrator) Stop() error {
	o.logger.Info("Stopping orchestrator")
	o.queue.Stop()
	return nil
}

// SubmitTask submits a task for execution
func (o *Orchestrator) SubmitTask(task *models.Task, callback ProgressCallback) error {
	o.logger.Info("Submitting task",
		zap.Uint("task_id", task.ID),
		zap.String("title", task.Title),
	)

	if err := o.queue.Enqueue(task, callback); err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	return nil
}

// CancelTask cancels a running task
func (o *Orchestrator) CancelTask(taskID uint) error {
	return o.queue.Cancel(taskID)
}

// GetActiveTasks returns all active tasks
func (o *Orchestrator) GetActiveTasks() []*TaskItem {
	return o.queue.GetActive()
}
