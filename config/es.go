package config

type Es struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}
