package dao

import (
	"go-gin-project/library/resource"
	"go-gin-project/model/types"

	"gorm.io/gorm"
)

type UserClient struct {
	db *gorm.DB
}

func NewUserClient() *UserClient {
	return &UserClient{
		db: resource.MySQLClient,
	}
}

type User interface {
	CreateTb() error
	GetUserNameByID(id string) (string, error)
}

func (u *UserClient) CreateTb() error {

	return u.db.Table("tb_user").AutoMigrate(&types.TbUser{})
}

func (u *UserClient) GetUserNameByID(id string) (string, error) {
	var info types.TbUser
	err := u.db.Table("tb_user").Where("id = ?", id).Select("username").Find(&info).Error
	return info.Username, err
}
