package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/xiebingnote/go-gin-project/servers/httpserver/controller/flink"
)

// Router registers the routes for the controllers.
//
// It is expected that the provided gin.RouterGroup is a subgroup of the main
// router.
func Router(r *gin.RouterGroup) {
	// Route for the alarm controller.
	//alarm.Router(r.Group("/alarm"))
	//test.Router(r.Group("/test"))
	flink.Router(r.Group("/flink"))
}
