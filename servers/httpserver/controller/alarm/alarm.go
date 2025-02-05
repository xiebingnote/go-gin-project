package alarm

import (
	"net/http"

	"project/library/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func List(c *gin.Context) {
	reqID := uuid.NewString()
	//todo

	c.JSON(http.StatusOK, types.NewOKRestResp("test success", reqID))
}
