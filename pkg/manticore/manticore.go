package manticore

import (
	"context"
	"fmt"
	manticore "github.com/manticoresoftware/manticoresearch-go"
	"github.com/xiebingnote/go-gin-project/library/resource"
)

type Document struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Insert inserts a new document into the ManticoreSearch index.
//
// It creates an insert document request for the specified table name
// and document fields. The document is assigned a unique ID for identification.
//
// Returns an error if the insertion operation fails, otherwise nil.
func Insert() error {
	// Create an insert document request with the specified table name and document fields
	tbName := "products"
	indexDoc := map[string]interface{}{"title": "Crossbody Bag with Tassel"}
	indexReq := manticore.NewInsertDocumentRequest(tbName, indexDoc)

	// Set the document ID to 1
	indexReq.SetId(1)

	// Execute the insert request and retrieve the response
	_, _, err := resource.ManticoreClient.IndexAPI.Insert(context.Background()).InsertDocumentRequest(*indexReq).Execute()
	if err != nil {
		return err
	}

	// Return nil on success
	return nil
}

// Search executes a search query on the ManticoreSearch index and retrieves the search results.
//
// It creates a search request with the specified query string and table name.
// The search results are retrieved and logged to the console.
//
// Returns an error if the search operation fails, otherwise nil.
func Search() error {
	// Create a search request with the specified query string and table name
	searchRequest := manticore.NewSearchRequest("products")
	searchQuery := manticore.NewSearchQuery()
	searchQuery.QueryString = "@title Crossbody"
	searchRequest.Query = searchQuery

	// Set up highlighting for the search results
	queryHighLight := manticore.NewHighlight()
	queryHighLight.Fields = map[string]interface{}{"title": map[string]interface{}{}}
	searchRequest.Highlight = queryHighLight

	// Execute the search request and retrieve the response
	resp, httpRes, err := resource.ManticoreClient.SearchAPI.Search(context.Background()).SearchRequest(*searchRequest).Execute()
	if err != nil {
		return err
	}

	// Log the HTTP response to the console
	fmt.Printf("%v\n\n", httpRes)

	// Log the search results to the console
	fmt.Printf("%v\n\n", resp)

	return nil
}
