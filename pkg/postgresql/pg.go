package postgresql

import "github.com/xiebingnote/go-gin-project/library/resource"

type Test struct {
	Id   int
	Name string
	Age  int
}

// GetTestAll retrieves all records from the "test" table.
//
// It queries the database using the Postgresql client and returns a slice of Test structs.
// If an error occurs during the query, it returns the error. If no records are found, it returns nil.
func GetTestAll() ([]Test, error) {
	var res []Test

	// Query the "test" table and select all columns.
	err := resource.PostgresqlClient.Table("test").Select("*").Find(&res).Error
	if err != nil {
		return nil, err // Return error if the query fails.
	}

	// Return the results if there are any records found.
	if len(res) > 0 {
		return res, nil
	}

	// Return nil if no records are found.
	return nil, nil
}
