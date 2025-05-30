package cache

import (
	"context"
	"time"
)

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存
	Get(ctx context.Context, key string) (string, error)
	
	// Set 设置缓存
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	
	// SetNX 设置缓存（不存在时才设置）
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	
	// Delete 删除缓存
	Delete(ctx context.Context, keys ...string) error
	
	// Exists 检查key是否存在
	Exists(ctx context.Context, keys ...string) (int64, error)
	
	// Expire 设置过期时间
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	
	// GetJSON 获取JSON格式的缓存并反序列化
	GetJSON(ctx context.Context, key string, dest interface{}) error
	
	// SetJSON 序列化为JSON并设置缓存
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	
	// Close 关闭连接
	Close() error
}

// CacheKey 缓存键名常量
const (
	// 文章相关缓存键
	ArticleDetailKey        = "article:detail:%d"                    // 文章详情
	ArticleListKey          = "article:list:page:%d:size:%d"        // 文章列表
	ArticleCategoryListKey  = "article:category:%d:page:%d:size:%d" // 分类文章列表
	ArticleTagListKey       = "article:tag:%d:page:%d:size:%d"      // 标签文章列表
	ArticleHotListKey       = "article:hot:page:%d:size:%d"         // 热门文章列表
	ArticleSearchKey        = "article:search:%s:page:%d:size:%d"   // 搜索结果
	
	// 统计相关缓存键
	ArticleStatsKey         = "stats:article:total"                 // 文章总数统计
	CategoryStatsKey        = "stats:category:%d:count"             // 分类文章数统计
	TagStatsKey             = "stats:tag:%d:count"                  // 标签文章数统计
	
	// 分类标签相关缓存键
	CategoryListKey         = "category:list:visible:%d"            // 分类列表
	TagListKey              = "tag:list:hot"                        // 热门标签列表
	
	// 用户相关缓存键
	UserInfoKey             = "user:info:%d"                        // 用户信息
	UserPermissionsKey      = "user:permissions:%d"                // 用户权限
	
	// 评论相关缓存键
	CommentListKey          = "comment:list:article:%d:page:%d"     // 评论列表
	
	// 布隆过滤器相关键
	BloomFilterArticleKey   = "bloom:article:exists"               // 文章存在性布隆过滤器
	BloomFilterUserKey      = "bloom:user:exists"                  // 用户存在性布隆过滤器
)

// CacheExpiration 缓存过期时间常量
const (
	ArticleDetailExpiration   = 30 * time.Minute  // 文章详情缓存30分钟
	ArticleListExpiration     = 15 * time.Minute  // 文章列表缓存15分钟
	ArticleHotExpiration      = 1 * time.Hour     // 热门文章缓存1小时
	ArticleSearchExpiration   = 5 * time.Minute   // 搜索结果缓存5分钟
	
	CategoryListExpiration    = 1 * time.Hour     // 分类列表缓存1小时
	TagListExpiration         = 2 * time.Hour     // 标签列表缓存2小时
	
	UserInfoExpiration        = 30 * time.Minute  // 用户信息缓存30分钟
	UserPermissionsExpiration = 1 * time.Hour     // 用户权限缓存1小时
	
	CommentListExpiration     = 10 * time.Minute  // 评论列表缓存10分钟
	
	StatsExpiration           = 30 * time.Minute  // 统计数据缓存30分钟
	
	BloomFilterExpiration     = 24 * time.Hour    // 布隆过滤器缓存24小时
) 