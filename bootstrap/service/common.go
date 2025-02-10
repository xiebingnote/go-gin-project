package service

import (
	"context"

	"project/library/resource"

	mapset "github.com/deckarep/golang-set/v2"
	cmap "github.com/orcaman/concurrent-map/v2"
)

// InitCommon initializes the common resources.
//
// This function is called by the Init function in the bootstrap package.
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
