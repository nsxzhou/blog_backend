package models

import (
	"blog/global"
)

type LogModel struct {
	MODEL    `json:","`
	Level    string `json:"level" gorm:"type:varchar(10);index"`
	Caller   string `json:"caller" gorm:"type:varchar(100)"`
	Message  string `json:"message" gorm:"type:text"`
	ErrorMsg string `json:"error_msg" gorm:"type:text"`
}

func (l *LogModel) Delete() error {
	return global.DB.Delete(l).Error
}
