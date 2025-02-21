package models

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"blog/global"

	"bufio"
	"encoding/base64"

	"github.com/importcjj/sensitive"
	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CommentModel 评论模型，用于存储文章评论信息
type CommentModel struct {
	MODEL           `json:","`
	SubComments     []*CommentModel `json:"sub_comments" gorm:"foreignKey:ParentCommentID;constraint:OnDelete:CASCADE"`
	ParentCommentID *uint           `json:"parent_comment_id" gorm:"index:idx_parent_article"`
	Content         string          `json:"content"`                                    // 评论内容
	DiggCount       uint            `json:"digg_count"`                                 // 点赞数
	CommentCount    uint            `json:"comment_count"`                              // 子评论数
	ArticleID       string          `json:"article_id" gorm:"index:idx_parent_article"` // 关联的文章ID
	UserID          uint            `json:"user_id"`                                    // 评论用户ID
	User            UserModel       `json:"user" gorm:"foreignKey:UserID"`              // 关联的用户信息
}

type CommentRequest struct {
	SortBy string `json:"sort_by" form:"sort_by" binding:"omitempty,oneof=created_at digg_count comment_count" validate:"omitempty,oneof=created_at digg_count comment_count"`
}

var (
	ErrEmptyContent          = errors.New("评论内容不能为空")
	ErrContentTooLong        = errors.New("评论内容不能超过1000字")
	ErrParentCommentNotExist = errors.New("父评论不存在")
)

var (
	sensitiveFilter *sensitive.Filter
)

// loadSensitiveWordsFromFile 从配置文件加载Base64编码的敏感词
func loadSensitiveWordsFromFile() error {
	filePath := "sensitive_words.txt"

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开敏感词文件失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行
		if line == "" {
			continue
		}

		// Base64解码
		decodedBytes, err := base64.StdEncoding.DecodeString(line)
		if err != nil {
			global.Log.Warn("Base64解码失败，跳过该行",
				zap.String("line", line),
				zap.String("error", err.Error()),
			)
			continue

		}

		// 转换为UTF-8字符串
		decodedStr := string(decodedBytes)
		decodedStr = strings.TrimSpace(decodedStr)

		if decodedStr == "" {
			continue
		}

		// 添加解码后的敏感词
		sensitiveFilter.AddWord(decodedStr)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取敏感词文件出错: %w", err)
	}

	return nil
}

func init() {
	// 敏感词过滤器初始化
	sensitiveFilter = sensitive.New()
	// 从文件加载敏感词
	if err := loadSensitiveWordsFromFile(); err != nil {
		log.Fatalf("加载敏感词失败: %v", err)
	}
}

// filterContent 过滤评论内容
func filterContent(content string) (string, error) {
	if sensitiveFilter == nil {
		// 如果过滤器未初始化，至少返回清理后的HTML
		return bluemonday.UGCPolicy().Sanitize(content), nil
	}

	// 清理HTML
	content = bluemonday.UGCPolicy().Sanitize(content)
	// 过滤敏感词
	content = sensitiveFilter.Replace(content, '*')
	return content, nil
}

// GetArticleCommentsWithTree 获取文章评论树
func GetArticleCommentsWithTree(articleID string) ([]*CommentModel, error) {
	var allComments []*CommentModel
	if err := global.DB.Model(&CommentModel{}).
		Where("article_id = ?", articleID).
		Preload("User").
		Order("created_at DESC").
		Find(&allComments).Error; err != nil {
		return nil, fmt.Errorf("获取评论失败: %w", err)
	}

	return buildCommentTree(allComments), nil
}

// buildCommentTree 将评论列表构建成树形结构
func buildCommentTree(allComments []*CommentModel) []*CommentModel {
	commentMap := make(map[uint]*CommentModel)
	var rootComments []*CommentModel

	// 1. 建立映射关系
	for _, comment := range allComments {
		commentMap[comment.ID] = comment
	}

	// 2. 构建树形结构
	for _, comment := range allComments {
		if comment.ParentCommentID == nil {
			rootComments = append(rootComments, comment)
		} else {
			if parent, exists := commentMap[*comment.ParentCommentID]; exists {
				parent.SubComments = append(parent.SubComments, comment)
			}
		}
	}

	return rootComments
}

// parentCommentExist 检查父评论是否存在
func parentCommentExist(tx *gorm.DB, parentID uint) error {
	var exists bool
	err := tx.Model(&CommentModel{}).
		Select("1").
		Where("id = ?", parentID).
		First(&exists).Error
	if err != nil {
		return ErrParentCommentNotExist
	}
	return nil
}

// parentCommentCountUpdate 更新父评论的评论计数
func parentCommentCountUpdate(tx *gorm.DB, parentID uint) error {
	return tx.Model(&CommentModel{}).
		Where("id = ?", parentID).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).
		Error
}

// commentValidate 验证评论
func commentValidate(comment *CommentModel) error {
	content := strings.TrimSpace(comment.Content)
	if content == "" {
		return ErrEmptyContent
	}
	if len(content) > 1000 {
		return ErrContentTooLong
	}

	if comment.ParentCommentID != nil {
		exists, err := commentExist(*comment.ParentCommentID)
		if err != nil {
			return fmt.Errorf("检查父评论失败: %w", err)
		}
		if !exists {
			return ErrParentCommentNotExist
		}
	}
	return nil
}

// commentValidateAndFilter 验证并过滤评论
func commentValidateAndFilter(comment *CommentModel) error {
	if err := commentValidate(comment); err != nil {
		return err
	}

	filteredContent, err := filterContent(comment.Content)
	if err != nil {
		return err
	}
	comment.Content = filteredContent
	return nil
}

// commentExist 检查评论是否存在
func commentExist(commentID uint) (bool, error) {
	var count int64
	err := global.DB.Model(&CommentModel{}).Where("id = ?", commentID).Count(&count).Error
	return count > 0, err
}

// CommentCreate 创建评论
func CommentCreate(comment *CommentModel) error {
	// 1. 评论内容验证和过滤
	if err := commentValidateAndFilter(comment); err != nil {
		return fmt.Errorf("评论验证失败: %w", err)
	}

	// 2. 事务处理
	return global.DB.Transaction(func(tx *gorm.DB) error {
		// 检查父评论是否存在
		if comment.ParentCommentID != nil {
			if err := parentCommentExist(tx, *comment.ParentCommentID); err != nil {
				return err
			}
		}

		// 创建评论
		if err := tx.Create(comment).Error; err != nil {
			return fmt.Errorf("创建评论失败: %w", err)
		}

		// 更新父评论的评论计数
		if comment.ParentCommentID != nil {
			if err := parentCommentCountUpdate(tx, *comment.ParentCommentID); err != nil {
				return err
			}
		}

		return nil
	})
}

// CommentDelete 删除评论
func CommentDelete(commentID uint, articleID string) error {
	var comment CommentModel
	if err := global.DB.First(&comment, commentID).Error; err != nil {
		return err
	}

	return global.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		updates := map[string]interface{}{"deleted_at": now}

		return tx.Model(&CommentModel{}).
			Where("id = ? OR parent_comment_id = ?", commentID, commentID).
			Where("article_id = ?", articleID).
			Updates(updates).Error
	})
}
