package types

// TbUser 用户表
type TbUser struct {
	ID       uint   `gorm:"column:id;type:int(10) unsigned;primary_key;AUTO_INCREMENT;" json:"id"` // 主键ID
	Username string `gorm:"column:username;type:varchar(20);NOT NULL" json:"username"`             // 用户名
	Password string `gorm:"column:password;type:varchar(128);NOT NULL" json:"password"`            // 密码
	Role     string `gorm:"column:role;type:varchar(20)" json:"role"`                              // 添加角色字段
	TbModel
}
