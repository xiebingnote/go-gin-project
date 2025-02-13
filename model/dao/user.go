package dao

import (
	"go-gin-project/library/resource"
	"go-gin-project/model/types"

	"gorm.io/gorm"
)

type UserClient struct {
	mysql *gorm.DB
}

func NewUserClient() *UserClient {
	return &UserClient{
		mysql: resource.MySQLClient,
	}
}

type User interface {
	CreateTb() error
	GetUserNameByID(id string) (string, error)
}

func (u *UserClient) CreateTb() error {
	return nil
	//return u.mysql.AutoMigrate(&types.TbUser{})
}

func (u *UserClient) GetUserNameByID(id string) (string, error) {
	err := u.mysql.Table("tb_user").Where("id = ?", id).Select("username").Find(&types.TbUser{}).Error
	return types.TbUser{}.Username, err
}
