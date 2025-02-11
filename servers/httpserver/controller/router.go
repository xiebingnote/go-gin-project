package controller

import (
	"project/servers/httpserver/controller/alarm"
	"project/servers/httpserver/controller/test"

	"github.com/gin-gonic/gin"
)

// Router registers the routes for the controllers.
//
// It is expected that the provided gin.RouterGroup is a subgroup of the main
// router.
func Router(r *gin.RouterGroup) {
	// Route for the alarm controller.
	alarm.Router(r.Group("/alarm"))
	test.Router(r.Group("/test"))
}
