package model

// LoginLog 用户登录日志
type LoginLog struct {
	Base
	UserID     uint   `gorm:"type:int(11);not null;index" json:"user_id"`
	IP         string `gorm:"type:varchar(50);not null" json:"ip"`
	UserAgent  string `gorm:"type:varchar(255)" json:"user_agent"`
	Status     int    `gorm:"type:tinyint(1);not null;index" json:"status"` // 0失败 1成功
	LoginType  string `gorm:"type:varchar(20);not null;default:'password'" json:"login_type"`
	DeviceInfo string `gorm:"type:varchar(255)" json:"device_info"`
	Location   string `gorm:"type:varchar(100)" json:"location"`

	// 关联
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (LoginLog) TableName() string {
	return "login_logs"
}

// OperationLog 操作日志
type OperationLog struct {
	Base
	UserID        uint   `gorm:"type:int(11);not null;index" json:"user_id"`
	Module        string `gorm:"type:varchar(50);not null;index" json:"module"`
	Operation     string `gorm:"type:varchar(50);not null" json:"operation"`
	Description   string `gorm:"type:varchar(255)" json:"description"`
	IP            string `gorm:"type:varchar(50);not null" json:"ip"`
	RequestMethod string `gorm:"type:varchar(10)" json:"request_method"`
	RequestURL    string `gorm:"type:varchar(255)" json:"request_url"`
	RequestParams string `gorm:"type:text" json:"request_params"`
	StatusCode    int    `gorm:"type:int(11)" json:"status_code"`
	ExecutionTime int    `gorm:"type:int(11)" json:"execution_time"` // 单位: 毫秒

	// 关联
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (OperationLog) TableName() string {
	return "operation_logs"
}
