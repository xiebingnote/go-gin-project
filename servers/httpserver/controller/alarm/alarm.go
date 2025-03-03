package alarm

import (
	"os"

	"github.com/xiebingnote/go-gin-project/library/resource"
	resp "github.com/xiebingnote/go-gin-project/library/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func List(c *gin.Context) {
	reqID := uuid.NewString()
	//todo

	resource.LoggerService.Info("Application started", zap.Int("pid", os.Getpid()))
	resp.NewOKResp(c, "test", reqID)
}
