package dao

import (
	"github.com/xiebingnote/go-gin-project/library/resource"
	"github.com/xiebingnote/go-gin-project/model/types"

	"gorm.io/gorm"
)

type UserClient struct {
	db *gorm.DB
}

// NewUserClient creates a new UserClient instance.
//
// It uses the global MySQLClient connection to interact with the database.
//
// Returns a new UserClient instance.
func NewUserClient() *UserClient {
	return &UserClient{
		db: resource.MySQLClient,
	}
}

type User interface {
	// CreateTb creates the table in the database.
	//
	// It calls the AutoMigrate method on the global MySQLClient connection to
	// create the table in the database.
	//
	// Returns an error if the table creation fails.
	CreateTb() error

	// GetUserNameByID retrieves the username associated with the given ID.
	//
	// It queries the "tb_user" table to find the user with the specified ID
	// and returns the username if found.
	//
	// Parameters:
	//   - id: The ID of the user whose username is to be retrieved.
	//
	// Returns:
	//   - A string containing the username associated with the given ID.
	//   - An error if the retrieval fails or the user is not found.
	GetUserNameByID(id string) (string, error)
}

// CreateTb creates the table in the database.
//
// It calls the AutoMigrate method on the global MySQLClient connection to
// create the table in the database.
//
// Returns an error if the table creation fails.
func (u *UserClient) CreateTb() error {
	// Create the table in the database.
	// The AutoMigrate method creates the table if it doesn't already exist.
	// If the table already exists, the method returns nil.
	// If the table creation fails, the method returns an error.
	return u.db.Table("tb_user").AutoMigrate(&types.TbUser{})
}

// GetUserNameByID retrieves the username associated with the given ID.
//
// It queries the "tb_user" table to find the user with the specified ID
// and returns the username if found.
//
// Parameters:
//   - id: The ID of the user whose username is to be retrieved.
//
// Returns:
//   - A string containing the username associated with the given ID.
//   - An error if the retrieval fails or the user is not found.
func (u *UserClient) GetUserNameByID(id string) (string, error) {
	// Retrieve the user information from the database
	// The `Find` method is used to query the "tb_user" table and retrieve the
	// user information associated with the given ID.
	// The `Select` method is used to specify that only the "username" column
	// should be retrieved.
	// The `Find` method returns an error if the query fails or the user is not found.
	var info types.TbUser
	err := u.db.Table("tb_user").
		Where("id = ?", id).
		Select("username").
		Find(&info).
		Error

	// Return the username if the user is found.
	// If the user is not found, return an empty string.
	// If the query fails, return the error.
	return info.Username, err
}
