package repositories

import (
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"gorm.io/gorm"
)

// AttachmentRepository handles attachment data access
type AttachmentRepository struct {
	db *gorm.DB
}

// NewAttachmentRepository creates a new attachment repository
func NewAttachmentRepository(db *gorm.DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

// Create creates a new attachment
func (r *AttachmentRepository) Create(attachment *models.Attachment) error {
	return r.db.Create(attachment).Error
}

// Update updates an attachment
func (r *AttachmentRepository) Update(attachment *models.Attachment) error {
	return r.db.Save(attachment).Error
}

// GetByID retrieves an attachment by ID
func (r *AttachmentRepository) GetByID(id uint) (*models.Attachment, error) {
	var attachment models.Attachment
	if err := r.db.First(&attachment, id).Error; err != nil {
		return nil, err
	}
	return &attachment, nil
}

// ListByUser retrieves attachments for a user
func (r *AttachmentRepository) ListByUser(userID uint, limit, offset int) ([]*models.Attachment, error) {
	var attachments []*models.Attachment
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&attachments).Error
	return attachments, err
}

// ListByTask retrieves attachments for a task
func (r *AttachmentRepository) ListByTask(taskID uint) ([]*models.Attachment, error) {
	var attachments []*models.Attachment
	err := r.db.Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&attachments).Error
	return attachments, err
}

// ListBySubtask retrieves attachments for a subtask
func (r *AttachmentRepository) ListBySubtask(subtaskID uint) ([]*models.Attachment, error) {
	var attachments []*models.Attachment
	err := r.db.Where("subtask_id = ?", subtaskID).
		Order("created_at ASC").
		Find(&attachments).Error
	return attachments, err
}

// Delete deletes an attachment
func (r *AttachmentRepository) Delete(id uint) error {
	return r.db.Delete(&models.Attachment{}, id).Error
}

// UpdateStatus updates attachment status
func (r *AttachmentRepository) UpdateStatus(id uint, status models.AttachmentStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	return r.db.Model(&models.Attachment{}).Where("id = ?", id).Updates(updates).Error
}
