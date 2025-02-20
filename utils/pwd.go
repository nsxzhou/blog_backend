package utils

import (
	"blog/global"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hash密码
func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		global.Log.Error("bcrypt.GenerateFromPassword() failed", zap.String("error", err.Error()))
		return "", err
	}
	return string(hash), nil

}

// CheckPassword 验证密码
func CheckPassword(hashPwd string, pwd string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashPwd), []byte(pwd))
	if err != nil {
		global.Log.Error("bcrypt.CompareHashAndPassword() failed", zap.String("error", err.Error()))
		return false
	}
	return true
}
