package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/thomas-maurice/goapp-mutating-webhook/docs"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/config"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/metrics"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/mutator"

	admissionv1 "k8s.io/api/admission/v1"
)

var _ = admissionv1.AdmissionReview{}

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

	a.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	metrics.MetricsMiddleware.Use(a.engine)

	return a, nil
}

// @BasePath /

// Mutate godoc
// @Summary Mutates a pod passed through an admission request
// @Schemes http https
// @Description Modifies the pod spec of a pod, and returns an appropriate admission object
// @Accept json
// @Produce json
// @Param review_request body admissionv1.AdmissionReview true "Input model"
// @Success 200 {object} admissionv1.AdmissionReview
// @Router /mutate [post]
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
