package cmd

import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	// 这些变量在编译时通过 -ldflags 设置
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
	Platform  = runtime.GOOS + "/" + runtime.GOARCH
)

// versionCmd 版本信息命令
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  `显示应用程序的版本信息`,
	Run: func(cmd *cobra.Command, args []string) {
		showVersion()
	},
}

func init() {
	// 将版本命令添加到根命令
	rootCmd.AddCommand(versionCmd)
}

// showVersion 显示版本信息
func showVersion() {
	fmt.Printf("🚀 博客API服务\n")
	fmt.Printf("版本: %s\n", Version)
	fmt.Printf("Git提交: %s\n", GitCommit)
	fmt.Printf("构建时间: %s\n", BuildTime)
	fmt.Printf("Go版本: %s\n", GoVersion)
	fmt.Printf("平台: %s\n", Platform)
	fmt.Printf("当前时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
} 