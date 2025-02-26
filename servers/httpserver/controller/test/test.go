package test

import (
	resp "go-gin-project/library/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Test(c *gin.Context) {
	reqID := uuid.NewString()
	resp.NewOKResp(c, "test", reqID)

}
