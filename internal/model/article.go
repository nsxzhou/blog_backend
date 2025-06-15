package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Article 文章模型
type Article struct {
	Base
	Title         string     `gorm:"type:varchar(255);not null" json:"title"`
	Summary       string     `gorm:"type:text" json:"summary"`
	Status        string     `gorm:"type:varchar(20);not null;default:'draft';index" json:"status"` // 状态: draft published
	ViewCount     int        `gorm:"type:int(11);not null;default:0" json:"view_count"`
	LikeCount     int        `gorm:"type:int(11);not null;default:0" json:"like_count"`
	CommentCount  int        `gorm:"type:int(11);not null;default:0" json:"comment_count"`
	FavoriteCount int        `gorm:"type:int(11);not null;default:0" json:"favorite_count"`
	AuthorID      uint       `gorm:"type:int(11);not null;index" json:"author_id"`
	CategoryID    uint       `gorm:"type:int(11);not null;index" json:"category_id"`
	CoverImage    string     `gorm:"type:varchar(255)" json:"cover_image"`
	WordCount     int        `gorm:"type:int(11);not null;default:0" json:"word_count"`
	AccessType    string     `gorm:"type:varchar(20);not null;default:'public';index" json:"access_type"`
	Password      string     `gorm:"type:varchar(100)" json:"password"`
	IsTop         int        `gorm:"type:tinyint(2);not null;default:0;index" json:"is_top"` // 0=否 1=是
	IsOriginal    int        `gorm:"type:tinyint(2);not null;default:1" json:"is_original"`  // 0=转载 1=原创
	SourceURL     string     `gorm:"type:varchar(255)" json:"source_url"`
	SourceName    string     `gorm:"type:varchar(100)" json:"source_name"`
	PublishedAt   *time.Time `gorm:"index" json:"published_at"`
	ESDocID       string     `gorm:"type:varchar(50);index" json:"es_doc_id"` // Elasticsearch文档ID

	// 关联
	Author   User     `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Category Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Tags     []Tag    `gorm:"many2many:article_tags;" json:"tags,omitempty"`
	TagIDs   []uint   `gorm:"-" json:"tag_ids,omitempty"` // 用于接收标签ID列表
}

// TableName 指定表名
func (Article) TableName() string {
	return "articles"
}

// AfterFind 查询后钩子
func (a *Article) AfterFind(tx *gorm.DB) error {
	// 如果 ESDocID 为空，生成默认值
	if a.ESDocID == "" {
		a.ESDocID = fmt.Sprintf("article_%d", a.ID)
	}
	return nil
}

// ToSearchDocument 转换为搜索文档
func (a *Article) ToSearchDocument(content string) *ESArticle {
	// 提取标签名称
	tags := make([]string, 0, len(a.Tags))
	for _, tag := range a.Tags {
		tags = append(tags, tag.Name)
	}

	// 处理发布时间
	var publishedAt time.Time
	if a.PublishedAt != nil {
		publishedAt = *a.PublishedAt
	}

	// 创建ES文档
	return &ESArticle{
		ID:           fmt.Sprintf("article_%d", a.ID),
		ArticleID:    a.ID,
		Title:        a.Title,
		Content:      content,
		Summary:      a.Summary,
		CategoryID:   a.CategoryID,
		CategoryName: a.Category.Name,
		AuthorID:     a.AuthorID,
		AuthorName:   a.Author.Username,
		Status:       a.Status,
		AccessType:   a.AccessType,
		Tags:         tags,
		IsTop:        a.IsTop,
		IsOriginal:   a.IsOriginal,
		PublishedAt:  publishedAt,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}
