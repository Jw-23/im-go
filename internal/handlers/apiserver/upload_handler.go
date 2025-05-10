// internal/handlers/apiserver/upload_handler.go
package apiserver

import (
	"fmt"
	"im-go/internal/config"
	"im-go/internal/imtypes" // Use imtypes.StorageService
	"log"
	"net/http"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB default max memory for multipart forms
)

// UploadHandler 封装了文件上传相关的 HTTP 处理器方法。
type UploadHandler struct {
	storageService imtypes.StorageService // Use the interface from imtypes
	cfg            config.StorageConfig   // Storage config for max size check
}

// NewUploadHandler 创建一个新的 UploadHandler 实例。
func NewUploadHandler(storageService imtypes.StorageService, cfg config.StorageConfig) *UploadHandler {
	return &UploadHandler{
		storageService: storageService,
		cfg:            cfg,
	}
}

// UploadFileHandler 处理文件上传请求。
func (h *UploadHandler) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 限制请求体大小 (可选，但推荐)
	maxUploadSize := h.cfg.MaxFileSizeMB << 20 // Convert MB to bytes
	if maxUploadSize <= 0 {
		maxUploadSize = defaultMaxMemory // Use a sensible default if not configured
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	// 2. 解析 multipart form
	// Max memory is used to store the non-file parts of the form in memory.
	// File parts are typically streamed to disk or other storage.
	if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
		if err.Error() == "http: request body too large" {
			msg := fmt.Sprintf("上传文件过大，最大允许 %d MB", maxUploadSize>>20)
			writeJSONError(w, msg, http.StatusRequestEntityTooLarge)
		} else {
			writeJSONError(w, fmt.Sprintf("解析表单失败: %v", err), http.StatusBadRequest)
		}
		return
	}

	// 3. 获取文件 "file" 是表单中文件的 key
	file, handler, err := r.FormFile("file") // "file" is the key in the multipart form
	if err != nil {
		if err == http.ErrMissingFile {
			writeJSONError(w, "请求中缺少 'file' 字段", http.StatusBadRequest)
		} else {
			writeJSONError(w, fmt.Sprintf("获取文件失败: %v", err), http.StatusBadRequest)
		}
		return
	}
	defer file.Close()

	// 4. 检查文件类型 (可选，但推荐)
	mimeType := handler.Header.Get("Content-Type")
	// TODO: Add logic to validate mimeType against allowed types if needed
	// e.g., allow only images, pdf, etc.
	log.Printf("收到上传文件: 名称=%s, 大小=%d, 类型=%s", handler.Filename, handler.Size, mimeType)

	// 5. 检查文件大小 (再次确认，因为 MaxBytesReader 是针对整个请求体的)
	if handler.Size > maxUploadSize {
		msg := fmt.Sprintf("上传文件过大，最大允许 %d MB", maxUploadSize>>20)
		writeJSONError(w, msg, http.StatusRequestEntityTooLarge)
		return
	}

	// 6. 调用存储服务上传文件
	fileInfo, err := h.storageService.UploadFile(r.Context(), file, handler.Size, handler.Filename, mimeType)
	if err != nil {
		log.Printf("存储文件失败: %v", err) // Log the detailed error
		writeJSONError(w, "存储文件失败", http.StatusInternalServerError)
		return
	}

	// 7. 返回成功响应，包含文件信息
	writeJSONResponse(w, http.StatusOK, fileInfo) // FileInfo struct is already JSON-ready
}
