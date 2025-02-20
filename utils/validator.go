package utils

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validate(i interface{}) error {
	return validate.Struct(i)
}

func FormatValidationError(errs validator.ValidationErrors) string {
	// 定义错误信息映射
	msgMap := map[string]string{
		"required": "不能为空",
		"min":      "长度不能小于%v",
		"max":      "长度不能大于%v",
		"email":    "必须是有效的邮箱地址",
		"url":      "必须是有效的网址",
		"oneof":    "必须是[%v]中的一个",
		"gt":       "必须大于%v",
		"gte":      "必须大于等于%v",
		"lt":       "必须小于%v",
		"lte":      "必须小于等于%v",
	}

	// 字段名称映射（可以将英文字段名转换为中文）
	fieldMap := map[string]string{
		"Title":    "标题",
		"Content":  "内容",
		"Email":    "邮箱",
		"Password": "密码",
		"Age":      "年龄",
		"Phone":    "手机号",
	}

	// 获取第一个错误（通常我们只返回第一个错误）
	firstErr := errs[0]

	// 获取字段中文名（如果没有映射，就用原字段名）
	fieldName := fieldMap[firstErr.Field()]
	if fieldName == "" {
		fieldName = firstErr.Field()
	}

	// 获取错误消息模板
	msgTemplate := msgMap[firstErr.Tag()]
	if msgTemplate == "" {
		msgTemplate = "验证失败"
	}

	// 如果错误标签有参数，则格式化消息
	if firstErr.Param() != "" {
		return fieldName + fmt.Sprintf(msgTemplate, firstErr.Param())
	}

	return fieldName + msgTemplate
}
