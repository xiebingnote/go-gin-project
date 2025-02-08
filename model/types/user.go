package types

import "gorm.io/gorm"

type TbUser struct {
	gorm.Model
	Username string `gorm:"unique"`
	Password string
	Role     string // 添加角色字段
}
