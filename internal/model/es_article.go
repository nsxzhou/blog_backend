package model

import "time"

// ESArticle Elasticsearch文章文档模型
type ESArticle struct {
	ID           string    `json:"id"`            // ES文档ID，格式为"article_{mysql_id}"
	ArticleID    uint      `json:"article_id"`    // MySQL中的文章ID
	Title        string    `json:"title"`         // 文章标题（用于搜索）
	Content      string    `json:"content"`       // 文章内容（核心搜索内容）
	Summary      string    `json:"summary"`       // 文章摘要（用于搜索）
	AuthorID     uint      `json:"author_id"`     // 作者ID（用于过滤）
	AuthorName   string    `json:"author_name"`   // 作者名称（用于展示）
	CategoryID   uint      `json:"category_id"`   // 分类ID（用于过滤）
	CategoryName string    `json:"category_name"` // 分类名称（用于展示）
	Tags         []string  `json:"tags"`          // 标签列表（用于过滤和搜索）
	Status       string    `json:"status"`        // 状态(draft/published)（用于过滤）
	AccessType   string    `json:"access_type"`   // 访问类型
	IsTop        int       `json:"is_top"`
	IsOriginal   int       `json:"is_original"`
	PublishedAt  time.Time `json:"published_at,omitempty"` // 发布时间（用于排序）
	CreatedAt    time.Time `json:"created_at"`             // 创建时间
	UpdatedAt    time.Time `json:"updated_at"`             // 更新时间
}

// ESIndexName 返回ES索引名称
func (ESArticle) ESIndexName() string {
	return "articles"
}

// ESMapping 返回ES索引映射
func (ESArticle) ESMapping() string {
	return `{
		"settings": {
			"number_of_shards": 1,
			"number_of_replicas": 1,
			"analysis": {
				"analyzer": {
					"text_analyzer": {
						"type": "custom",
						"tokenizer": "standard",
						"char_filter": ["html_strip"],
						"filter": ["lowercase", "asciifolding"]
					}
				}
			}
		},
		"mappings": {
			"properties": {
				"id": { "type": "keyword" },
				"article_id": { "type": "long" },
				"title": { 
					"type": "text", 
					"analyzer": "text_analyzer",
					"fields": {
						"keyword": { "type": "keyword" }
					}
				},
				"content": { 
					"type": "text", 
					"analyzer": "text_analyzer"
				},
				"summary": { "type": "text", "analyzer": "text_analyzer" },
				"author_id": { "type": "long" },
				"author_name": { "type": "keyword" },
				"category_id": { "type": "long" },
				"category_name": { "type": "keyword" },
				"tags": { "type": "keyword" },
				"status": { "type": "keyword" },
				"access_type": { "type": "keyword" },
				"is_top": { "type": "integer" },
				"is_original": { "type": "integer" },
				"published_at": { "type": "date" },
				"created_at": { "type": "date" },
				"updated_at": { "type": "date" }
			}
		}
	}`
}
