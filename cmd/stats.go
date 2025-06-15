package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/spf13/cobra"
)

// statsCmd 统计命令
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "统计信息命令",
	Long:  `显示系统统计信息，包括用户、文章、评论等数据`,
}

// systemStatsCmd 系统统计命令
var systemStatsCmd = &cobra.Command{
	Use:   "system",
	Short: "系统统计信息",
	Long:  `显示系统整体统计信息`,
	Run: func(cmd *cobra.Command, args []string) {
		showSystemStats()
	},
}

// userStatsCmd 用户统计命令
var userStatsCmd = &cobra.Command{
	Use:   "users",
	Short: "用户统计信息",
	Long:  `显示用户相关统计信息`,
	Run: func(cmd *cobra.Command, args []string) {
		showUserStats()
	},
}

// articleStatsCmd 文章统计命令
var articleStatsCmd = &cobra.Command{
	Use:   "articles",
	Short: "文章统计信息",
	Long:  `显示文章相关统计信息`,
	Run: func(cmd *cobra.Command, args []string) {
		showArticleStats()
	},
}

// dbStatusCmd 数据库状态命令
var dbStatusCmd = &cobra.Command{
	Use:   "db-status",
	Short: "数据库状态",
	Long:  `显示数据库连接状态`,
	Run: func(cmd *cobra.Command, args []string) {
		showDatabaseStatus()
	},
}

func init() {
	// 添加统计相关子命令
	statsCmd.AddCommand(systemStatsCmd)
	statsCmd.AddCommand(userStatsCmd)
	statsCmd.AddCommand(articleStatsCmd)
	statsCmd.AddCommand(dbStatusCmd)
	
	// 将统计命令添加到根命令
	rootCmd.AddCommand(statsCmd)
}

// showSystemStats 显示系统统计信息
func showSystemStats() {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()
	
	fmt.Println("=== 系统统计信息 ===")
	
	// 用户统计
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)
	
	var activeUserCount int64
	db.Model(&model.User{}).Where("status = ?", 1).Count(&activeUserCount)
	
	var adminCount int64
	db.Model(&model.User{}).Where("role = ?", "admin").Count(&adminCount)
	
	// 文章统计
	var articleCount int64
	db.Model(&model.Article{}).Count(&articleCount)
	
	var publishedArticleCount int64
	db.Model(&model.Article{}).Where("status = ?", "published").Count(&publishedArticleCount)
	
	var draftArticleCount int64
	db.Model(&model.Article{}).Where("status = ?", "draft").Count(&draftArticleCount)
	
	// 评论统计
	var commentCount int64
	db.Model(&model.Comment{}).Count(&commentCount)
	
	// 分类统计
	var categoryCount int64
	db.Model(&model.Category{}).Count(&categoryCount)
	
	// 标签统计
	var tagCount int64
	db.Model(&model.Tag{}).Count(&tagCount)
	
	// 今日新增统计
	today := time.Now().Format("2006-01-02")
	var todayUsers, todayArticles, todayComments int64
	
	db.Model(&model.User{}).Where("DATE(created_at) = ?", today).Count(&todayUsers)
	db.Model(&model.Article{}).Where("DATE(created_at) = ?", today).Count(&todayArticles)
	db.Model(&model.Comment{}).Where("DATE(created_at) = ?", today).Count(&todayComments)
	
	fmt.Printf("用户总数: %d (活跃: %d, 管理员: %d)\n", userCount, activeUserCount, adminCount)
	fmt.Printf("文章总数: %d (已发布: %d, 草稿: %d)\n", articleCount, publishedArticleCount, draftArticleCount)
	fmt.Printf("评论总数: %d\n", commentCount)
	fmt.Printf("分类总数: %d\n", categoryCount)
	fmt.Printf("标签总数: %d\n", tagCount)
	fmt.Printf("今日新增: 用户 %d, 文章 %d, 评论 %d\n", todayUsers, todayArticles, todayComments)
}

// showUserStats 显示用户统计信息
func showUserStats() {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()
	userService := service.NewUserService()
	
	fmt.Println("=== 用户统计信息 ===")
	
	// 获取用户统计
	stats, err := userService.GetUserStats()
	if err != nil {
		fmt.Printf("获取用户统计失败: %v\n", err)
		return
	}
	
	fmt.Printf("总用户数: %d\n", stats.TotalUsers)
	fmt.Printf("活跃用户: %d\n", stats.ActiveUsers)
	fmt.Printf("管理员用户: %d\n", stats.AdminUsers)	
	fmt.Printf("禁用用户: %d\n", stats.DisabledUsers)
	fmt.Printf("新注册用户: %d\n", stats.NewUsers)
	
	// 最近注册的用户
	var recentUsers []model.User
	db.Select("id, username, email, created_at").
		Order("created_at DESC").
		Limit(5).
		Find(&recentUsers)
	
	fmt.Println("\n最近注册用户:")
	for _, user := range recentUsers {
		fmt.Printf("- %s - %s\n", user.Username, user.CreatedAt.Format("2006-01-02 15:04"))
	}
	
	// 用户角色分布
	var roleStats []struct {
		Role  string
		Count int64
	}
	db.Model(&model.User{}).
		Select("role, COUNT(*) as count").
		Group("role").
		Find(&roleStats)
	
	fmt.Println("\n用户角色分布:")
	for _, stat := range roleStats {
		fmt.Printf("- %s: %d\n", stat.Role, stat.Count)
	}
}

// showArticleStats 显示文章统计信息
func showArticleStats() {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()
	
	fmt.Println("=== 文章统计信息 ===")
	
	// 文章状态统计
	var statusStats []struct {
		Status string
		Count  int64
	}
	db.Model(&model.Article{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusStats)
	
	fmt.Println("文章状态分布:")
	for _, stat := range statusStats {
		fmt.Printf("- %s: %d\n", stat.Status, stat.Count)
	}
	
	// 分类文章统计
	var categoryStats []struct {
		CategoryName string
		Count        int64
	}
	db.Table("articles").
		Select("categories.name as category_name, COUNT(*) as count").
		Joins("LEFT JOIN categories ON articles.category_id = categories.id").
		Where("articles.status = ?", "published").
		Group("categories.name").
		Order("count DESC").
		Limit(10).
		Find(&categoryStats)
	
	fmt.Println("\n热门分类 (前10):")
	for _, stat := range categoryStats {
		fmt.Printf("- %s: %d 篇\n", stat.CategoryName, stat.Count)
	}
	
	// 最受欢迎的文章
	var popularArticles []struct {
		ID         uint
		Title      string
		ViewCount  int
		LikeCount  int
		AuthorName string
	}
	db.Table("articles").
		Select("articles.id, articles.title, articles.view_count, articles.like_count, users.nickname as author_name").
		Joins("LEFT JOIN users ON articles.author_id = users.id").
		Where("articles.status = ?", "published").
		Order("articles.view_count DESC").
		Limit(5).
		Find(&popularArticles)
	
	fmt.Println("\n最受欢迎文章 (前5):")
	for i, article := range popularArticles {
		fmt.Printf("%d. %s (作者: %s, 浏览: %d, 点赞: %d)\n", 
			i+1, article.Title, article.AuthorName, article.ViewCount, article.LikeCount)
	}
	
	// 最活跃作者
	var activeAuthors []struct {
		AuthorName   string
		ArticleCount int64
		TotalViews   int64
	}
	db.Table("articles").
		Select("users.nickname as author_name, COUNT(*) as article_count, SUM(articles.view_count) as total_views").
		Joins("LEFT JOIN users ON articles.author_id = users.id").
		Where("articles.status = ?", "published").
		Group("users.nickname").
		Order("article_count DESC").
		Limit(5).
		Find(&activeAuthors)
	
	fmt.Println("\n最活跃作者 (前5):")
	for i, author := range activeAuthors {
		fmt.Printf("%d. %s (文章: %d, 总浏览: %d)\n", 
			i+1, author.AuthorName, author.ArticleCount, author.TotalViews)
	}
}

// showDatabaseStatus 显示数据库状态
func showDatabaseStatus() {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 数据库状态 ===")
	
	// MySQL状态
	db := database.GetDB()
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Printf("MySQL: 连接失败 - %v\n", err)
	} else {
		if err := sqlDB.Ping(); err != nil {
			fmt.Printf("MySQL: 连接失败 - %v\n", err)
		} else {
			stats := sqlDB.Stats()
			fmt.Printf("MySQL: 连接正常\n")
			fmt.Printf("  - 最大连接数: %d\n", stats.MaxOpenConnections)
			fmt.Printf("  - 当前连接数: %d\n", stats.OpenConnections)
			fmt.Printf("  - 空闲连接数: %d\n", stats.Idle)
			fmt.Printf("  - 使用中连接数: %d\n", stats.InUse)
		}
	}
	
	// Elasticsearch状态
	es := database.GetES()
	if es == nil {
		fmt.Println("Elasticsearch: 连接失败")
	} else {
		ctx := context.Background()
		res, err := es.Info(es.Info.WithContext(ctx))
		if err != nil {
			fmt.Printf("Elasticsearch: 连接失败 - %v\n", err)
		} else {
			fmt.Printf("Elasticsearch: 连接正常 - %s\n", res.Status())
		}
	}
	
	// Redis状态
	redis := database.GetRedis()
	if redis == nil {
		fmt.Println("Redis: 连接失败")
	} else {
		ctx := context.Background()
		pong, err := redis.Ping(ctx).Result()
		if err != nil {
			fmt.Printf("Redis: 连接失败 - %v\n", err)
		} else {
			fmt.Printf("Redis: 连接正常 - %s\n", pong)
			
			// Redis信息
			info, err := redis.Info(ctx, "memory").Result()
			if err == nil {
				fmt.Printf("  - 内存信息: %s\n", info[:100]+"...")
			}
		}
	}
} 