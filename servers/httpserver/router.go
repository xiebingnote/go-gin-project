package httpserver

import (
	"github.com/gin-gonic/gin"
	"project/servers/httpserver/controller"
)

func Router(r *gin.RouterGroup) {
	controller.Router(r.Group("/v1"))
}
