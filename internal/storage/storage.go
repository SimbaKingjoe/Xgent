package storage

import (
	"fmt"

	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"github.com/xcode-ai/xgent-go/internal/storage/repositories"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config contains storage configuration
type Config struct {
	Driver   string // mysql or postgres
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

// Storage manages database access
type Storage struct {
	db     *gorm.DB
	logger *zap.Logger

	users       *repositories.UserRepository
	workspaces  *repositories.WorkspaceRepository
	resources   *repositories.ResourceRepository
	tasks       *repositories.TaskRepository
	sessions    *repositories.SessionRepository
	attachments *repositories.AttachmentRepository
}

// New creates a new storage instance
func New(cfg *Config, log *zap.Logger) (*Storage, error) {
	var dsn string
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		dialector = mysql.Open(dsn)

	case "postgres":
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database)
		dialector = postgres.Open(dsn)

	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	storage := &Storage{
		db:     db,
		logger: log,
	}

	storage.users = repositories.NewUserRepository(db)
	storage.workspaces = repositories.NewWorkspaceRepository(db)
	storage.resources = repositories.NewResourceRepository(db)
	storage.tasks = repositories.NewTaskRepository(db)
	storage.sessions = repositories.NewSessionRepository(db)
	storage.attachments = repositories.NewAttachmentRepository(db)

	return storage, nil
}

// AutoMigrate runs database migrations
func (s *Storage) AutoMigrate() error {
	return s.db.AutoMigrate(
		&models.User{},
		&models.Workspace{},
		&models.Resource{},
		&models.Task{},
		&models.SubTask{},
		&models.TaskLog{},
		&models.Session{},
		&models.Message{},
		&models.APIKey{},
		&models.Attachment{},
	)
}

// DB returns the database instance
func (s *Storage) DB() *gorm.DB {
	return s.db
}

// Users returns the user repository
func (s *Storage) Users() *repositories.UserRepository {
	return s.users
}

// Workspaces returns the workspace repository
func (s *Storage) Workspaces() *repositories.WorkspaceRepository {
	return s.workspaces
}

// Resources returns the resource repository
func (s *Storage) Resources() *repositories.ResourceRepository {
	return s.resources
}

// Tasks returns the task repository
func (s *Storage) Tasks() *repositories.TaskRepository {
	return s.tasks
}

// Sessions returns the session repository
func (s *Storage) Sessions() *repositories.SessionRepository {
	return s.sessions
}

// Attachments returns the attachment repository
func (s *Storage) Attachments() *repositories.AttachmentRepository {
	return s.attachments
}

// Close closes the database connection
func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
