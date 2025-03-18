package ctypes

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type MyTime time.Time

// MarshalJSON json.Marshal 的时候会自动调用这个方法
func (t MyTime) MarshalJSON() ([]byte, error) {
	stamp := time.Time(t).Format(time.RFC3339)
	return []byte(`"` + stamp + `"`), nil
}

// UnmarshalJSON json.Unmarshal 的时候会自动调用这个方法
func (t *MyTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	// 先尝试解析 ISO-8601 格式
	tmp, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// 如果解析失败，尝试解析传统格式
		tmp, err = time.Parse(time.DateTime, s)
		if err != nil {
			return err
		}
	}
	*t = MyTime(tmp)
	return nil
}

// String 实现 Stringer 接口，方便打印
func (t MyTime) String() string {
	return time.Time(t).Format("2006-01-02 15:04:05")
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
		pt, err := time.Parse("2006-01-02 15:04:05", v)
		if err != nil {
			return err
		}
		*t = MyTime(pt)
	default:
		return fmt.Errorf("无法将 %v 转换为 MyTime", value)
	}
	return nil
}
