package config

type QQ struct {
	AppID       string `mapstructure:"app_id"`
	AppKey      string `mapstructure:"app_key"`
	RedirectURL string `mapstructure:"redirect_url"`
}
