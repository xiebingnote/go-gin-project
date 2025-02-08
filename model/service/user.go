package service

import (
	"project/model/dao"
)

var mysqlDB = dao.NewUserClient()

func CreateTb() error {
	return mysqlDB.CreateTb()
}

func GetUserNameByID() {

	return
}
