package redis_ser

import (
	"blog/global"
	"context"
	"strconv"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"go.uber.org/zap"
)

const (
	// 文章统计数据的键前缀
	ArticleStatsKey = "article:stats:"

	// 统计字段名
	FieldLookCount    = "look_count"
	FieldCommentCount = "comment_count"

	ViewIPExpire       = 10 * time.Minute // IP访问记录过期时间
	ViewBatchSize      = 100              // 批量更新阈值
	ViewUpdateInterval = time.Second      // 批量更新间隔

	// 布隆过滤器相关常量
	BloomFilterKey     = "article:bloom" // 布隆过滤器的键
	BloomFilterSize    = 100000          // 预期元素数量
	BloomFalsePositive = 0.01            // 期望的误判率

	ArticleStatsExpire = 7 * 24 * time.Hour  // 文章统计数据过期时间
	BloomFilterExpire  = 30 * 24 * time.Hour // 布隆过滤器过期时间
)

// 获取文章统计数据的Redis键
func GetArticleStatsKey(articleID string) string {
	return BuildKey(ArticlePrefix, articleID)
}

// 获取文章浏览数
func GetArticleLookCount(articleID string) (int64, error) {
	count, err := global.Redis.HGet(
		context.Background(),
		GetArticleStatsKey(articleID),
		FieldLookCount,
	).Int64()
	if err != nil {
		return 0, err // 如果字段不存在，返回错误
	}
	return count, nil
}

// 检查IP是否最近访问过文章
func checkIPViewArticle(articleID, ip string) (bool, error) {
	key := BuildKey(ArticlePrefix, "view", "ip", articleID, ip)
	// SetNX 命令：设置一个锁，如果这个 IP 最近没访问过这篇文章，设置成功，返回 true
	// 如果这个 IP 最近访问过这篇文章，设置失败，返回 false
	return global.Redis.SetNX(
		context.Background(),
		key,
		1,
		ViewIPExpire,
	).Result()
}

// 优化后的增加文章浏览数函数
func IncrArticleLookCount(articleID, ip string) error {
	ctx := context.Background()

	// 检查IP是否最近访问过
	isNewView, err := checkIPViewArticle(articleID, ip)
	if err != nil {
		global.Log.Error("检查IP访问记录失败",
			zap.String("articleID", articleID),
			zap.String("ip", ip),
			zap.String("error", err.Error()),
		)
		return err

	}

	// 如果最近访问过，不增加计数
	if !isNewView {
		global.Log.Info("IP最近已访问过文章",
			zap.String("articleID", articleID),
			zap.String("ip", ip))
		return nil
	}

	// 使用Pipeline批量执行命令
	pipe := global.Redis.Pipeline()

	// 增加浏览计数
	pipe.HIncrBy(
		ctx,
		GetArticleStatsKey(articleID),
		FieldLookCount,
		1,
	)

	// 执行Pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		global.Log.Error("增加文章浏览数失败",
			zap.String("articleID", articleID),
			zap.String("ip", ip),
			zap.String("error", err.Error()),
		)

		// 重试一次
		_, err = pipe.Exec(ctx)
		if err != nil {
			return err
		}
	}

	global.Log.Info("文章浏览数增加成功",
		zap.String("articleID", articleID),
		zap.String("ip", ip))

	return nil
}

// 获取文章评论数
func GetArticleCommentCount(articleID string) (int64, error) {
	return global.Redis.HGet(
		context.Background(),
		GetArticleStatsKey(articleID),
		FieldCommentCount,
	).Int64()
}

// 增加文章评论数
func IncrArticleCommentCount(articleID string) error {
	return global.Redis.HIncrBy(
		context.Background(),
		GetArticleStatsKey(articleID),
		FieldCommentCount,
		1,
	).Err()
}

// 减少文章评论数
func DecrArticleCommentCount(articleID string) error {
	global.Log.Info("DecrArticleCommentCount", zap.String("articleID", articleID))
	return global.Redis.HIncrBy(
		context.Background(),
		GetArticleStatsKey(articleID),
		FieldCommentCount,
		-1,
	).Err()
}

// 设置文章评论数
func SetArticleCommentCount(articleID string, count int64) error {
	return global.Redis.HSet(
		context.Background(),
		GetArticleStatsKey(articleID),
		FieldCommentCount,
		count,
	).Err()
}

// 获取布隆过滤器
func getBloomFilter() (*bloom.BloomFilter, error) {
	ctx := context.Background()

	// 尝试从Redis获取布隆过滤器数据
	data, err := global.Redis.Get(ctx, BloomFilterKey).Bytes()
	if err != nil && err.Error() != "redis: nil" {
		return nil, err
	}

	// 创建布隆过滤器
	filter := bloom.NewWithEstimates(BloomFilterSize, BloomFalsePositive)

	// 如果Redis中存在数据，则恢复布隆过滤器状态
	if len(data) > 0 {
		filter.UnmarshalBinary(data)
	}

	return filter, nil
}

// 保存布隆过滤器到Redis
func saveBloomFilter(filter *bloom.BloomFilter) error {
	ctx := context.Background()

	// 将布隆过滤器序列化
	data, err := filter.MarshalBinary()
	if err != nil {
		return err
	}

	// 保存到Redis
	return global.Redis.Set(ctx, BloomFilterKey, data, 0).Err()
}

// 将文章ID添加到布隆过滤器
func AddToBloomFilter(articleID string) error {
	filter, err := getBloomFilter()
	if err != nil {
		global.Log.Error("获取布隆过滤器失败", zap.Error(err))
		return err
	}

	// 添加文章ID到布隆过滤器
	filter.Add([]byte(articleID))

	// 保存更新后的布隆过滤器
	if err := saveBloomFilter(filter); err != nil {
		global.Log.Error("保存布隆过滤器失败", zap.Error(err))
		return err
	}

	return nil
}

// 检查文章ID是否可能存在
func CheckBloomFilter(articleID string) (bool, error) {
	filter, err := getBloomFilter()
	if err != nil {
		global.Log.Error("获取布隆过滤器失败", zap.Error(err))
		return false, err
	}

	return filter.Test([]byte(articleID)), nil
}

// 获取文章所有统计数据
func GetArticleStats(articleID string) (map[string]int64, error) {
	// 先检查布隆过滤器
	exists, err := CheckBloomFilter(articleID)
	if err != nil {
		global.Log.Error("检查布隆过滤器失败",
			zap.String("articleID", articleID),
			zap.Error(err))
		// 如果布隆过滤器检查失败，继续执行原有逻辑
	} else if !exists {
		global.Log.Info("文章ID在布隆过滤器中不存在",
			zap.String("articleID", articleID))
		return nil, nil // 文章一定不存在
	}

	result, err := global.Redis.HGetAll(
		context.Background(),
		GetArticleStatsKey(articleID),
	).Result()

	if err != nil {
		return nil, err
	}

	// 将字符串值转换为int64
	stats := make(map[string]int64)
	for field, value := range result {
		count, _ := strconv.ParseInt(value, 10, 64)
		stats[field] = count
	}

	return stats, nil
}

// 删除文章统计数据
func DeleteArticleStats(articleID string) error {
	return global.Redis.Del(
		context.Background(),
		GetArticleStatsKey(articleID),
	).Err()
}
