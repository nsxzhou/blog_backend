package model

// Favorite 收藏模型
type Favorite struct {
	Base
	UserID    uint `gorm:"type:int(11);not null;index" json:"user_id"`
	ArticleID uint `gorm:"type:int(11);not null;index" json:"article_id"`

	// 关联
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Article Article `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
}

// TableName 指定表名
func (Favorite) TableName() string {
	return "favorites"
}

// ArticleLike 文章点赞模型
type ArticleLike struct {
	Base
	UserID    uint `gorm:"type:int(11);not null;index" json:"user_id"`
	ArticleID uint `gorm:"type:int(11);not null;index" json:"article_id"`

	// 关联
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Article Article `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
}

// TableName 指定表名
func (ArticleLike) TableName() string {
	return "article_likes"
}

// UserFollow 用户关注模型
type UserFollow struct {
	Base
	FollowerID uint `gorm:"type:int(11);not null;index" json:"follower_id"`
	FollowedID uint `gorm:"type:int(11);not null;index" json:"followed_id"`

	// 关联
	Follower User `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Followed User `gorm:"foreignKey:FollowedID" json:"followed,omitempty"`
}

// TableName 指定表名
func (UserFollow) TableName() string {
	return "user_follows"
}
