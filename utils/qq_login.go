package utils

import (
	"io"
	"net/http"
	"regexp"
	"strings"
)

// HttpGet 发送GET请求
func HttpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ParseQueryString 解析URL查询字符串
func ParseQueryString(query string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// ExtractOpenID 从QQ回调响应中提取OpenID
func ExtractOpenID(resp string) string {
	re := regexp.MustCompile(`"openid":"(.*?)"`)
	matches := re.FindStringSubmatch(resp)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
