package config

type Captcha struct {
	Open      bool `mapstructure:"open"`
	KeyLong   int  `mapstructure:"key_long"`
	ImgWidth  int  `mapstructure:"img_width"`
	ImgHeight int  `mapstructure:"img_height"`
}
