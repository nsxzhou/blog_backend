package user

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"blog/global"
	"blog/models"
	"blog/models/ctypes"
	"blog/models/res"
	"blog/service/redis_ser"
	"blog/utils"

	"github.com/avast/retry-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type QQLoginURL struct {
	URL string `json:"url"`
}

// GetQQLoginURL 获取QQ登录URL
func (u *User) GetQQLoginURL(c *gin.Context) {
	// 构建QQ登录URL
	loginURL := fmt.Sprintf("https://graph.qq.com/oauth2.0/authorize?"+
		"response_type=code&"+
		"client_id=%s&"+
		"redirect_uri=%s",
		global.Config.QQ.AppID,
		url.QueryEscape(global.Config.QQ.RedirectURL),
	)

	res.Success(c, QQLoginURL{
		URL: loginURL,
	})

}

// QQLoginCallback QQ登录回调处理
func (u *User) QQLoginCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		res.Error(c, res.InvalidParameter, "无效的授权码")
		return
	}

	// 1. 获取 access_token
	tokenURL := fmt.Sprintf("https://graph.qq.com/oauth2.0/token?grant_type=authorization_code&client_id=%s&client_secret=%s&code=%s&redirect_uri=%s",
		global.Config.QQ.AppID,
		global.Config.QQ.AppKey,
		code,
		global.Config.QQ.RedirectURL,
	)

	// 使用带超时的HTTP客户端
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	accessToken, err := getQQAccessToken(ctx, tokenURL)
	if err != nil {
		global.Log.Error("获取QQ access_token失败", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "QQ登录失败")
		return
	}

	// 2. 获取 OpenID
	openIDURL := fmt.Sprintf("https://graph.qq.com/oauth2.0/me?access_token=%s", accessToken)
	openID, err := getQQOpenID(openIDURL)
	if err != nil {
		global.Log.Error("获取QQ OpenID失败", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "QQ登录失败")
		return
	}

	// 3. 获取用户信息
	userInfoURL := fmt.Sprintf("https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&openid=%s",
		accessToken,
		global.Config.QQ.AppID,
		openID,
	)
	qqUserInfo, err := getQQUserInfo(userInfoURL)
	if err != nil {
		global.Log.Error("获取QQ用户信息失败", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "QQ登录失败")
		return
	}

	// 4. 查找或创建用户
	var user models.UserModel
	err = user.FindByQQOpenID(openID)
	if err != nil {
		// 创建新用户
		account, err := utils.GenerateID()
		if err != nil {
			global.Log.Error("生成用户ID失败", zap.String("error", err.Error()))
			res.Error(c, res.ServerError, "用户创建失败")
			return
		}

		user = models.UserModel{
			Account:  strconv.FormatInt(account, 10),
			Nickname: qqUserInfo.Nickname,
			QQOpenID: openID,
			Role:     ctypes.RoleUser,
			Password: "nsxzhou.fun",
		}

		if err := user.Create(c.ClientIP()); err != nil {
			global.Log.Error("创建用户失败", zap.String("error", err.Error()))
			res.Error(c, res.ServerError, "用户创建失败")
			return
		}

	}

	// 5. 生成登录token
	// ... 复用现有的token生成逻辑 ...
	userPayload := utils.PayLoad{
		Account: user.Account,
		Role:    user.Role,
		UserID:  user.ID,
	}

	accessToken, err = utils.GenerateAccessToken(userPayload)
	if err != nil {
		global.Log.Error("生成access token失败", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "登录失败")
		return
	}

	// 6. 更新用户token并返回
	if err := user.UpdateToken(accessToken); err != nil {
		global.Log.Error("更新用户token失败", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "登录失败")
		return
	}

	c.Request.Header.Set("Authorization", "Bearer "+accessToken)

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		global.Log.Error("utils.GenerateRefreshToken() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "生成refresh token失败")
		return
	}

	expiration := time.Duration(global.Config.Jwt.Expires) * 24 * time.Hour
	key := redis_ser.RefreshToken + strconv.Itoa(int(user.ID))
	err = global.Redis.Set(context.Background(), redis_ser.GetRedisKey(key), refreshToken, expiration).Err()
	if err != nil {
		global.Log.Error("global.Redis.Set() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "设置 refresh token 到 redis 失败")
		return
	}

	global.Log.Info("用户QQ登录成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, accessToken)
}

// QQUserInfo QQ用户信息结构体
type QQUserInfo struct {
	Nickname     string `json:"nickname"`
	Figureurl    string `json:"figureurl"`
	Figureurl1   string `json:"figureurl_1"`
	Figureurl2   string `json:"figureurl_2"`
	Gender       string `json:"gender"`
	VipInfo      string `json:"vip"`   // 改为string类型
	Level        string `json:"level"` // 改为string类型
	IsYellowVip  string `json:"is_yellow_vip"`
	IsYellowYear string `json:"is_yellow_year_vip"`
}

// 获取QQ access token
func getQQAccessToken(ctx context.Context, url string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	var accessToken string
	err := retry.Do(
		func() error {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return fmt.Errorf("创建请求失败: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("发送请求失败: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("读取响应失败: %v", err)
			}

			// QQ返回的格式类似：access_token=YOUR_TOKEN&expires_in=7776000&refresh_token=YOUR_REFRESH_TOKEN
			params := utils.ParseQueryString(string(body))

			accessToken = params["access_token"]
			if accessToken == "" {
				return fmt.Errorf("access_token 为空")
			}

			return nil
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
		retry.OnRetry(func(n uint, err error) {
			global.Log.Warn("重试获取access token",
				zap.Uint("attempt", n+1),
				zap.String("error", err.Error()))
		}),
	)

	return accessToken, err
}

// 辅助函数：获取QQ OpenID
func getQQOpenID(url string) (string, error) {
	resp, err := utils.HttpGet(url)
	if err != nil {
		return "", fmt.Errorf("请求OpenID失败: %v", err)
	}

	// QQ返回的格式类似：callback( {"client_id":"YOUR_APPID","openid":"YOUR_OPENID"} );
	openID := utils.ExtractOpenID(resp)
	if openID == "" {
		return "", fmt.Errorf("解析OpenID失败: %s", resp)
	}

	return openID, nil
}

// 辅助函数：获取QQ用户信息
func getQQUserInfo(url string) (*QQUserInfo, error) {
	resp, err := utils.HttpGet(url)
	if err != nil {
		return nil, fmt.Errorf("请求用户信息失败: %v", err)
	}

	// 添加原始响应日志
	global.Log.Debug("QQ用户信息原始响应", zap.String("response", resp))

	var userInfo QQUserInfo
	if err := json.Unmarshal([]byte(resp), &userInfo); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %v, 原始数据: %s", err, resp)
	}

	if userInfo.Nickname == "" {
		return nil, fmt.Errorf("获取用户信息失败,昵称为空: %s", resp)
	}

	return &userInfo, nil
}
