package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// GenCode 生成4位随机验证码
func GenCode() string {
	rand.NewSource(time.Now().UnixNano())
	return fmt.Sprintf("%04d", rand.Intn(10000))
}
