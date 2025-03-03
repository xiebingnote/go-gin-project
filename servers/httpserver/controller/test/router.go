package test

import "github.com/gin-gonic/gin"

func Router(r *gin.RouterGroup) {
	test := r.Group("/test")
	{
		test.GET("", Test)
	}
}
