package httpserver

import (
	"project/servers/httpserver/controller"

	"github.com/gin-gonic/gin"
)

func Router(r *gin.RouterGroup) {
	controller.Router(r.Group("/v1"))
	controller.Router(r.Group("/v2"))
}
