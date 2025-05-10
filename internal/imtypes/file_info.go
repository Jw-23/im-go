// internal/imtypes/file_info.go
package imtypes

// FileInfo 包含上传文件的基本信息和访问路径。
type FileInfo struct {
	URL      string `json:"url"`      // 可公开访问的文件 URL
	Path     string `json:"path"`     // 文件在存储系统中的路径或标识符
	Size     int64  `json:"size"`     // 文件大小 (字节)
	MimeType string `json:"mimeType"` // 文件的 MIME 类型
	FileName string `json:"fileName"` // 原始文件名
}
