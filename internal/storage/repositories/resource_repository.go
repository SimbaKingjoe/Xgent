package repositories

import (
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"gorm.io/gorm"
)

// ResourceRepository handles CRD resource data access
type ResourceRepository struct {
	db *gorm.DB
}

// NewResourceRepository creates a new resource repository
func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

// Create creates a new resource
func (r *ResourceRepository) Create(resource *models.Resource) error {
	return r.db.Create(resource).Error
}

// Update updates a resource
func (r *ResourceRepository) Update(resource *models.Resource) error {
	return r.db.Save(resource).Error
}

// GetByID retrieves a resource by ID
func (r *ResourceRepository) GetByID(id uint) (*models.Resource, error) {
	var resource models.Resource
	if err := r.db.First(&resource, id).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

// GetByName retrieves a resource by name, type and workspace
func (r *ResourceRepository) GetByName(workspaceID uint, name string, resourceType models.ResourceType) (*models.Resource, error) {
	var resource models.Resource
	if err := r.db.Where("workspace_id = ? AND name = ? AND type = ?", workspaceID, name, resourceType).
		First(&resource).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

// List retrieves resources for a workspace
func (r *ResourceRepository) List(workspaceID uint, resourceType models.ResourceType, limit, offset int) ([]*models.Resource, error) {
	var resources []*models.Resource
	query := r.db.Where("workspace_id = ?", workspaceID)
	
	if resourceType != "" {
		query = query.Where("type = ?", resourceType)
	}
	
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&resources).Error
	return resources, err
}

// Delete deletes a resource
func (r *ResourceRepository) Delete(id uint) error {
	return r.db.Delete(&models.Resource{}, id).Error
}

// Exists checks if a resource exists
func (r *ResourceRepository) Exists(workspaceID uint, name string, resourceType models.ResourceType) (bool, error) {
	var count int64
	err := r.db.Model(&models.Resource{}).
		Where("workspace_id = ? AND name = ? AND type = ?", workspaceID, name, resourceType).
		Count(&count).Error
	return count > 0, err
}
