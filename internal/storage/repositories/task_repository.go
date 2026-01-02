package repositories

import (
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"gorm.io/gorm"
)

// TaskRepository handles task data access
type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create creates a new task
func (r *TaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

// Update updates a task
func (r *TaskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

// GetByID retrieves a task by ID
func (r *TaskRepository) GetByID(id uint) (*models.Task, error) {
	var task models.Task
	if err := r.db.Preload("SubTasks").Preload("Logs").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// List retrieves tasks for a workspace
func (r *TaskRepository) List(workspaceID uint, limit, offset int) ([]*models.Task, error) {
	var tasks []*models.Task
	err := r.db.Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error
	return tasks, err
}

// ListByUser retrieves tasks for a user
func (r *TaskRepository) ListByUser(userID uint, limit, offset int) ([]*models.Task, error) {
	var tasks []*models.Task
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error
	return tasks, err
}

// ListByStatus retrieves tasks by status
func (r *TaskRepository) ListByStatus(workspaceID uint, status models.TaskStatus) ([]*models.Task, error) {
	var tasks []*models.Task
	err := r.db.Where("workspace_id = ? AND status = ?", workspaceID, status).
		Order("created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

// Delete deletes a task
func (r *TaskRepository) Delete(id uint) error {
	return r.db.Delete(&models.Task{}, id).Error
}

// AddLog adds a log entry to a task
func (r *TaskRepository) AddLog(log *models.TaskLog) error {
	return r.db.Create(log).Error
}

// GetLogs retrieves logs for a task
func (r *TaskRepository) GetLogs(taskID uint, limit int) ([]*models.TaskLog, error) {
	var logs []*models.TaskLog
	query := r.db.Where("task_id = ?", taskID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&logs).Error
	return logs, err
}
