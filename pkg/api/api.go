package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/config"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/metrics"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/mutator"
)

type Api struct {
	logger *slog.Logger
	engine *gin.Engine
	config *config.Config
}

func NewAPI(logger *slog.Logger, config *config.Config) (*Api, error) {
	a := &Api{
		logger: logger,
		engine: gin.New(),
		config: config,
	}

	a.engine.Use(gin.Recovery())
	a.engine.Use(sloggin.NewWithConfig(logger, sloggin.Config{
		WithRequestBody:    false,
		WithUserAgent:      false,
		WithRequestHeader:  false,
		WithResponseBody:   false,
		WithResponseHeader: false,
	}))

	a.engine.POST("/mutate", a.Mutate)

	a.engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{})
	})

	metrics.MetricsMiddleware.Use(a.engine)

	return a, nil
}

func (a *Api) Mutate(ctx *gin.Context) {
	ctx.Set("logger", log.GetLogger().With())

	request, err := mutator.GetAdmissionReview(ctx)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		ctx.Writer.Write([]byte(fmt.Sprintf("bad request: %s", err)))
		return
	}

	err = mutator.CheckRequest(request)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		ctx.Writer.Write([]byte(fmt.Sprintf("bad request: %s", err)))
		return
	}

	response, err := mutator.MutatePod(a.config, request)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		ctx.Writer.Write([]byte(fmt.Sprintf("internal server error: %s", err)))
		return
	}

	ctx.JSON(200, response)
}

func (a *Api) Serve(addr, cert, key string) error {
	return a.engine.RunTLS(addr, cert, key)
}
