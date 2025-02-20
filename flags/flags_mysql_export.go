package flags

import (
	"blog/global"
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// MysqlExport 导出MySQL数据库备份
// 该函数通过Docker容器执行mysqldump命令来导出数据库
// 参数 c *cli.Context: CLI上下文（虽然当前未使用）
// 返回 error: 如果发生错误则返回错误信息
func MysqlExport(c *cli.Context) (err error) {
	// 从全局配置中获取MySQL配置
	mysql := global.Config.Mysql
	// 生成当前时间戳，格式为YYYYMMDD
	timer := time.Now().Format("20060102")

	// 构建SQL文件保存路径：./数据库名_日期.sql
	sqlPath := fmt.Sprintf("./%s_%s.sql", mysql.DB, timer)

	// 构建mysqldump命令
	// TODO: 建议从配置文件中读取MySQL用户名和密码，而不是硬编码
	cmder := fmt.Sprintf("docker exec mysql mysqldump -uroot -proot blog > %s", sqlPath)

	var cmd *exec.Cmd
	// 根据操作系统类型选择不同的命令执行方式
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmder)
	} else {
		// 对于Unix/Linux系统使用sh执行
		cmd = exec.Command("sh", "-c", cmder)
	}

	// 创建缓冲区用于捕获命令的标准输出和错误输出
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// 执行命令
	err = cmd.Run()
	if err != nil {
		// 如果发生错误，记录错误日志
		global.Log.Error("导出数据库失败",
			zap.String("error", err.Error()),
			zap.String("stderr", stderr.String()),
		)
		return fmt.Errorf("导出数据库失败: %v, stderr: %s", err, stderr.String())

	}

	// 导出成功，记录成功日志
	global.Log.Info("数据库导出成功",
		zap.String("文件路径", sqlPath),
		zap.String("数据库", mysql.DB),
	)
	return nil
}
