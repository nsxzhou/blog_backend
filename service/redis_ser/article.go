package redis_ser

import (
	"blog/global"
	"context"
	"strconv"
	"time"

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
)

// 获取文章统计数据的Redis键
func GetArticleStatsKey(articleID string) string {
	return Prefix + ArticleStatsKey + articleID
}

// 设置文章浏览数（直接设置新值，不是增加）
func SetArticleLookCount(articleID string, count int64) error {
	return global.Redis.HSet(
		context.Background(),
		GetArticleStatsKey(articleID),
		FieldLookCount,
		count,
	).Err()
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
	key := Prefix + "article:view:ip:" + articleID + ":" + ip
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

// 获取文章所有统计数据
func GetArticleStats(articleID string) (map[string]int64, error) {
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

// 批量设置文章统计数据（在原有基础上增加）
func SetArticleStats(articleID string, stats map[string]int64) error {
	// 先获取原有的统计数据
	currentStats, err := GetArticleStats(articleID)
	if err != nil && err.Error() != "redis: nil" {
		return err
	}

	// 在原有基础上增加新的值
	data := make(map[string]interface{})
	for field, value := range stats {
		data[field] = currentStats[field] + value
	}

	return global.Redis.HMSet(
		context.Background(),
		GetArticleStatsKey(articleID),
		data,
	).Err()
}

// 删除文章统计数据
func DeleteArticleStats(articleID string) error {
	return global.Redis.Del(
		context.Background(),
		GetArticleStatsKey(articleID),
	).Err()
}
