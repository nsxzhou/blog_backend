package models

import (
	"blog/global"
)

type VisitModel struct {
	MODEL        `json:","`
	PublicIP     string `gorm:"comment:公网IP" json:"public_ip"`
	InternalIP   string `gorm:"comment:内网IP" json:"internal_ip"`
	VisitorID    string `gorm:"comment:访客ID" json:"visitor_id"`
	UserAgent    string `gorm:"comment:用户代理" json:"user_agent"`
	Distribution string `gorm:"comment:地区" json:"distribution"`
}

func (v *VisitModel) Create() error {
	return global.DB.Create(v).Error
}

func (v *VisitModel) Delete() error {
	return global.DB.Delete(v).Error
}
