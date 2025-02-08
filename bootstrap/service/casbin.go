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
func InitEnforcer(_ context.Context) {
	adapter, err := gormadapter.NewAdapterByDB(resource.MySQLClient)
	if err != nil {
		panic(err)
	}

	enforcer, err := casbin.NewEnforcer("conf/servicer/casbin.conf", adapter)
	if err != nil {
		panic(err)
	}

	resource.Enforcer = enforcer
	return
}
