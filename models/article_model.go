package models

import (
	"blog/global"
	"blog/models/ctypes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/refresh"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Article 文章模型
type Article struct {
	ID            string        `json:"id"`
	CreatedAt     ctypes.MyTime `json:"created_at"`     // 创建时间
	UpdatedAt     ctypes.MyTime `json:"updated_at"`     // 更新时间
	Title         string        `json:"title"`          // 文章标题
	Abstract      string        `json:"abstract"`       // 文章简介
	Content       string        `json:"content"`        // 文章内容
	LookCount     uint          `json:"look_count"`     // 浏览量
	CommentCount  uint          `json:"comment_count"`  // 评论量
	DiggCount     uint          `json:"digg_count"`     // 点赞量
	CollectsCount uint          `json:"collects_count"` // 收藏量
	UserID        uint          `json:"user_id"`        // 用户id
	UserName      string        `json:"user_name"`      // 用户昵称
	Category      []string      `json:"category"`       // 文章分类
	CoverID       uint          `json:"cover_id"`       // 封面id
	CoverURL      string        `json:"cover_url"`      // 封面
	Version       int64         `json:"version"`        // 版本号
}

const (
	articleIndex = "article_index"
	batchSize    = 1000
	timeout      = time.Second * 5
)

// ArticleService 文章服务
type ArticleService struct {
	ctx          context.Context
	articleIndex string
	batchSize    int
	timeout      time.Duration
}

type DateRange struct {
	Start string `json:"start" form:"start"`
	End   string `json:"end" form:"end"`
}

// SearchParams 搜索参数
type SearchParams struct {
	PageInfo
	SortField string    `json:"sort_field" form:"sort_field"`
	SortOrder string    `json:"sort_order" form:"sort_order"`
	Category  []string  `json:"category" form:"category"`
	DateRange DateRange `json:"date_range" form:"date_range"`
}

// SearchResult 搜索结果
type SearchResults struct {
	Articles []Article
	Total    int64
}

// NewArticleService 创建文章服务实例
func NewArticleService() *ArticleService {
	return &ArticleService{
		ctx:          context.Background(),
		articleIndex: articleIndex,
		batchSize:    batchSize,
		timeout:      timeout,
	}
}

// IndexCreate 创建索引
func (s *ArticleService) IndexCreate() error {
	if s.ctx == nil {
		s.ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	exist, err := s.IndexExist()
	if err != nil {
		return fmt.Errorf("检查索引是否存在失败: %w", err)
	}

	if exist {
		if err := s.IndexDelete(); err != nil {
			return fmt.Errorf("删除已存在的索引失败: %w", err)
		}
	}

	// 索引映射
	properties := map[string]types.Property{
		"title":          types.NewTextProperty(),
		"abstract":       types.NewTextProperty(),
		"content":        types.NewTextProperty(),
		"category":       types.NewKeywordProperty(),
		"created_at":     types.NewDateProperty(),
		"updated_at":     types.NewDateProperty(),
		"look_count":     types.NewIntegerNumberProperty(),
		"comment_count":  types.NewIntegerNumberProperty(),
		"digg_count":     types.NewIntegerNumberProperty(),
		"collects_count": types.NewIntegerNumberProperty(),
		"user_id":        types.NewIntegerNumberProperty(),
		"user_name":      types.NewKeywordProperty(),
		"cover_id":       types.NewIntegerNumberProperty(),
		"cover_url":      types.NewKeywordProperty(),
		"version":        types.NewLongNumberProperty(),
	}

	_, err = global.Es.Indices.Create(articleIndex).
		Mappings(&types.TypeMapping{
			// 设置索引的映射规则
			Properties: properties,
		}).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}
	global.Log.Info("创建索引成功", zap.String("method", "IndexCreate"), zap.String("path", "models/article_model.go"))
	return nil
}

// IndexExist 检查索引是否存在
func (s *ArticleService) IndexExist() (bool, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	resp, err := global.Es.Indices.Exists(s.articleIndex).Do(ctx)
	if err != nil {
		return false, fmt.Errorf("检查索引是否存在失败: %w", err)
	}
	return resp, nil
}

// IndexDelete 删除索引
func (s *ArticleService) IndexDelete() error {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)

	defer cancel()

	_, err := global.Es.Indices.Delete(s.articleIndex).Do(ctx)

	if err != nil {
		return fmt.Errorf("删除索引失败: %w", err)
	}
	return nil
}

// ArticleCreate 创建文章
func (s *ArticleService) ArticleCreate(article *Article) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	exists, err := s.ArticleExist(article.ID)
	if err != nil {
		return fmt.Errorf("检查文章是否存在失败: %w", err)
	}
	if exists {
		return fmt.Errorf("文章已存在")
	}
	article.CreatedAt = ctypes.MyTime(time.Now())
	article.UpdatedAt = ctypes.MyTime(time.Now())
	article.Version = 1

	_, err = global.Es.Index(s.articleIndex).
		Id(article.ID).
		Document(article).
		Refresh(refresh.True).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("创建文章失败: %w", err)
	}

	return nil
}

// ArticleGet 获取文章
func (s *ArticleService) ArticleGet(id string) (*Article, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	var result Article
	resp, err := global.Es.Get(s.articleIndex, id).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取文章失败: %w", err)
	}

	if err := json.Unmarshal(resp.Source_, &result); err != nil {
		return nil, fmt.Errorf("解析文章数据失败: %w", err)
	}

	return &result, nil
}

// ArticleUpdate 更新文章
func (s *ArticleService) ArticleUpdate(article *Article) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	article.Version++
	article.UpdatedAt = ctypes.MyTime(time.Now())

	_, err := global.Es.Update(s.articleIndex, article.ID).
		Doc(article).
		Refresh(refresh.True).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("更新文章失败: %w", err)
	}

	return nil
}

// ArticleDelete 批量删除文章
func (s *ArticleService) ArticleDelete(ids []string) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	//errgroup errgroup 提供了一种同步机制，用于管理一组 goroutine 并收集它们的错误
	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batch := ids[i:end]

		// 构建批量删除请求
		bulkRequest := global.Es.Bulk().Index(s.articleIndex)

		for _, id := range batch {
			bulkRequest.DeleteOp(types.DeleteOperation{Id_: &id})
		}

		// 执行批量删除请求
		g.Go(func() error {
			resp, err := bulkRequest.Refresh(refresh.True).Do(ctx)
			if err != nil {
				return fmt.Errorf("批量删除文章失败: %w", err)
			}

			if resp.Errors {
				return fmt.Errorf("批量删除文章时发生错误")
			}

			return nil
		})
	}
	// g.Wait() 等待所有并发任务完成，返回第一个发生的错误（如果有）
	return g.Wait()
}

// ArticleSearch 搜索文章
func (s *ArticleService) ArticleSearch(params SearchParams) (*SearchResults, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	// 1. 构建布尔查询
	boolQuery := types.NewBoolQuery()

	// 2. 关键词搜索增强
	if params.PageInfo.Key != "" {
		// 多字段匹配，使用best_fields策略
		multiMatchQuery := types.NewMultiMatchQuery()
		multiMatchQuery.Query = params.PageInfo.Key
		multiMatchQuery.Fields = []string{
			"title^3",       // 标题权重最高
			"abstract^2",    // 摘要次之
			"content",       // 内容权重最低
			"user_name^1.5", // 作者名也可搜索
		}
		boolQuery.Must = append(boolQuery.Must, types.Query{MultiMatch: multiMatchQuery})
	}

	// 3. 分类过滤
	if len(params.Category) > 0 {
		// 只匹配其中一个分类
		// boolQuery.Filter = append(boolQuery.Filter, types.Query{
		// 	Terms: &types.TermsQuery{
		// 		TermsQuery: map[string]types.TermsQueryField{
		// 			"category": params.Category,
		// 		},
		// 	},
		// })
		//
		// 匹配多个分类
		for _, category := range params.Category {
			boolQuery.Filter = append(boolQuery.Filter, types.Query{
				Term: map[string]types.TermQuery{
					"category": {Value: category},
				},
			})
		}
	}

	// 4. 日期范围过滤
	if params.DateRange.Start != "" && params.DateRange.End != "" {
		rangeQuery := types.NewDateRangeQuery()
		rangeQuery.Gte = &params.DateRange.Start
		rangeQuery.Lte = &params.DateRange.End
		boolQuery.Filter = append(boolQuery.Filter, types.Query{
			Range: map[string]types.RangeQuery{"created_at": *rangeQuery},
		})
	}

	// 6. 分页处理
	page := params.PageInfo.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageInfo.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	from := (page - 1) * pageSize

	// 7. 排序处理
	sortField := params.SortField
	sortOrder := params.SortOrder

	if sortField == "" {
		sortField = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// 10. 构建搜索请求
	searchRequest := global.Es.Search().
		Index(s.articleIndex).
		Query(&types.Query{Bool: boolQuery}).
		Sort(types.SortOptions{
			SortOptions: map[string]types.FieldSort{
				sortField: {Order: &sortorder.SortOrder{Name: sortOrder}},
			},
		}).
		From(from).
		Size(pageSize)

	// 12. 执行搜索
	resp, err := searchRequest.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("搜索文章失败: %w", err)
	}

	// 13. 处理搜索结果
	articles := make([]Article, 0, int(resp.Hits.Total.Value))
	for _, hit := range resp.Hits.Hits {
		var article Article
		if err := json.Unmarshal(hit.Source_, &article); err != nil {
			global.Log.Error("解析文章数据失败",
				zap.String("error", err.Error()),
				zap.String("document_id", *hit.Id_),
			)
			continue
		}
		articles = append(articles, article)
	}

	return &SearchResults{
		Articles: articles,
		Total:    resp.Hits.Total.Value,
	}, nil
}

// ArticleExist 检查文章是否存在
func (s *ArticleService) ArticleExist(id string) (bool, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	exists, err := global.Es.Exists(s.articleIndex, id).Do(ctx)
	if err != nil {
		return false, fmt.Errorf("检查文章是否存在失败: %w", err)
	}

	return exists, nil
}

// ArticleStats 文章统计数据
type ArticleStats struct {
	TotalArticles int64 `json:"total_articles"` // 文章总数
	TotalComments int64 `json:"total_comments"` // 评论数
	TotalViews    int64 `json:"total_views"`    // 总浏览量
	TotalDiggs    int64 `json:"total_diggs"`    // 总点赞数
	TotalCollects int64 `json:"total_collects"` // 总收藏数
}

// GetArticleStats 获取文章统计数据
func (s *ArticleService) GetArticleStats() (*ArticleStats, error) {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	// 定义需要统计的字段
	statsFields := map[string]string{
		"total_comments": "comment_count",
		"total_views":    "look_count",
		"total_diggs":    "digg_count",
		"total_collects": "collects_count",
	}

	// 构建聚合查询
	aggs := make(map[string]types.Aggregations, len(statsFields))
	for aggName, field := range statsFields {
		aggs[aggName] = types.Aggregations{
			Sum: &types.SumAggregation{
				// 聚合字段
				Field: &[]string{field}[0],
			},
		}
	}

	// 执行查询
	resp, err := global.Es.Search().
		Index(s.articleIndex).
		Size(0). // 不需要返回文档，只需要聚合结果
		Aggregations(aggs).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取统计数据失败: %w", err)
	}

	// 处理结果
	stats := &ArticleStats{
		TotalArticles: resp.Hits.Total.Value,
	}

	// 统一处理聚合结果
	for aggName, field := range map[string]*int64{
		"total_comments": &stats.TotalComments,
		"total_views":    &stats.TotalViews,
		"total_diggs":    &stats.TotalDiggs,
		"total_collects": &stats.TotalCollects,
	} {
		if agg, found := resp.Aggregations[aggName]; found {
			var sumAgg types.SumAggregate
			aggBytes, _ := json.Marshal(agg)
			if err := json.Unmarshal(aggBytes, &sumAgg); err != nil {
				global.Log.Error("解析聚合结果失败",
					zap.String("aggregation", aggName),
					zap.Error(err),
				)
				continue
			}
			*field = int64(sumAgg.Value)
		}
	}

	return stats, nil
}

// IncrementCount 更新指定计数字段
func (s *ArticleService) IncrementCount(id string, field string, increment int) error {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	script := fmt.Sprintf("ctx._source.%s += params.increment", field)
	_, err := global.Es.Update(s.articleIndex, id).
		Script(&types.InlineScript{
			Source: script,
			Params: map[string]json.RawMessage{
				"increment": json.RawMessage(fmt.Sprintf("%d", increment)),
			},
		}).
		Refresh(refresh.True).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("更新%s失败: %w", field, err)
	}

	return nil
}
