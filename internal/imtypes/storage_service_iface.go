// internal/imtypes/storage_service_iface.go
package imtypes

import (
	"context"
	"io"
)

// StorageService 定义了文件存储操作的接口。
// 将接口定义放在 imtypes 中以打破 storage 和 services 之间的循环依赖。
type StorageService interface {
	// UploadFile 将读取器中的内容上传到存储系统。
	// fileName 是原始文件名，用于可能的存储路径或元数据。
	// mimeType 是文件的 MIME 类型。
	// 返回文件的信息 (FileInfo)，包括访问 URL。
	UploadFile(ctx context.Context, reader io.Reader, fileSize int64, fileName string, mimeType string) (*FileInfo, error) // Note: FileInfo is also defined in imtypes

	// DeleteFile 从存储系统中删除文件 (可选实现)。
	// pathOrIdentifier 是 UploadFile 返回的 Path 或其他唯一标识。
	// DeleteFile(ctx context.Context, pathOrIdentifier string) error
}
