package attachment

import (
	"fmt"
	"strings"

	"github.com/xcode-ai/xgent-go/internal/storage/models"
)

// DocumentParser handles document text extraction
type DocumentParser struct{}

// NewDocumentParser creates a new document parser
func NewDocumentParser() *DocumentParser {
	return &DocumentParser{}
}

// Parse extracts text from a file based on its MIME type
func (p *DocumentParser) Parse(data []byte, mimeType string) (string, error) {
	switch mimeType {
	case "text/plain", "text/markdown", "text/html", "application/json", "application/xml":
		return p.parseText(data)
	case "application/pdf":
		return p.parsePDF(data)
	case "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return p.parseWord(data)
	case "image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp":
		return p.parseImage(data)
	default:
		return "", fmt.Errorf("unsupported MIME type: %s", mimeType)
	}
}

// parseText extracts text from plain text files
func (p *DocumentParser) parseText(data []byte) (string, error) {
	text := string(data)
	if len(text) > models.MaxTextLength {
		text = text[:models.MaxTextLength]
	}
	return text, nil
}

// parsePDF extracts text from PDF files
// TODO: Implement actual PDF parsing using a library like pdfcpu or unidoc
func (p *DocumentParser) parsePDF(data []byte) (string, error) {
	// Placeholder implementation
	// In production, use: github.com/ledongthuc/pdf or github.com/unidoc/unipdf
	return "[PDF content - parser not implemented yet]", nil
}

// parseWord extracts text from Word documents
// TODO: Implement actual Word parsing using a library
func (p *DocumentParser) parseWord(data []byte) (string, error) {
	// Placeholder implementation
	// In production, use: github.com/fumiama/go-docx or similar
	return "[Word document content - parser not implemented yet]", nil
}

// parseImage processes image files
// TODO: Implement OCR or image description
func (p *DocumentParser) parseImage(data []byte) (string, error) {
	// Placeholder implementation
	// In production, integrate with OCR service or vision API
	return "[Image file - OCR not implemented yet]", nil
}

// IsSupportedMimeType checks if a MIME type is supported
func IsSupportedMimeType(mimeType string) bool {
	for _, types := range models.SupportedMimeTypes {
		for _, t := range types {
			if t == mimeType {
				return true
			}
		}
	}
	return false
}

// GetFileExtension extracts file extension from filename
func GetFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return "." + parts[len(parts)-1]
	}
	return ""
}
