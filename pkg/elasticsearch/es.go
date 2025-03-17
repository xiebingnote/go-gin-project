package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/olivere/elastic/v7"
)

var ctx = context.Background()

type Test struct {
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

// Insert adds a new document to the Elasticsearch index.
//
// This function creates a bulk insert request with the specified index name,
// which includes the current date.
//
// It then constructs a document with the given fields and adds it to the request.
// Finally, it executes the request and checks for errors.
//
// Returns an error if the insertion operation fails, otherwise nil.
func Insert() error {
	// Construct the index name using the current date
	indexName := "products" + time.Now().Format("20060102")

	// Create a bulk request for the specified index name
	req := resource.ElasticSearchClient.Bulk().Index(indexName)

	// Create a document with the specified fields
	doc := elastic.NewBulkCreateRequest().Doc(map[string]interface{}{
		"title":      "Cross body Bag with Tassel",
		"created_at": time.Now(),
	})

	// Add the document to the bulk request
	req.Add(doc)

	// Execute the insert request and retrieve the response
	_, err := req.Do(ctx)
	if err != nil {
		return err
	}

	// Return nil on success
	return nil
}

// Search searches the specified index name for documents.
//
// It creates a search request with the specified index name and sorts
// the results in descending order by the "created_at" field. It then
// executes the search request and retrieves the response.
//
// If the search operation fails, it returns an error, otherwise nil.
//
// The method returns the first document found in the index, if any.
func Search() error {
	// The name of the index to search
	indexName := "products*"
	// Create a search request with the specified index name and sorts
	// the results in descending order by the "created_at" field
	req := resource.ElasticSearchClient.Search().
		Index(indexName).
		Sort("created_at", false).
		Size(1)

	// Execute the search request and retrieve the response
	res, err := req.Do(ctx)
	if err != nil {
		return err
	}

	// Print the results
	if res.Hits.TotalHits.Value == 0 {
		return nil
	}

	// Iterate over the results
	for _, hit := range res.Hits.Hits {
		// Unmarshal the response into a Test struct
		var test Test
		if err = json.Unmarshal(hit.Source, &test); err != nil {
			return err
		}
		// Print the document found
		// Example: Found document 1 in index products20230209:
		// {Cross body Bag with Tassel 2023-02-09 16:51:03.357 +0800 CST}
		fmt.Printf("Found document %s in index %s:\n", hit.Id, indexName)
		fmt.Println(test)
	}

	return nil
}
