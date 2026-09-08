package service

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

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

	// Create and validate log directories
	if err := CreateDirectories(config.StarRocksConfig.StarRocks.FileDir); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to create starrocks directories: %v", err))
	}
}

// CreateDirectories creates and validates log directories.
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - error: An error if directory creation fails, nil otherwise
func CreateDirectories(dir string) error {

	// Check if directory already exists
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("file path exists but is not a directory: %s", dir)
		}
		// Directory exists, check permissions
		return ValidateDirectoryPermissions(dir)
	}

	// Create directory with proper permissions
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Validate the created directory
	if err := ValidateDirectoryPermissions(dir); err != nil {
		return fmt.Errorf("directory validation failed: %w", err)
	}

	return nil
}

// ValidateDirectoryPermissions validates that the log directory has proper permissions.
//
// Parameters:
//   - dirPath: The directory path to validate
//
// Returns:
//   - error: An error if validation fails, nil otherwise
func ValidateDirectoryPermissions(dirPath string) error {
	// Test write permissions by creating a temporary file
	file, err := os.CreateTemp(dirPath, ".write_test-*")
	if err != nil {
		return fmt.Errorf("directory %s is not writable: %w", dirPath, err)
	}
	return errors.Join(file.Close(), os.Remove(file.Name()))
}
