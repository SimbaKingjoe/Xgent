package attachment

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

// Service handles attachment business logic
type Service struct {
	storage    *storage.Storage
	parser     *DocumentParser
	uploadDir  string
	logger     *zap.Logger
}

// NewService creates a new attachment service
func NewService(storage *storage.Storage, uploadDir string, logger *zap.Logger) *Service {
	// Ensure upload directory exists
	os.MkdirAll(uploadDir, 0755)
	
	return &Service{
		storage:   storage,
		parser:    NewDocumentParser(),
		uploadDir: uploadDir,
		logger:    logger,
	}
}

// Upload handles file upload
func (s *Service) Upload(file *multipart.FileHeader, userID uint) (*models.Attachment, error) {
	// Validate file size
	if file.Size > models.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", models.MaxFileSize)
	}

	// Detect MIME type
	mimeType := file.Header.Get("Content-Type")
	if !IsSupportedMimeType(mimeType) {
		return nil, fmt.Errorf("unsupported file type: %s", mimeType)
	}

	// Generate unique filename
	ext := GetFileExtension(file.Filename)
	uniqueFilename := uuid.New().String() + ext
	storagePath := filepath.Join(s.uploadDir, uniqueFilename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(storagePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Create attachment record
	attachment := &models.Attachment{
		UserID:      userID,
		Filename:    file.Filename,
		FileSize:    file.Size,
		MimeType:    mimeType,
		FileExt:     ext,
		Status:      models.AttachmentStatusPending,
		StoragePath: storagePath,
	}

	if err := s.storage.Attachments().Create(attachment); err != nil {
		os.Remove(storagePath)
		return nil, fmt.Errorf("failed to create attachment record: %w", err)
	}

	// Process file asynchronously
	go s.processFile(attachment.ID)

	return attachment, nil
}

// processFile extracts text from uploaded file
func (s *Service) processFile(attachmentID uint) {
	attachment, err := s.storage.Attachments().GetByID(attachmentID)
	if err != nil {
		s.logger.Error("Failed to get attachment", zap.Error(err))
		return
	}

	// Update status to processing
	attachment.Status = models.AttachmentStatusProcessing
	s.storage.Attachments().Update(attachment)

	// Read file content
	data, err := os.ReadFile(attachment.StoragePath)
	if err != nil {
		s.logger.Error("Failed to read file", zap.Error(err))
		s.storage.Attachments().UpdateStatus(attachmentID, models.AttachmentStatusFailed, err.Error())
		return
	}

	// Parse file
	text, err := s.parser.Parse(data, attachment.MimeType)
	if err != nil {
		s.logger.Error("Failed to parse file", zap.Error(err))
		s.storage.Attachments().UpdateStatus(attachmentID, models.AttachmentStatusFailed, err.Error())
		return
	}

	// Truncate text if too long
	if len(text) > models.MaxTextLength {
		text = text[:models.MaxTextLength]
	}

	// Update attachment with extracted text
	attachment.TextContent = text
	attachment.TextLength = len(text)
	attachment.Status = models.AttachmentStatusCompleted
	s.storage.Attachments().Update(attachment)

	s.logger.Info("File processed successfully",
		zap.Uint("attachment_id", attachmentID),
		zap.Int("text_length", len(text)),
	)
}

// GetFile retrieves file content
func (s *Service) GetFile(attachmentID uint, userID uint) ([]byte, string, error) {
	attachment, err := s.storage.Attachments().GetByID(attachmentID)
	if err != nil {
		return nil, "", fmt.Errorf("attachment not found")
	}

	// Check ownership
	if attachment.UserID != userID {
		return nil, "", fmt.Errorf("access denied")
	}

	// Read file
	data, err := os.ReadFile(attachment.StoragePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	return data, attachment.Filename, nil
}

// Delete deletes an attachment
func (s *Service) Delete(attachmentID uint, userID uint) error {
	attachment, err := s.storage.Attachments().GetByID(attachmentID)
	if err != nil {
		return fmt.Errorf("attachment not found")
	}

	// Check ownership
	if attachment.UserID != userID {
		return fmt.Errorf("access denied")
	}

	// Delete file from disk
	if err := os.Remove(attachment.StoragePath); err != nil {
		s.logger.Warn("Failed to delete file from disk", zap.Error(err))
	}

	// Delete database record
	return s.storage.Attachments().Delete(attachmentID)
}

// AttachToTask attaches a file to a task
func (s *Service) AttachToTask(attachmentID, taskID, userID uint) error {
	attachment, err := s.storage.Attachments().GetByID(attachmentID)
	if err != nil {
		return fmt.Errorf("attachment not found")
	}

	// Check ownership
	if attachment.UserID != userID {
		return fmt.Errorf("access denied")
	}

	// Update attachment
	attachment.TaskID = &taskID
	return s.storage.Attachments().Update(attachment)
}
