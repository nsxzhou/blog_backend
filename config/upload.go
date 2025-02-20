package config

type Upload struct {
	Size uint   `mapstructure:"size"`
	Path string `mapstructure:"path"`
}
