package models

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"blog/global"
	"blog/utils"

	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 存储类型常量
const (
	LocalStorage  = "local"  // 本地存储
	OnlineStorage = "online" // 在线存储
)

// WhiteList 定义允许上传的图片格式
var WhiteList = []string{
	"jpg", "png", "jpeg", "ico",
	"tiff", "gif", "svg", "webp",
}

// ImageModel 图片模型
type ImageModel struct {
	MODEL
	Path string `json:"path" gorm:"comment:图片路径"`
	Hash string `json:"hash" gorm:"uniqueIndex:idx_hash,length:32;comment:图片哈希值"`
	Name string `json:"name" gorm:"comment:图片名称"`
	Type string `json:"type" gorm:"comment:存储类型;"`
	Size int64  `json:"size" gorm:"comment:图片大小"`
}

// UploadResponse 定义上传响应结构
type UploadResponse struct {
	FileName  string `json:"file_name"`      // 文件名
	IsSuccess bool   `json:"is_success"`     // 是否上传成功
	Msg       string `json:"msg"`            // 响应信息
	Size      int64  `json:"size,omitempty"` // 文件大小
	Hash      string `json:"hash,omitempty"` // 文件哈希值
}

// imageValidate 图片验证函数
func (im *ImageModel) imageValidate(file *multipart.FileHeader) error {
	if file == nil {
		return fmt.Errorf("文件不能为空")
	}

	// 验证文件格式
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" || !utils.InList(ext[1:], WhiteList) {
		return fmt.Errorf("不支持的文件格式: %s", ext)
	}

	// 验证文件大小
	sizeMB := float64(file.Size) / float64(1024*1024)
	if sizeMB >= float64(global.Config.Upload.Size) {
		return fmt.Errorf("图片大小超过设定,当前大小为:%.2fMB,设定大小为:%dMB",
			sizeMB, global.Config.Upload.Size)
	}
	return nil
}

// Upload 文件上传主函数
func (im *ImageModel) Upload(file *multipart.FileHeader) (res UploadResponse) {
	// 1. 验证图片
	if err := im.imageValidate(file); err != nil {
		res.Msg = err.Error()
		return
	}

	// 2. 读取文件内容
	byteData, err := im.readFileContent(file)
	if err != nil {
		res.Msg = err.Error()
		return
	}

	// 3. 计算并检查文件哈希值是否重复
	imageHash := utils.Md5(byteData)
	if existingImage, exists := im.checkDuplicate(imageHash); exists {
		return existingImage
	}

	// 4. 处理文件上传（本地和腾讯云）
	// 生成本地文件路径
	basePath := global.Config.Upload.Path
	fileName := file.Filename
	localFilePath := filepath.Join("/", basePath, fileName)

	// 确保目录存在
	uploadDir := filepath.Join(basePath)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		global.Log.Error("创建目录失败", zap.String("error", err.Error()))
		res.Msg = "创建上传目录失败"
		return
	}

	// 写入本地文件
	if err := os.WriteFile(filepath.Join(uploadDir, fileName), byteData, 0644); err != nil {
		global.Log.Error("写入文件失败", zap.String("error", err.Error()))
		res.Msg = "保存文件失败"
		return
	}

	// 尝试上传到腾讯云
	cosFilePath, err := im.uploadToTencentCOS(file, byteData)
	var finalPath string
	var storageType string

	if err != nil {
		// 腾讯云上传失败，使用本地存储
		global.Log.Warn("上传到腾讯云失败，将使用本地存储",
			zap.String("error", err.Error()),
			zap.String("localPath", localFilePath),
		)
		finalPath = localFilePath
		storageType = LocalStorage
	} else {
		// 腾讯云上传成功
		finalPath = cosFilePath
		storageType = OnlineStorage
	}

	// 5. 保存记录到数据库
	if err := im.imageRecordSave(file, finalPath, storageType, imageHash); err != nil {
		if storageType == LocalStorage {
			// 如果是本地存储且数据库保存失败，删除已上传的文件
			if err := os.Remove(filepath.Join(uploadDir, fileName)); err != nil {
				global.Log.Error("删除文件失败", zap.String("error", err.Error()))
			}
		}
		res.Msg = "保存图片记录失败"
		return
	}

	return UploadResponse{
		FileName:  finalPath,
		IsSuccess: true,
		Msg:       "上传成功",
		Size:      file.Size,
		Hash:      imageHash,
	}
}

// readFileContent 读取文件内容
func (im *ImageModel) readFileContent(file *multipart.FileHeader) ([]byte, error) {
	fileObj, err := file.Open()
	if err != nil {
		global.Log.Error("打开文件失败", zap.String("error", err.Error()))
		return nil, fmt.Errorf("无法打开文件")
	}

	defer fileObj.Close()

	return io.ReadAll(fileObj)
}

// checkDuplicate 检查重复文件
func (im *ImageModel) checkDuplicate(hash string) (UploadResponse, bool) {
	var existImage ImageModel
	if err := global.DB.Where("hash = ?", hash).First(&existImage).Error; err == nil {
		return UploadResponse{
			FileName:  existImage.Path,
			IsSuccess: true,
			Msg:       "图片已存在",
			Hash:      hash,
		}, true
	}
	return UploadResponse{}, false
}

// uploadToTencentCOS 上传文件到腾讯云COS
func (im *ImageModel) uploadToTencentCOS(file *multipart.FileHeader, data []byte) (string, error) {
	// 获取腾讯云配置
	cosConfig := global.Config.TencentCos

	// 创建COS客户端
	u, _ := url.Parse(cosConfig.BucketURL)
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cosConfig.SecretID,
			SecretKey: cosConfig.SecretKey,
		},
	})

	// 生成文件名，使用时间戳避免重名
	ext := filepath.Ext(file.Filename)                          // 获取文件扩展名
	fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext) // 使用时间戳作为文件名

	// 上传对象
	r := bytes.NewReader(data)
	_, err := client.Object.Put(context.Background(), fileName, r, nil)
	if err != nil {
		global.Log.Error("上传到腾讯云失败", zap.String("error", err.Error()))
		return "", fmt.Errorf("上传到腾讯云失败: %v", err)
	}

	// 使用存储桶URL
	return fmt.Sprintf("%s/%s", strings.TrimRight(cosConfig.BucketURL, "/"), fileName), nil
}

// imageRecordSave 保存图片记录到数据库
func (im *ImageModel) imageRecordSave(file *multipart.FileHeader, filePath, fileType, hash string) error {
	im.Hash = hash
	im.Path = filePath
	im.Name = file.Filename
	im.Type = fileType
	im.Size = file.Size

	return global.DB.Create(im).Error
}

// BeforeDelete 删除钩子：在删除数据库记录前删除对应的文件
func (im *ImageModel) BeforeDelete(tx *gorm.DB) error {
	switch im.Type {
	case LocalStorage:
		filePath := im.Path[1:] // 移除路径开头的'/'
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			global.Log.Error("删除本地文件失败",
				zap.String("path", im.Path),
				zap.String("error", err.Error()),
			)
			return fmt.Errorf("删除文件失败: %v", err)
		}
	case OnlineStorage:
		// 删除腾讯云COS中的文件
		cosConfig := global.Config.TencentCos
		u, _ := url.Parse(cosConfig.BucketURL)
		b := &cos.BaseURL{BucketURL: u}
		client := cos.NewClient(b, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  cosConfig.SecretID,
				SecretKey: cosConfig.SecretKey,
			},
		})

		// 从完整URL中提取对象键
		objectKey := strings.TrimPrefix(im.Path, cosConfig.BucketURL+"/")

		// 确保 objectKey 不为空
		if objectKey == "" {
			global.Log.Error("无法从路径中提取对象键",
				zap.String("path", im.Path),
			)
			return fmt.Errorf("无效的文件路径")
		}

		// 删除对象
		_, err := client.Object.Delete(context.Background(), objectKey)
		if err != nil {
			// 如果对象不存在，不返回错误
			if strings.Contains(err.Error(), "NoSuchKey") {
				global.Log.Warn("腾讯云文件不存在",
					zap.String("path", im.Path),
				)
				return nil
			}
			global.Log.Error("删除腾讯云文件失败",
				zap.String("path", im.Path),
				zap.String("objectKey", objectKey),
				zap.String("error", err.Error()),
			)
			return fmt.Errorf("删除腾讯云文件失败: %v", err)
		}
	}
	return nil
}
