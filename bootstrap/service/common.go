package service

import (
	"context"
	"fmt"
	"github.com/xiebingnote/go-gin-project/library/config"
	"os"
	"path/filepath"

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
	testFile := filepath.Join(dirPath, ".write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("flile directory is not writable: %s", dirPath)
	}
	err = file.Close()
	if err != nil {
		return err
	}

	// Clean up test file
	if err := os.Remove(testFile); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to remove test file %s: %v\n", testFile, err)
	}

	return nil
}
