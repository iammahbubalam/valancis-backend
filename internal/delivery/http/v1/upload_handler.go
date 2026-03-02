package v1

import (
	"log"
	"net/http"
	"path/filepath"
	"valancis-backend/pkg/storage"
	"valancis-backend/pkg/utils"
	"strings"
)

var (
	// L9: Configurable allowed types - move to config if needed to support more types
	allowedMimeTypes = map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	allowedExtensions = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".gif":  true,
	}
)

type UploadHandler struct {
	storage       *storage.R2Storage
	maxUploadSize int64
}

func NewUploadHandler(s *storage.R2Storage, maxUploadSizeMB int64) *UploadHandler {
	return &UploadHandler{
		storage:       s,
		maxUploadSize: maxUploadSizeMB << 20, // Convert MB to bytes
	}
}

func (h *UploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Multipart Form with configurable limit
	err := r.ParseMultipartForm(h.maxUploadSize)
	if err != nil {
		log.Printf("Upload Error: ParseMultipartForm failed: %v", err)
		utils.WriteError(w, http.StatusBadRequest, "File too large or invalid format")
		return
	}

	// 2. Get File
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Upload Error: FormFile failed: %v", err)
		utils.WriteError(w, http.StatusBadRequest, "Invalid file")
		return
	}
	defer file.Close()

	// 3. Validate MIME Type
	contentType := header.Header.Get("Content-Type")
	log.Printf("Upload Debug: Filename=%s, ContentType=%s, Size=%d", header.Filename, contentType, header.Size)
	if !allowedMimeTypes[contentType] {
		log.Printf("Upload Error: Invalid MIME type: %s", contentType)
		utils.WriteError(w, http.StatusBadRequest, "Invalid file type. Allowed: JPEG, PNG, WebP, GIF")
		return
	}

	// 4. Validate File Extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		log.Printf("Upload Error: Invalid extension: %s", ext)
		utils.WriteError(w, http.StatusBadRequest, "Invalid file extension")
		return
	}

	// 5. Process Image (Resize + WebP)
	processedData, newContentType, err := utils.ProcessImage(file, header.Filename)
	if err != nil {
		log.Printf("Image Processing Error: %v", err)
		utils.WriteError(w, http.StatusInternalServerError, "Failed to process image")
		return
	}

	// 6. Upload Processed Buffer to R2
	url, err := h.storage.UploadBuffer(r.Context(), processedData, newContentType)
	if err != nil {
		log.Printf("R2 Upload Error: %v", err)
		utils.WriteError(w, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	// 6. Return URL
	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"url": url,
	})
}
