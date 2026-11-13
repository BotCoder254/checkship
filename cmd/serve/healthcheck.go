package serve

import (
	"github.com/checkship-deploy/checkship/cmd/version"
	"github.com/checkship-deploy/checkship/pkg/http"
	"github.com/gin-gonic/gin"
)

type healthCheckResponse struct {
	Version string `json:"version"`
}

func (s *server) healthcheckHandler(ctx *gin.Context) {
	_ = http.Ok(ctx, healthCheckResponse{
		Version: version.Current(),
	})
}
