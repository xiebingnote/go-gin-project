package test

import (
	"net/http"

	"project/library/types"

	"github.com/gin-gonic/gin"
)

func Test(c *gin.Context) {

	c.JSON(http.StatusOK, types.NewOKRestResp(nil, ""))
}
