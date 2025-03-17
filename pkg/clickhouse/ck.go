package clickhouse

import "github.com/xiebingnote/go-gin-project/library/resource"

type Test struct {
	Id   int
	Name *string
	Age  *int32
}

// GetTestAll retrieves all records from the "test" table.
//
// It queries the ClickHouse database using the ClickHouse client and returns a slice of Test structs.
// If an error occurs during the query, it returns the error. If no records are found, it returns nil.
func GetTestAll() ([]Test, error) {
	var resAll []Test

	rows, err := resource.ClickHouseClient.Query("select * from test")
	if err != nil {
		// Return error if the query fails.
		return nil, err
	}

	// Iterate over all rows returned by the query.
	for rows.Next() {
		var test Test
		err = rows.Scan(&test.Id, &test.Name, &test.Age)
		if err != nil {
			// Return error if the query fails.
			return nil, err
		}

		// Append the retrieved record to the result slice.
		resAll = append(resAll, test)
	}

	return resAll, nil
}
