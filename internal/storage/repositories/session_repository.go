package repositories

import (
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"gorm.io/gorm"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(session *models.Session) error {
	return r.db.Create(session).Error
}

func (r *SessionRepository) GetBySessionID(sessionID string) (*models.Session, error) {
	var session models.Session
	if err := r.db.Where("session_id = ?", sessionID).Preload("Messages").First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) ListByUser(userID uint, limit, offset int) ([]*models.Session, error) {
	var sessions []*models.Session
	err := r.db.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error
	return sessions, err
}

func (r *SessionRepository) Delete(sessionID string) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&models.Session{}).Error
}

func (r *SessionRepository) AddMessage(message *models.Message) error {
	return r.db.Create(message).Error
}

func (r *SessionRepository) GetMessages(sessionID string, limit int) ([]*models.Message, error) {
	var messages []*models.Message
	query := r.db.Where("session_id = ?", sessionID).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&messages).Error
	return messages, err
}

func (r *SessionRepository) ClearMessages(sessionID string) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&models.Message{}).Error
}
