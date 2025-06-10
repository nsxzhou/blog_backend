package dto

import "encoding/xml"

// RSSQuery RSS查询参数
type RSSQuery struct {
	Limit      int  `form:"limit" binding:"omitempty,min=1,max=100"`      // 限制文章数量，默认20，最大100
	CategoryID uint `form:"category_id" binding:"omitempty,min=1"`        // 按分类筛选
	TagID      uint `form:"tag_id" binding:"omitempty,min=1"`             // 按标签筛选
}

// RSSItem RSS订阅项
type RSSItem struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	PubDate     string    `xml:"pubDate"`
	GUID        string    `xml:"guid"`
	Author      string    `xml:"author"`
	Category    string    `xml:"category"`
	Enclosure   *RSSEnclosure `xml:"enclosure,omitempty"`
}

// RSSEnclosure RSS附件（如图片）
type RSSEnclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length int    `xml:"length,attr"`
}

// RSSChannel RSS频道信息
type RSSChannel struct {
	Title           string    `xml:"title"`
	Description     string    `xml:"description"`
	Link            string    `xml:"link"`
	Language        string    `xml:"language"`
	LastBuildDate   string    `xml:"lastBuildDate"`
	Generator       string    `xml:"generator"`
	WebMaster       string    `xml:"webMaster"`
	ManagingEditor  string    `xml:"managingEditor"` 
	Items           []RSSItem `xml:"item"`
}

// RSSFeed RSS订阅数据
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
} 