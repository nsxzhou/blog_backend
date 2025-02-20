package config

import (
	"strconv"
)

type Config struct {
	Mysql   Mysql   `mapstructure:"mysql"`
	Redis   Redis   `mapstructure:"redis"`
	Log     Log     `mapstructure:"log"`
	System  System  `mapstructure:"system"`
	Es      Es      `mapstructure:"es"`
	Jwt     Jwt     `mapstructure:"jwt"`
	Captcha Captcha `mapstructure:"captcha"`
	Upload  Upload  `mapstructure:"upload"`
	QQ      QQ      `mapstructure:"qq"`
	TencentCos TencentCos `mapstructure:"tencent_cos"`
}



func (m Mysql) Dsn() string {
	return m.User + ":" + m.Password + "@tcp(" + m.Host + ":" + strconv.Itoa(m.Port) + ")/" + m.DB + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func (m Mysql) DSNWithoutDB() string {
	return m.User + ":" + m.Password + "@tcp(" + m.Host + ":" + strconv.Itoa(m.Port) + ")/" + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func (e Es) Dsn() string {
	return "http://" + e.Host + ":" + strconv.Itoa(e.Port)
}
