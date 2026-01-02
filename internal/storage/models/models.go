package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Username string `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	Email    string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password string `gorm:"type:varchar(255);not null" json:"-"` // Never expose password in JSON

	GitID     string `json:"git_id,omitempty"`
	GitLogin  string `json:"git_login,omitempty"`
	GitEmail  string `json:"git_email,omitempty"`
	GitToken  string `json:"-"` // Encrypted token
	GitAvatar string `json:"git_avatar,omitempty"`

	Workspaces []Workspace `gorm:"foreignKey:UserID" json:"workspaces,omitempty"`
}

// Workspace represents a user's workspace
type Workspace struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Name        string `gorm:"not null" json:"name"`
	Description string `json:"description"`
	UserID      uint   `gorm:"not null;index" json:"user_id"`

	Resources []Resource `gorm:"foreignKey:WorkspaceID" json:"resources,omitempty"`
	Tasks     []Task     `gorm:"foreignKey:WorkspaceID" json:"tasks,omitempty"`
}

// ResourceType represents the type of CRD resource
type ResourceType string

const (
	ResourceTypeSoul          ResourceType = "Soul"
	ResourceTypeMind          ResourceType = "Mind"
	ResourceTypeCraft         ResourceType = "Craft"
	ResourceTypeRobot         ResourceType = "Robot"
	ResourceTypeTeam          ResourceType = "Team"
	ResourceTypeCollaboration ResourceType = "Collaboration"
)

// Resource represents a CRD resource
type Resource struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	WorkspaceID uint         `gorm:"not null;index" json:"workspace_id"`
	Type        ResourceType `gorm:"not null;index" json:"type"`
	Name        string       `gorm:"not null;index" json:"name"`
	Description string       `json:"description"`
	Spec        string       `gorm:"type:text" json:"spec"` // YAML spec
	Status      string       `gorm:"default:'active'" json:"status"`

	// Metadata
	Labels      string `gorm:"type:text" json:"labels,omitempty"`      // JSON
	Annotations string `gorm:"type:text" json:"annotations,omitempty"` // JSON
}

// TaskStatus represents task execution status
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task represents an execution task
type Task struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	WorkspaceID uint       `gorm:"not null;index" json:"workspace_id"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	Status      TaskStatus `gorm:"not null;index;default:'pending'" json:"status"`

	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description"`
	Prompt      string `gorm:"type:text;not null" json:"prompt"`

	// Execution config
	ResourceType string `json:"resource_type"` // bot or team
	ResourceName string `json:"resource_name"`
	Mode         string `json:"mode,omitempty"` // For team: coordinate, collaborate, route

	// Git integration
	GitURL     string `json:"git_url,omitempty"`
	BranchName string `json:"branch_name,omitempty"`

	// Results
	Result    string `gorm:"type:longtext" json:"result,omitempty"`
	Error     string `gorm:"type:text" json:"error,omitempty"`
	Progress  int    `gorm:"default:0" json:"progress"`
	EventLogs string `gorm:"type:longtext" json:"event_logs,omitempty"`

	// Execution metadata
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int64      `json:"duration,omitempty"` // milliseconds

	// Relations
	SubTasks []SubTask `gorm:"foreignKey:TaskID" json:"sub_tasks,omitempty"`
	Logs     []TaskLog `gorm:"foreignKey:TaskID" json:"logs,omitempty"`
}

// SubTask represents a subtask of a task
type SubTask struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	TaskID   uint       `gorm:"not null;index" json:"task_id"`
	Status   TaskStatus `gorm:"not null;default:'pending'" json:"status"`
	Title    string     `gorm:"not null" json:"title"`
	AgentID  string     `json:"agent_id,omitempty"`
	Progress int        `gorm:"default:0" json:"progress"`
	Result   string     `gorm:"type:text" json:"result,omitempty"`
	Error    string     `gorm:"type:text" json:"error,omitempty"`
}

// TaskLog represents execution logs
type TaskLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	TaskID    uint   `gorm:"not null;index" json:"task_id"`
	Level     string `gorm:"not null" json:"level"` // info, warning, error
	Message   string `gorm:"type:text;not null" json:"message"`
	EventType string `json:"event_type,omitempty"`                // agent_start, tool_call, etc.
	Metadata  string `gorm:"type:text" json:"metadata,omitempty"` // JSON
}

// Session represents an agent session
type Session struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	SessionID string `gorm:"type:varchar(255);uniqueIndex;not null" json:"session_id"`
	UserID    uint   `gorm:"not null;index" json:"user_id"`
	AgentID   string `json:"agent_id,omitempty"`

	Messages []Message `gorm:"foreignKey:SessionID;references:SessionID" json:"messages,omitempty"`
}

// Message represents a conversation message
type Message struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	SessionID string `gorm:"not null;index" json:"session_id"`
	Role      string `gorm:"not null" json:"role"` // system, user, assistant, tool
	Content   string `gorm:"type:longtext;not null" json:"content"`
	Name      string `json:"name,omitempty"`

	// Tool call information
	ToolCalls string `gorm:"type:text" json:"tool_calls,omitempty"` // JSON
	ToolID    string `json:"tool_id,omitempty"`
	Metadata  string `gorm:"type:text" json:"metadata,omitempty"` // JSON
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	UserID      uint       `gorm:"not null;index" json:"user_id"`
	Name        string     `gorm:"not null" json:"name"`
	Key         string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"` // Hashed
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions string     `gorm:"type:text" json:"permissions,omitempty"` // JSON
}

// ProgressCallback is called to report task execution progress
type ProgressCallback func(taskID uint, progress int, status TaskStatus, message string, metadata map[string]interface{})
