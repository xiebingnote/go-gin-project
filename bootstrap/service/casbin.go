package service

import (
	"context"

	"project/library/resource"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

// InitEnforcer initializes the Casbin enforcer.
//
// This function creates a new Gorm adapter with the MySQL client and uses it to
// initialize the Casbin enforcer. The enforcer is then stored in the resource
// package for later use.
//
// Parameters:
//   - context.Context: Context for the initialization.
//
// Returns:
//   - No return values.
func InitEnforcer(_ context.Context) {
	// Create a Gorm adapter with the MySQL client.
	adapter, err := gormadapter.NewAdapterByDB(resource.MySQLClient)
	if err != nil {
		// Panic if the adapter creation fails.
		panic(err)
	}

	// Initialize the Casbin enforcer.
	enforcer, err := casbin.NewEnforcer("./conf/servicer/casbin.conf", adapter)
	if err != nil {
		// Panic if the enforcer initialization fails.
		panic(err)
	}

	// Store the enforcer in the resource package.
	resource.Enforcer = enforcer
	return
}
