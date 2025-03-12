package user

import (
	"github.com/xiebingnote/go-gin-project/model/dao/user"
)

var clientUser = user.NewUserClient()

// CreateTb creates the table in the database.
//
// It calls the AutoMigrate method on the global MySQLClient connection to
// create the table in the database.
//
// Returns an error if the table creation fails.
func CreateTb() error {
	return clientUser.CreateTb()
}

// GetUserNameByID retrieves the username associated with the given ID.
//
// It calls the GetUserNameByID method on the userClient to retrieve the username
// associated with the given ID. If the retrieval fails or the user is not found,
// it returns an empty string and the error.
func GetUserNameByID(id string) (string, error) {
	return clientUser.GetUserNameByID(id)
}
