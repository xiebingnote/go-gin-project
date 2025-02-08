package alarm

import (
	"net/http"
	"os"

	"project/library/resource"
	resp "project/library/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func List(c *gin.Context) {
	reqID := uuid.NewString()
	//todo

	resource.LoggerService.Info("Application started", zap.Int("pid", os.Getpid()))
	c.JSON(http.StatusOK, resp.NewOKRestResp("test success", reqID))
}
