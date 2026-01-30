package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

const (
	// MaxFileSize is the maximum file size in bytes (5MB)
	MaxFileSize = 5 * 1024 * 1024
)

// CloudinaryService handles Cloudinary image operations
type CloudinaryService struct {
	cld       *cloudinary.Cloudinary
	cloudName string
}

// NewCloudinaryService creates a new Cloudinary service instance
func NewCloudinaryService(cloudName, apiKey, apiSecret string) (*CloudinaryService, error) {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cloudinary client: %w", err)
	}

	return &CloudinaryService{
		cld:       cld,
		cloudName: cloudName,
	}, nil
}

// UploadProductImage uploads a product image to Cloudinary
// Returns: (secureURL, publicID, error)
func (s *CloudinaryService) UploadProductImage(ctx context.Context, file multipart.File, filename string) (string, string, error) {
	// Read file into memory buffer to avoid seek issues with multipart.File
	// This ensures we can validate and upload reliably
	fileData, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("failed to read file: %w", err)
	}

	if len(fileData) == 0 {
		return "", "", errors.New("file is empty")
	}

	fileSize := len(fileData)
	log.Printf("Uploading to Cloudinary - filename: %s, size: %d bytes, cloudName: %s", filename, fileSize, s.cloudName)

	// Validate file content
	if err := s.validateFileContent(fileData); err != nil {
		return "", "", fmt.Errorf("file validation failed: %w", err)
	}

	// Check file size
	if fileSize > MaxFileSize {
		return "", "", fmt.Errorf("file size exceeds maximum allowed size: %d bytes (max: %d bytes)", fileSize, MaxFileSize)
	}

	// Create a reader from the bytes for Cloudinary
	fileReader := bytes.NewReader(fileData)

	// Upload to Cloudinary with optimization
	resp, err := s.cld.Upload.Upload(ctx, fileReader, uploader.UploadParams{
		Folder:         "flower-supply/products",
		PublicID:       filename,
		ResourceType:   "image",
		Transformation: "c_limit,w_1200,h_1200,q_auto,f_auto",
	})

	// Check for error in response even if err is nil
	if resp != nil && resp.Error.Message != "" {
		log.Printf("ERROR: Cloudinary returned error in response: %s", resp.Error.Message)
		return "", "", fmt.Errorf("Cloudinary upload failed: %s", resp.Error.Message)
	}

	if err != nil {
		log.Printf("ERROR: Cloudinary upload failed with error: %v", err)
		return "", "", fmt.Errorf("failed to upload image to Cloudinary: %w", err)
	}

	// Log full Cloudinary response for debugging
	log.Printf("Cloudinary upload response - PublicID: %s, SecureURL: %s, URL: %s, Format: %s, Bytes: %d, ResourceType: %s, Error: %v, ResponseError: %s",
		resp.PublicID, resp.SecureURL, resp.URL, resp.Format, resp.Bytes, resp.ResourceType, err, resp.Error.Message)

	// Check if response indicates an error (even if err is nil)
	if resp == nil {
		log.Printf("ERROR: Cloudinary returned nil response")
		return "", "", fmt.Errorf("Cloudinary returned nil response - check credentials and configuration")
	}

	if resp.PublicID == "" && resp.SecureURL == "" && resp.URL == "" {
		log.Printf("WARNING: Cloudinary returned empty response - PublicID: '%s', SecureURL: '%s', URL: '%s', ErrorMessage: '%s'",
			resp.PublicID, resp.SecureURL, resp.URL, resp.Error.Message)
		return "", "", fmt.Errorf("Cloudinary returned empty response - check credentials and configuration. Error: %s", resp.Error.Message)
	}

	// Check if we have a secure URL, fallback to regular URL, or construct from PublicID
	var imageURL string
	if resp.SecureURL != "" {
		imageURL = resp.SecureURL
	} else if resp.URL != "" {
		imageURL = resp.URL
	} else if resp.PublicID != "" {
		// Construct secure URL from PublicID if both are empty
		// Note: PublicID already includes folder path when Folder parameter is used
		imageURL = fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/%s", s.cloudName, resp.PublicID)
	} else {
		return "", "", fmt.Errorf("Cloudinary returned empty URL and PublicID")
	}

	if resp.PublicID == "" {
		return "", "", fmt.Errorf("Cloudinary returned empty PublicID")
	}

	return imageURL, resp.PublicID, nil
}

// UploadVariantImage uploads a variant image to Cloudinary
// Returns: (secureURL, publicID, error)
func (s *CloudinaryService) UploadVariantImage(ctx context.Context, file multipart.File, filename string) (string, string, error) {
	// Read file into memory buffer to avoid seek issues with multipart.File
	fileData, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("failed to read file: %w", err)
	}

	if len(fileData) == 0 {
		return "", "", errors.New("file is empty")
	}

	fileSize := len(fileData)
	log.Printf("Uploading variant to Cloudinary - filename: %s, size: %d bytes", filename, fileSize)

	// Validate file content
	if err := s.validateFileContent(fileData); err != nil {
		return "", "", fmt.Errorf("file validation failed: %w", err)
	}

	// Check file size
	if fileSize > MaxFileSize {
		return "", "", fmt.Errorf("file size exceeds maximum allowed size: %d bytes (max: %d bytes)", fileSize, MaxFileSize)
	}

	// Create a reader from the bytes for Cloudinary
	fileReader := bytes.NewReader(fileData)

	// Upload to Cloudinary with optimization
	resp, err := s.cld.Upload.Upload(ctx, fileReader, uploader.UploadParams{
		Folder:         "flower-supply/variants",
		PublicID:       filename,
		Transformation: "c_limit,w_800,h_800,q_auto,f_auto",
		ResourceType:   "image",
	})

	// Check for error in response even if err is nil
	if resp != nil && resp.Error.Message != "" {
		log.Printf("ERROR: Cloudinary returned error in variant response: %s", resp.Error.Message)
		return "", "", fmt.Errorf("Cloudinary variant upload failed: %s", resp.Error.Message)
	}

	if err != nil {
		log.Printf("ERROR: Cloudinary variant upload failed with error: %v", err)
		return "", "", fmt.Errorf("failed to upload variant image to Cloudinary: %w", err)
	}

	// Log upload response for debugging
	log.Printf("Cloudinary variant upload response - PublicID: %s, SecureURL: %s, URL: %s, ResponseError: %s",
		resp.PublicID, resp.SecureURL, resp.URL, resp.Error.Message)

	// Check if response indicates an error (even if err is nil)
	if resp == nil {
		log.Printf("ERROR: Cloudinary returned nil response for variant")
		return "", "", fmt.Errorf("Cloudinary returned nil response - check credentials and configuration")
	}

	if resp.PublicID == "" && resp.SecureURL == "" && resp.URL == "" {
		log.Printf("WARNING: Cloudinary returned empty response for variant upload - PublicID: '%s', SecureURL: '%s', URL: '%s', ErrorMessage: '%s'",
			resp.PublicID, resp.SecureURL, resp.URL, resp.Error.Message)
		return "", "", fmt.Errorf("Cloudinary returned empty response - check credentials and configuration. Error: %s", resp.Error.Message)
	}

	// Check if we have a secure URL, fallback to regular URL, or construct from PublicID
	var imageURL string
	if resp.SecureURL != "" {
		imageURL = resp.SecureURL
	} else if resp.URL != "" {
		imageURL = resp.URL
	} else if resp.PublicID != "" {
		// Construct secure URL from PublicID if both are empty
		// Note: PublicID already includes folder path when Folder parameter is used
		imageURL = fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/%s", s.cloudName, resp.PublicID)
	} else {
		return "", "", fmt.Errorf("Cloudinary returned empty URL and PublicID")
	}

	if resp.PublicID == "" {
		return "", "", fmt.Errorf("Cloudinary returned empty PublicID")
	}

	return imageURL, resp.PublicID, nil
}

// DeleteImage deletes an image from Cloudinary by public ID
func (s *CloudinaryService) DeleteImage(ctx context.Context, publicID string) error {
	if publicID == "" {
		return errors.New("public ID cannot be empty")
	}

	_, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

// validateFileContent validates file type from file data
// Checks file type using magic numbers (not just extension)
func (s *CloudinaryService) validateFileContent(fileData []byte) error {
	if len(fileData) == 0 {
		return errors.New("file is empty")
	}

	// Read first 512 bytes (or all if smaller) to detect content type
	buffer := fileData
	if len(buffer) > 512 {
		buffer = buffer[:512]
	}

	// Detect content type using magic numbers
	contentType := http.DetectContentType(buffer)

	// Allowed MIME types
	allowedTypes := []string{
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	// Check if content type is allowed
	isAllowed := false
	for _, allowed := range allowedTypes {
		if contentType == allowed {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("invalid file type: %s (allowed: JPEG, PNG, WebP)", contentType)
	}

	return nil
}
