package config

type TencentCos struct {
	SecretID  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
	BucketURL string `mapstructure:"bucket_url"`
}

