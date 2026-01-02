package models

import (
	"time"

	"gorm.io/gorm"
)

// AttachmentStatus represents the status of an attachment
type AttachmentStatus string

const (
	AttachmentStatusPending    AttachmentStatus = "pending"
	AttachmentStatusProcessing AttachmentStatus = "processing"
	AttachmentStatusCompleted  AttachmentStatus = "completed"
	AttachmentStatusFailed     AttachmentStatus = "failed"
)

// Attachment represents an uploaded file
type Attachment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	UserID      uint             `gorm:"not null;index" json:"user_id"`
	SubtaskID   *uint            `gorm:"index" json:"subtask_id,omitempty"`
	TaskID      *uint            `gorm:"index" json:"task_id,omitempty"`
	Filename    string           `gorm:"not null" json:"filename"`
	FileSize    int64            `gorm:"not null" json:"file_size"`
	MimeType    string           `gorm:"not null" json:"mime_type"`
	FileExt     string           `json:"file_extension"`
	Status      AttachmentStatus `gorm:"not null;default:'pending'" json:"status"`
	StoragePath string           `gorm:"not null" json:"storage_path"`

	// Extracted text content
	TextContent  string `gorm:"type:longtext" json:"text_content,omitempty"`
	TextLength   int    `json:"text_length"`
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Metadata
	Metadata string `gorm:"type:text" json:"metadata,omitempty"` // JSON
}

// SupportedMimeTypes defines supported file types
var SupportedMimeTypes = map[string][]string{
	"document": {
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"text/csv",
	},
	"text": {
		"text/plain",
		"text/markdown",
		"text/html",
		"application/json",
		"application/xml",
	},
	"image": {
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/bmp",
		"image/webp",
		"image/svg+xml",
	},
}

// MaxFileSize defines the maximum file size (20MB)
const MaxFileSize = 20 * 1024 * 1024

// MaxTextLength defines the maximum extracted text length
const MaxTextLength = 50000
