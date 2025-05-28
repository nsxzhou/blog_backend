package service

import (
	"bufio"
	"encoding/base64"
	"os"
	"strings"
	"sync"

	"github.com/nsxzhou1114/blog-api/internal/logger"
	"go.uber.org/zap"
)

var (
	sensitiveService     *SensitiveService
	sensitiveServiceOnce sync.Once
)

// SensitiveService 敏感词过滤服务
type SensitiveService struct {
	sensitiveWords map[string]struct{} // 敏感词集合，使用map提高查询效率
	logger         *zap.SugaredLogger
}

// NewSensitiveService 创建敏感词过滤服务实例
func NewSensitiveService() *SensitiveService {
	sensitiveServiceOnce.Do(func() {
		sensitiveService = &SensitiveService{
			sensitiveWords: make(map[string]struct{}),
			logger:         logger.GetSugaredLogger(),
		}
		// 从文件加载敏感词
		if err := sensitiveService.loadSensitiveWordsFromFile("sensitive_words.txt"); err != nil {
			sensitiveService.logger.Errorf("加载敏感词失败: %v", err)
		}
	})
	return sensitiveService
}

// loadSensitiveWordsFromFile 从文件加载敏感词
func (s *SensitiveService) loadSensitiveWordsFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Base64解码
		decoded, err := base64.StdEncoding.DecodeString(line)
		if err != nil {
			s.logger.Warnf("Base64解码敏感词失败: %v, 原文: %s", err, line)
			continue
		}

		// 添加到敏感词集合
		word := strings.TrimSpace(string(decoded))
		if word != "" {
			s.sensitiveWords[word] = struct{}{}
		}
	}

	s.logger.Infof("已加载 %d 个敏感词", len(s.sensitiveWords))
	return scanner.Err()
}

// ContainsSensitiveWord 检测文本是否包含敏感词
func (s *SensitiveService) ContainsSensitiveWord(text string) bool {
	if len(s.sensitiveWords) == 0 {
		return false
	}

	// 遍历敏感词集合，检查文本是否包含任一敏感词
	for word := range s.sensitiveWords {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

// GetSensitiveWords 获取文本中包含的敏感词
func (s *SensitiveService) GetSensitiveWords(text string) []string {
	if len(s.sensitiveWords) == 0 {
		return nil
	}

	var found []string
	for word := range s.sensitiveWords {
		if strings.Contains(text, word) {
			found = append(found, word)
		}
	}
	return found
}

// FilterSensitiveWords 过滤文本中的敏感词，将其替换为***
func (s *SensitiveService) FilterSensitiveWords(text string) string {
	if len(s.sensitiveWords) == 0 {
		return text
	}

	result := text
	for word := range s.sensitiveWords {
		if strings.Contains(result, word) {
			replacement := strings.Repeat("*", len([]rune(word)))
			result = strings.ReplaceAll(result, word, replacement)
		}
	}
	return result
}
