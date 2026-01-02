package repositories

import (
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"gorm.io/gorm"
)

type WorkspaceRepository struct {
	db *gorm.DB
}

func NewWorkspaceRepository(db *gorm.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

func (r *WorkspaceRepository) Create(workspace *models.Workspace) error {
	return r.db.Create(workspace).Error
}

func (r *WorkspaceRepository) Update(workspace *models.Workspace) error {
	return r.db.Save(workspace).Error
}

func (r *WorkspaceRepository) GetByID(id uint) (*models.Workspace, error) {
	var workspace models.Workspace
	if err := r.db.First(&workspace, id).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (r *WorkspaceRepository) ListByUser(userID uint) ([]*models.Workspace, error) {
	var workspaces []*models.Workspace
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&workspaces).Error
	return workspaces, err
}

func (r *WorkspaceRepository) Delete(id uint) error {
	return r.db.Delete(&models.Workspace{}, id).Error
}
