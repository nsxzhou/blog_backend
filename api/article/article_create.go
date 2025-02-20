package article

import (
	"math/rand"
	"strconv"
	"time"

	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ArticleRequest struct {
	Title    string   `json:"title" validate:"required,min=1,max=50"`
	Abstract string   `json:"abstract" validate:"required,min=1,max=100"`
	Category []string `json:"category" validate:"required,min=1,max=10,dive,min=1,max=10"`
	Content  string   `json:"content" validate:"required,min=1,max=100000"`
	CoverID  uint     `json:"cover_id" validate:"required,gt=0"`
}

func (a *Article) ArticleCreate(c *gin.Context) {
	var req ArticleRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		global.Log.Error("c.ShouldBindJSON() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "请求参数格式错误")
		return
	}

	err = utils.Validate(req)
	if err != nil {
		global.Log.Error("utils.Validate() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, utils.FormatValidationError(err.(validator.ValidationErrors)))
		return
	}
	_claims, _ := c.Get("claims")
	claims := _claims.(*utils.CustomClaims)
	userID := claims.UserID
	html, err := utils.ConvertMarkdownToHTML(req.Content)
	if err != nil {
		global.Log.Error("utils.ConvertMarkdownToHTML() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "utils.ConvertMarkdownToHTML失败")
		return
	}

	content, err := utils.ConvertHTMLToMarkdown(html)
	if err != nil {
		global.Log.Error("utils.ConvertHTMLToMarkdown() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "utils.ConvertHTMLToMarkdown失败")
		return
	}

	if req.CoverID == 0 {
		var imageIDList []uint
		global.DB.Model(models.ImageModel{}).Select("id").Scan(&imageIDList)
		if len(imageIDList) == 0 {
			global.Log.Error("global.DB.Model(models.ImageModel{}).Select() failed")
			res.Error(c, res.ServerError, "获取图片id失败")
			return
		}

		rand.New(rand.NewSource(time.Now().UnixNano()))
		req.CoverID = imageIDList[rand.Intn(len(imageIDList))]
	}

	var coverUrl string
	err = global.DB.Model(models.ImageModel{}).Where("id = ?", req.CoverID).Select("path").Scan(&coverUrl).Error
	if err != nil {
		global.Log.Error("global.DB.Model(models.ImageModel{}).Where().Select().Scan() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取图片路径失败")
		return
	}

	var user models.UserModel
	err = global.DB.Where("id = ?", userID).First(&user).Error
	if err != nil {
		global.Log.Error("global.DB.Where().First() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "查找user失败")
		return
	}

	id, err := utils.GenerateID()
	if err != nil {
		global.Log.Error("utils.GenerateID() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "生成ID失败")
		return
	}

	article := models.Article{
		ID:       strconv.FormatInt(id, 10),
		Title:    req.Title,
		Abstract: req.Abstract,
		Category: req.Category,
		Content:  content,
		CoverID:  req.CoverID,
		CoverURL: coverUrl,
		UserID:   userID,
		UserName: user.Nickname,
	}
	articleService := models.NewArticleService()
	err = articleService.ArticleCreate(&article)
	if err != nil {
		global.Log.Error("articleService.ArticleCreate() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "创建文章失败")
		return
	}
	global.Log.Info("创建文章成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)

}
