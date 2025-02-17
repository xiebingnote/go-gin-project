package service

import (
	"go-gin-project/model/dao"
)

var user = dao.NewUserClient()

func CreateTb() error {
	return user.CreateTb()
}

func GetUserNameByID(id string) (string, error) {
	return user.GetUserNameByID(id)
}
