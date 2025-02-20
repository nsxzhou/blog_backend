package image

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (i *Image) ImageUpload(c *gin.Context) {
	// 1. 获取上传文件
	form, err := c.MultipartForm()
	if err != nil {
		global.Log.Error("c.MultipartForm() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取MultipartForm失败")
		return
	}

	fileList, ok := form.File["images"]
	if !ok || len(fileList) == 0 {
		res.Error(c, res.InvalidParameter, "参数验证失败")
		return
	}

	// 2. 确保上传目录存在
	if err := ensureUploadDir(global.Config.Upload.Path); err != nil {
		global.Log.Error("ensureUploadDir() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "创建上传目录失败")
		return
	}

	// 3. 并发处理文件上传
	var (
		wg      sync.WaitGroup
		resList []models.UploadResponse
		mutex   sync.Mutex
	)

	for _, file := range fileList {
		wg.Add(1)
		go func(file *multipart.FileHeader) {
			defer wg.Done()

			// 处理单个文件上传
			serviceRes := processFileUpload(c, file)

			mutex.Lock()
			resList = append(resList, serviceRes)
			mutex.Unlock()
		}(file)
	}
	wg.Wait()

	res.Success(c, resList)
}

// 确保上传目录存在
func ensureUploadDir(path string) error {
	if _, err := os.ReadDir(path); err != nil {
		return os.MkdirAll(path, fs.ModePerm)
	}
	return nil
}

// 处理单个文件上传
func processFileUpload(c *gin.Context, file *multipart.FileHeader) models.UploadResponse {
	serviceRes := (&models.ImageModel{}).Upload(file)
	if !serviceRes.IsSuccess {
		return serviceRes
	}

	// 提取纯文件名
	fileName := serviceRes.FileName
	fileName = filepath.Base(strings.TrimPrefix(fileName, "/")) // 移除路径前缀，只保留文件名

	// 使用 filepath.Join 来正确拼接路径
	fullPath := filepath.Join(global.Config.Upload.Path, fileName)

	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		global.Log.Error("c.SaveUploadedFile() failed",
			zap.String("filename", file.Filename),
			zap.String("fullPath", fullPath),
			zap.String("error", err.Error()))
		return models.UploadResponse{
			IsSuccess: false,
			Msg:       "文件保存失败",
		}
	}
	global.Log.Info("图片上传成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	return serviceRes
}
