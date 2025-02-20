package corn_ser

import (
	"blog/global"
	"blog/models"
	"blog/service/redis_ser"
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SyncArticleData 同步文章数据从Redis到ES
func SyncArticleData() {
	// 创建文章服务实例
	articleService := models.NewArticleService()

	// 获取所有文章统计数据的键
	ctx := context.Background()
	pattern := redis_ser.Prefix + redis_ser.ArticleStatsKey + "*"
	iter := global.Redis.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		// 从键中提取文章ID
		articleID := strings.TrimPrefix(key, redis_ser.Prefix+redis_ser.ArticleStatsKey)

		// 获取Redis中的统计数据
		stats, err := redis_ser.GetArticleStats(articleID)
		if err != nil {
			global.Log.Error("获取Redis文章统计数据失败",
				zap.String("article_id", articleID),
				zap.String("error", err.Error()),
			)
			continue

		}

		// 获取ES中的文章数据
		article, err := articleService.ArticleGet(articleID)
		if err != nil {
			global.Log.Error("获取ES文章数据失败",
				zap.String("article_id", articleID),
				zap.String("error", err.Error()),
			)
			continue
		}

		// 更新文章统计数据
		needsUpdate := false
		if lookCount, exists := stats[redis_ser.FieldLookCount]; exists && uint(lookCount) != article.LookCount {
			article.LookCount = uint(lookCount)
			needsUpdate = true
		}
		if commentCount, exists := stats[redis_ser.FieldCommentCount]; exists && uint(commentCount) != article.CommentCount {
			article.CommentCount = uint(commentCount)
			needsUpdate = true
		}

		// 如果有数据需要更新，则更新ES中的文章数据
		if needsUpdate {
			if err := articleService.ArticleUpdate(article); err != nil {
				global.Log.Error("更新ES文章数据失败",
					zap.String("article_id", articleID),
					zap.String("error", err.Error()),
				)
				continue

			}
			global.Log.Info("同步文章数据成功",
				zap.String("article_id", articleID),
				zap.Any("stats", stats))
		}

		// 避免过快请求
		time.Sleep(time.Millisecond * 100)
	}

	if err := iter.Err(); err != nil {
		global.Log.Error("遍历Redis键失败", zap.String("error", err.Error()))
	}

}
