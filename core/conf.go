package core

import (
	"blog/global"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// 配置文件路径
const filePath = "settings.yaml"

// InitConf 初始化配置
func InitConf() {
	// 设置配置文件路径
	viper.SetConfigFile(filePath)
	// 读取配置文件
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("viper.ReadInConfig() Failed, err:%v\n", err)
	}
	// 将配置文件解码到global.Config
	if err := viper.Unmarshal(&global.Config); err != nil {
		log.Fatalf("viper.Unmarshal() Failed, err:%v\n", err)
	}
	// 监听配置文件变化
	viper.WatchConfig()
	// 当配置文件发生变化时，重新读取配置文件
	viper.OnConfigChange(func(in fsnotify.Event) {
		log.Println("Config file changed...")
		if err := viper.Unmarshal(global.Config); err != nil {
			log.Fatalf("viper.Unmarshal() Failed, err:%v\n", err)
		}
	})
}
