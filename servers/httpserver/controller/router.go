package controller

import (
	"github.com/gin-gonic/gin"
	"project/servers/httpserver/controller/alarm"
)

// Router registers the routes for the controllers.
//
// It is expected that the provided gin.RouterGroup is a sub-group of the main
// router.
func Router(r *gin.RouterGroup) {
	// Route for the alarm controller.
	alarm.Router(r.Group("/alarm"))
}
