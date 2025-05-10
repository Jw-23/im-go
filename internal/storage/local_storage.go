package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"im-go/internal/config"
	"im-go/internal/imtypes"

	"github.com/google/uuid"
)

// LocalStorageService 实现了 imtypes.StorageService 接口。
type LocalStorageService struct {
	basePath string // 本地存储的基础路径，例如 "./uploads"
	baseURL  string // 用于构建文件访问 URL 的基础 URL，例如 "/static/uploads"
}

// NewLocalStorageService 创建一个新的 LocalStorageService 实例。
// basePath 是文件存储的根目录。
// baseURL 是文件访问 URL 的前缀。
// 返回类型是 imtypes.StorageService 接口。
func NewLocalStorageService(cfg config.StorageConfig, baseURL string) (imtypes.StorageService, error) {
	// 确保 basePath 存在
	if err := os.MkdirAll(cfg.LocalPath, 0755); err != nil {
		return nil, fmt.Errorf("创建本地存储目录失败 '%s': %w", cfg.LocalPath, err)
	}
	return &LocalStorageService{
		basePath: cfg.LocalPath,
		baseURL:  baseURL, // 通常是 /static/ 或类似，取决于如何提供静态文件
	}, nil
}

// UploadFile 将文件保存到本地文件系统。
func (s *LocalStorageService) UploadFile(ctx context.Context, reader io.Reader, fileSize int64, fileName string, mimeType string) (*imtypes.FileInfo, error) {
	// 生成一个唯一的文件名，保留原始扩展名
	ext := filepath.Ext(fileName)
	if ext == "" {
		// 如果没有扩展名，尝试从 MIME 类型推断
		extensions, _ := mime.ExtensionsByType(mimeType)
		if len(extensions) > 0 {
			ext = extensions[0] // 取第一个常见的扩展名
		}
	}
	uniqueFileName := uuid.New().String() + ext

	// 构造完整的文件路径
	dstPath := filepath.Join(s.basePath, uniqueFileName)

	// 创建目标文件
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("创建目标文件失败 '%s': %w", dstPath, err)
	}
	defer dst.Close()

	// 从 reader 复制数据到目标文件
	written, err := io.Copy(dst, reader)
	if err != nil {
		// 如果复制出错，尝试删除已创建的文件
		os.Remove(dstPath)
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}
	if written != fileSize {
		os.Remove(dstPath)
		return nil, fmt.Errorf("文件大小不匹配: 预期 %d, 实际写入 %d", fileSize, written)
	}

	// 构建文件的可访问 URL
	// 这里假设 baseURL 已经包含了斜杠，或者需要处理一下
	fileURL := strings.TrimSuffix(s.baseURL, "/") + "/" + url.PathEscape(uniqueFileName)

	fileInfo := &imtypes.FileInfo{
		URL:      fileURL,
		Path:     dstPath, // 返回本地路径作为内部标识
		Size:     fileSize,
		MimeType: mimeType,
		FileName: fileName, // 返回原始文件名
	}

	return fileInfo, nil
}

// DeleteFile (可选实现)
// func (s *LocalStorageService) DeleteFile(ctx context.Context, pathOrIdentifier string) error {
// 	 return os.Remove(pathOrIdentifier) // 假设 pathOrIdentifier 是本地文件路径
// }
