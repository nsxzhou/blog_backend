package corn_ser

import (
	"time"

	"github.com/robfig/cron/v3"
)

// 每隔5分钟执行一次
//"0 */5 * * * *"     //每隔5分钟（00:05:00, 00:10:00, ...)

// 每小时执行一次
//"0 0 * * * *"      // 每小时的开始（01:00:00, 02:00:00, ...)

// 每天执行一次
//"0 0 0 * * *"      // 每天凌晨（00:00:00）
//"0 0 12 * * *"     // 每天中午12点（12:00:00）

// 每周执行一次
//"0 0 0 * * 0"      // 每周日凌晨
//"0 0 0 * * MON"    // 每周一凌晨

// 每月执行一次
//"0 0 0 1 * *"      // 每月1号凌晨

func CornInit() {
	timezone, _ := time.LoadLocation("Asia/Shanghai")
	Cron := cron.New(cron.WithSeconds(), cron.WithLocation(timezone))
	Cron.AddFunc("0 */1 * * * *", SyncArticleData)
	Cron.Start()
}
