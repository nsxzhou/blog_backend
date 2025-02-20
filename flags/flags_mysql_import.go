package flags

import (
	"blog/global"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// MysqlImport 从SQL文件导入数据到MySQL数据库
// 参数 c *cli.Context 包含命令行参数，需要包含 --path 参数指定SQL文件路径
// 返回 error 如果导入过程中出现错误则返回对应错误信息
func MysqlImport(c *cli.Context) error {
	// 获取命令行参数中指定的SQL文件路径
	path := c.String("path")

	// 读取SQL文件内容
	byteData, err := os.ReadFile(path)
	if err != nil {
		global.Log.Error("读取SQL文件失败", zap.String("error", err.Error()), zap.String("path", path))
		return err
	}

	// 统一换行符处理，解决跨平台兼容性问题
	// 先将 Windows 的 \r\n 转换为 \n
	content := strings.ReplaceAll(string(byteData), "\r\n", "\n")
	// 再将可能存在的 Mac 的 \r 转换为 \n
	content = strings.ReplaceAll(content, "\r", "\n")

	// 按分号和换行符分割SQL语句
	sqlList := strings.Split(content, ";\n")

	// 开启数据库事务
	tx := global.DB.Begin()

	// 设置 defer 处理，确保发生 panic 时事务能够回滚
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 用于记录执行失败的SQL语句
	failedSQL := make([]string, 0)

	// 遍历执行所有SQL语句
	for i, sql := range sqlList {
		// 去除SQL语句前后的空白字符
		sql = strings.TrimSpace(sql)
		if sql == "" {
			continue
		}

		// 在事务中执行SQL语句
		if err := tx.Exec(sql).Error; err != nil {
			// 记录详细的错误信息
			global.Log.Error("SQL执行失败",
				zap.String("error", err.Error()),
				zap.Int("index", i),
				zap.String("sql", sql))

			// 回滚事务
			tx.Rollback()
			return fmt.Errorf("导入失败：第%d条SQL执行出错: %v", i+1, err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		global.Log.Error("事务提交失败", zap.String("error", err.Error()))
		return err
	}

	// 记录导入完成的统计信息
	global.Log.Info("数据库导入成功",
		zap.Int("total", len(sqlList)),
		zap.Int("failed", len(failedSQL)))

	return nil
}
