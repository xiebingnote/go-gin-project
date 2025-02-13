package test

import (
	"net/http"

	resp "go-gin-project/library/response"

	"github.com/gin-gonic/gin"
)

func Test(c *gin.Context) {

	c.JSON(http.StatusOK, resp.NewOKRestResp(nil, ""))
}
