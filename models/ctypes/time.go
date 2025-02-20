package ctypes

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type MyTime time.Time

// MarshalJSON 自定义序列化方法
func (t MyTime) MarshalJSON() ([]byte, error) {
	// 将 Time 转换为 time.Time 后格式化
	stamp := time.Time(t).Format("2006-01-02")
	// 注意要加上引号，因为 JSON 中的字符串必须用引号括起来
	return []byte(`"` + stamp + `"`), nil
}

// UnmarshalJSON 自定义反序列化方法
func (t *MyTime) UnmarshalJSON(data []byte) error {
	// 去掉引号
	str := string(data)[1 : len(data)-1]
	// 解析时间字符串
	pt, err := time.Parse("2006-01-02", str)
	if err != nil {
		return err
	}
	*t = MyTime(pt)
	return nil
}

// String 实现 Stringer 接口，方便打印
func (t MyTime) String() string {
	return time.Time(t).Format("2006-01-02")
}

// Value 实现 driver.Valuer 接口
func (t MyTime) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return nil, nil
	}
	return time.Time(t), nil
}

// Scan 实现 sql.Scanner 接口
func (t *MyTime) Scan(value interface{}) error {
	if value == nil {
		*t = MyTime(time.Time{})
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*t = MyTime(v)
	case string:
		pt, err := time.Parse("2006-01-02", v)
		if err != nil {
			return err
		}
		*t = MyTime(pt)
	default:
		return fmt.Errorf("无法将 %v 转换为 MyTime", value)
	}
	return nil
}
