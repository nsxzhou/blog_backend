package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// ArticleCacheService 文章缓存服务
type ArticleCacheService struct {
	cache       Cache
	bloomFilter BloomFilter
}

// NewArticleCacheService 创建文章缓存服务
func NewArticleCacheService(cache Cache, bloomFilter BloomFilter) *ArticleCacheService {
	return &ArticleCacheService{
		cache:       cache,
		bloomFilter: bloomFilter,
	}
}

// GetArticleDetail 获取文章详情缓存
func (a *ArticleCacheService) GetArticleDetail(ctx context.Context, articleID uint, dest interface{}) error {
	// 首先检查布隆过滤器，防止缓存穿透
	exists, err := a.bloomFilter.Test(ctx, strconv.FormatUint(uint64(articleID), 10))
	if err != nil {
		return fmt.Errorf("check bloom filter failed: %w", err)
	}
	
	// 如果布隆过滤器说不存在，直接返回不存在错误
	if !exists {
		return redis.Nil // 模拟缓存不存在
	}
	
	// 从缓存获取文章详情
	key := fmt.Sprintf(ArticleDetailKey, articleID)
	return a.cache.GetJSON(ctx, key, dest)
}

// SetArticleDetail 设置文章详情缓存
func (a *ArticleCacheService) SetArticleDetail(ctx context.Context, articleID uint, article interface{}) error {
	// 先添加到布隆过滤器
	if err := a.bloomFilter.Add(ctx, strconv.FormatUint(uint64(articleID), 10)); err != nil {
		return fmt.Errorf("add to bloom filter failed: %w", err)
	}
	
	// 设置缓存
	key := fmt.Sprintf(ArticleDetailKey, articleID)
	return a.cache.SetJSON(ctx, key, article, ArticleDetailExpiration)
}

// DeleteArticleDetail 删除文章详情缓存
func (a *ArticleCacheService) DeleteArticleDetail(ctx context.Context, articleID uint) error {
	key := fmt.Sprintf(ArticleDetailKey, articleID)
	return a.cache.Delete(ctx, key)
}

// GetArticleList 获取文章列表缓存
func (a *ArticleCacheService) GetArticleList(ctx context.Context, page, pageSize int, dest interface{}) error {
	key := fmt.Sprintf(ArticleListKey, page, pageSize)
	return a.cache.GetJSON(ctx, key, dest)
}

// SetArticleList 设置文章列表缓存
func (a *ArticleCacheService) SetArticleList(ctx context.Context, page, pageSize int, articles interface{}) error {
	key := fmt.Sprintf(ArticleListKey, page, pageSize)
	return a.cache.SetJSON(ctx, key, articles, ArticleListExpiration)
}

// DeleteArticleListCache 删除文章列表相关的所有缓存
func (a *ArticleCacheService) DeleteArticleListCache(ctx context.Context) error {
	// 这里可以实现删除所有文章列表缓存的逻辑
	// 由于Redis没有直接的模式删除，我们可以使用一个更简单的方法
	// 或者维护一个缓存键的集合
	
	// 删除常用的分页缓存
	keys := make([]string, 0)
	for page := 1; page <= 10; page++ { // 删除前10页的缓存
		for pageSize := 10; pageSize <= 50; pageSize += 10 {
			key := fmt.Sprintf(ArticleListKey, page, pageSize)
			keys = append(keys, key)
		}
	}
	
	if len(keys) > 0 {
		return a.cache.Delete(ctx, keys...)
	}
	
	return nil
}

// GetArticleCategoryList 获取分类文章列表缓存
func (a *ArticleCacheService) GetArticleCategoryList(ctx context.Context, categoryID uint, page, pageSize int, dest interface{}) error {
	key := fmt.Sprintf(ArticleCategoryListKey, categoryID, page, pageSize)
	return a.cache.GetJSON(ctx, key, dest)
}

// SetArticleCategoryList 设置分类文章列表缓存
func (a *ArticleCacheService) SetArticleCategoryList(ctx context.Context, categoryID uint, page, pageSize int, articles interface{}) error {
	key := fmt.Sprintf(ArticleCategoryListKey, categoryID, page, pageSize)
	return a.cache.SetJSON(ctx, key, articles, ArticleListExpiration)
}

// DeleteArticleCategoryListCache 删除分类文章列表缓存
func (a *ArticleCacheService) DeleteArticleCategoryListCache(ctx context.Context, categoryID uint) error {
	keys := make([]string, 0)
	for page := 1; page <= 10; page++ {
		for pageSize := 10; pageSize <= 50; pageSize += 10 {
			key := fmt.Sprintf(ArticleCategoryListKey, categoryID, page, pageSize)
			keys = append(keys, key)
		}
	}
	
	if len(keys) > 0 {
		return a.cache.Delete(ctx, keys...)
	}
	
	return nil
}

// GetArticleTagList 获取标签文章列表缓存
func (a *ArticleCacheService) GetArticleTagList(ctx context.Context, tagID uint, page, pageSize int, dest interface{}) error {
	key := fmt.Sprintf(ArticleTagListKey, tagID, page, pageSize)
	return a.cache.GetJSON(ctx, key, dest)
}

// SetArticleTagList 设置标签文章列表缓存
func (a *ArticleCacheService) SetArticleTagList(ctx context.Context, tagID uint, page, pageSize int, articles interface{}) error {
	key := fmt.Sprintf(ArticleTagListKey, tagID, page, pageSize)
	return a.cache.SetJSON(ctx, key, articles, ArticleListExpiration)
}

// DeleteArticleTagListCache 删除标签文章列表缓存
func (a *ArticleCacheService) DeleteArticleTagListCache(ctx context.Context, tagID uint) error {
	keys := make([]string, 0)
	for page := 1; page <= 10; page++ {
		for pageSize := 10; pageSize <= 50; pageSize += 10 {
			key := fmt.Sprintf(ArticleTagListKey, tagID, page, pageSize)
			keys = append(keys, key)
		}
	}
	
	if len(keys) > 0 {
		return a.cache.Delete(ctx, keys...)
	}
	
	return nil
}

// GetHotArticles 获取热门文章缓存
func (a *ArticleCacheService) GetHotArticles(ctx context.Context, page, pageSize int, dest interface{}) error {
	key := fmt.Sprintf(ArticleHotListKey, page, pageSize)
	return a.cache.GetJSON(ctx, key, dest)
}

// SetHotArticles 设置热门文章缓存
func (a *ArticleCacheService) SetHotArticles(ctx context.Context, page, pageSize int, articles interface{}) error {
	key := fmt.Sprintf(ArticleHotListKey, page, pageSize)
	return a.cache.SetJSON(ctx, key, articles, ArticleHotExpiration)
}

// GetSearchResults 获取搜索结果缓存
func (a *ArticleCacheService) GetSearchResults(ctx context.Context, query string, page, pageSize int, dest interface{}) error {
	key := fmt.Sprintf(ArticleSearchKey, query, page, pageSize)
	return a.cache.GetJSON(ctx, key, dest)
}

// SetSearchResults 设置搜索结果缓存
func (a *ArticleCacheService) SetSearchResults(ctx context.Context, query string, page, pageSize int, results interface{}) error {
	key := fmt.Sprintf(ArticleSearchKey, query, page, pageSize)
	return a.cache.SetJSON(ctx, key, results, ArticleSearchExpiration)
}

// GetFullTextSearchResults 获取全文搜索结果缓存
func (a *ArticleCacheService) GetFullTextSearchResults(ctx context.Context, query string, page, pageSize int, dest interface{}) error {
	key := fmt.Sprintf(ArticleFullTextSearchKey, query, page, pageSize)
	return a.cache.GetJSON(ctx, key, dest)
}

// SetFullTextSearchResults 设置全文搜索结果缓存
func (a *ArticleCacheService) SetFullTextSearchResults(ctx context.Context, query string, page, pageSize int, results interface{}) error {
	key := fmt.Sprintf(ArticleFullTextSearchKey, query, page, pageSize)
	return a.cache.SetJSON(ctx, key, results, ArticleSearchExpiration)
}

// GetArticleStats 获取文章统计缓存
func (a *ArticleCacheService) GetArticleStats(ctx context.Context) (string, error) {
	return a.cache.Get(ctx, ArticleStatsKey)
}

// SetArticleStats 设置文章统计缓存
func (a *ArticleCacheService) SetArticleStats(ctx context.Context, stats interface{}) error {
	return a.cache.SetJSON(ctx, ArticleStatsKey, stats, StatsExpiration)
}

// GetCategoryStats 获取分类统计缓存
func (a *ArticleCacheService) GetCategoryStats(ctx context.Context, categoryID uint) (string, error) {
	key := fmt.Sprintf(CategoryStatsKey, categoryID)
	return a.cache.Get(ctx, key)
}

// SetCategoryStats 设置分类统计缓存
func (a *ArticleCacheService) SetCategoryStats(ctx context.Context, categoryID uint, count interface{}) error {
	key := fmt.Sprintf(CategoryStatsKey, categoryID)
	return a.cache.SetJSON(ctx, key, count, StatsExpiration)
}

// GetTagStats 获取标签统计缓存
func (a *ArticleCacheService) GetTagStats(ctx context.Context, tagID uint) (string, error) {
	key := fmt.Sprintf(TagStatsKey, tagID)
	return a.cache.Get(ctx, key)
}

// SetTagStats 设置标签统计缓存
func (a *ArticleCacheService) SetTagStats(ctx context.Context, tagID uint, count interface{}) error {
	key := fmt.Sprintf(TagStatsKey, tagID)
	return a.cache.SetJSON(ctx, key, count, StatsExpiration)
}

// BatchAddArticlesToBloomFilter 批量添加文章ID到布隆过滤器
func (a *ArticleCacheService) BatchAddArticlesToBloomFilter(ctx context.Context, articleIDs []uint) error {
	elements := make([]string, len(articleIDs))
	for i, id := range articleIDs {
		elements[i] = strconv.FormatUint(uint64(id), 10)
	}
	
	return a.bloomFilter.BatchAdd(ctx, elements)
}

// InvalidateArticleCaches 清除文章相关的所有缓存（文章更新时调用）
func (a *ArticleCacheService) InvalidateArticleCaches(ctx context.Context, articleID uint, categoryID uint, tagIDs []uint) error {
	// 删除文章详情缓存
	if err := a.DeleteArticleDetail(ctx, articleID); err != nil {
		return fmt.Errorf("delete article detail cache failed: %w", err)
	}
	
	// 删除文章列表缓存
	if err := a.DeleteArticleListCache(ctx); err != nil {
		return fmt.Errorf("delete article list cache failed: %w", err)
	}
	
	// 删除分类文章列表缓存
	if categoryID > 0 {
		if err := a.DeleteArticleCategoryListCache(ctx, categoryID); err != nil {
			return fmt.Errorf("delete category article list cache failed: %w", err)
		}
	}
	
	// 删除标签文章列表缓存
	for _, tagID := range tagIDs {
		if err := a.DeleteArticleTagListCache(ctx, tagID); err != nil {
			return fmt.Errorf("delete tag article list cache failed: %w", err)
		}
	}
	
	return nil
} 