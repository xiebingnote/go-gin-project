package service

import (
	"context"

	"project/library/resource"

	mapset "github.com/deckarep/golang-set/v2"
	cmap "github.com/orcaman/concurrent-map/v2"
)

// InitCommon initializes the common resources.
//
// This function creates a new set for strings and a new concurrent map,
// and stores them in the resource package.
//
// Parameters:
//   - _ context.Context, the context passed to this function is ignored.
func InitCommon(_ context.Context) {
	// Create a new set for strings
	testStringMapSet := mapset.NewSet[string]()
	// Store the set in the resource package
	resource.TestStringMapSet = &testStringMapSet

	// Create a new concurrent map
	testCMap := cmap.New[string]()
	// Store the map in the resource package
	resource.TestCMap = &testCMap
}
