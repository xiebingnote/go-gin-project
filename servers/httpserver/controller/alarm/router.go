package alarm

import "github.com/gin-gonic/gin"

func Router(r *gin.RouterGroup) {
	r.GET("/list", List)
	r.Group("/high_alarm")
	{
		r.GET("/high", List)
	}

	r.Group("/low_alarm")
	{
		r.GET("/low", List)
	}
}
