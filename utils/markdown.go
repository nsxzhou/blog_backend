package utils

import (
	"errors"
	"strings"

	"blog/global"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/russross/blackfriday"
	"go.uber.org/zap"
)

var (
	ErrEmptyContent = errors.New("内容不能为空")
)

// ConvertMarkdownToHTML 将 Markdown 内容转换为 HTML 并移除可能的恶意脚本标签
func ConvertMarkdownToHTML(content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", ErrEmptyContent
	}

	// 转换 Markdown 为 HTML
	unsafe := blackfriday.MarkdownCommon([]byte(content))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(unsafe)))
	if err != nil {
		global.Log.Error("解析 HTML 文档失败",
			zap.String("error", err.Error()),
			zap.String("content", content[:min(len(content), 100)]), // 只记录前100个字符
		)
		return "", err

	}

	// 移除所有脚本标签以提高安全性
	doc.Find("script").Remove()

	// 获取处理后的 HTML
	html, err := doc.Html()
	if err != nil {
		global.Log.Error("生成 HTML 失败",
			zap.String("error", err.Error()),
			zap.String("content", content[:min(len(content), 100)]),
		)
		return "", err
	}


	return html, nil
}

// ConvertHTMLToMarkdown 将 HTML 内容转换回 Markdown 格式
func ConvertHTMLToMarkdown(htmlContent string) (string, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return "", ErrEmptyContent
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(htmlContent)
	if err != nil {
		global.Log.Error("HTML 转 Markdown 失败",
			zap.String("error", err.Error()),
			zap.String("content", htmlContent[:min(len(htmlContent), 100)]),
		)
		return "", err
	}


	return markdown, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
