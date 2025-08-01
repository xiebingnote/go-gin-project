package flink

import (
	"github.com/gin-gonic/gin"

	"github.com/xiebingnote/go-gin-project/servers/httpserver/controller/flink/api"
)

func Router(r *gin.RouterGroup) {
	cdc := r.Group("/cdc")
	{
		cdc.POST("", api.CreateFlinkCDCConfig)
		cdc.GET("", api.GetFlinkCDCConfigInfo)
		cdc.GET("/list", api.GetFlinkCDCConfigList)
		cdc.PUT("", api.AddFlinkCDCJob)
		cdc.DELETE("", api.DeleteFlinkCDCConfig)
	}
}
