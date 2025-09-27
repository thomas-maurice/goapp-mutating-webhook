package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/config"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/k8sclient"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/k8sconfig"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/metrics"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/mutator"
	mapper "github.com/thomas-maurice/goapp-mutating-webhook/pkg/restmapper"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type Api struct {
	logger     *slog.Logger
	engine     *gin.Engine
	config     *config.Config
	restConfig *rest.Config
	k8sClient  kubernetes.Interface

	restMapper *restmapper.DeferredDiscoveryRESTMapper

	RegisterHooksFunc func(ctx context.Context) error
}

type EngineKey struct{}

var ginEngineKey = EngineKey{}

func EngineToContext(ctx context.Context, engine *gin.Engine) context.Context {
	return context.WithValue(ctx, ginEngineKey, engine)
}

func EngineFromContext(ctx context.Context) (*gin.Engine, error) {
	engine, ok := ctx.Value(ginEngineKey).(*gin.Engine)
	if !ok {
		return nil, fmt.Errorf("failed to extract *gin.Engine from context")
	}

	return engine, nil
}

func NewAPI(logger *slog.Logger, cfg *config.Config, restConfig *rest.Config) (*Api, error) {
	restMapper, err := mapper.GetRESTMapper(restConfig)
	if err != nil {
		return nil, err
	}

	client, err := k8sclient.GetClient(restConfig)
	if err != nil {
		return nil, err
	}

	a := &Api{
		logger:     logger,
		engine:     gin.New(),
		config:     cfg,
		restMapper: restMapper,
		restConfig: restConfig,
		k8sClient:  client,
	}

	a.engine.Use(gin.Recovery())
	a.engine.Use(sloggin.NewWithConfig(logger, sloggin.Config{
		WithRequestBody:    false,
		WithUserAgent:      false,
		WithRequestHeader:  false,
		WithResponseBody:   false,
		WithResponseHeader: false,
	}))

	a.engine.GET("/readyz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{})
	})

	a.engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{})
	})

	metrics.MetricsMiddleware.Use(a.engine)

	ctx := context.Background()
	ctx = config.ToContext(ctx, cfg)
	ctx = k8sclient.ToContext(ctx, client)
	ctx = k8sconfig.ToContext(ctx, restConfig)
	ctx = mapper.ToContext(ctx, restMapper)
	ctx = EngineToContext(ctx, a.engine)

	err = RegisterMutationHookContext(ctx, "/mutate", []mutator.Mutation[*unstructured.Unstructured]{
		mutator.UnstructuredMutation{},
	})
	if err != nil {
		return nil, err
	}

	if a.RegisterHooksFunc != nil {
		err = a.RegisterHooksFunc(ctx)
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

func RegisterMutationHookContext[T runtime.Object](
	ctx context.Context,
	apiPath string,
	hooks []mutator.Mutation[T],
) error {
	cfg, err := config.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get config form context: %w", err)
	}

	restMapper, err := mapper.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get restmapper form context: %w", err)
	}

	engine, err := EngineFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gin engine from context: %w", err)
	}

	k8sClient, err := k8sclient.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client from context: %w", err)
	}

	engine.POST(apiPath, func(ctx *gin.Context) {
		mutCtx := log.ToContext(ctx, log.GetLogger())
		mutCtx = k8sclient.ToContext(mutCtx, k8sClient)

		admissionReview, err := mutator.DecodeAdmissionReview(ctx)
		if err != nil {
			ctx.Status(http.StatusBadRequest)
			fmt.Fprintf(ctx.Writer, "bad request: %s", err)

			return
		}

		obj, _, err := mutator.CheckAndDecodeRequest[T](admissionReview, restMapper)
		if err != nil {
			panic(err)
		}

		response, err := mutator.Mutate(mutCtx, cfg, admissionReview, obj, hooks)
		if err != nil {
			ctx.Status(http.StatusBadRequest)
			fmt.Fprintf(ctx.Writer, "bad request: %s", err)

			return
		}

		b, err := json.Marshal(&response)
		if err != nil {
			ctx.Status(http.StatusBadRequest)
			fmt.Fprintf(ctx.Writer, "failed to marshal response: %s", err)

			return
		}

		//nolint:errcheck
		ctx.Writer.Write(b)
	})

	return nil
}

func (a *Api) Serve(addr, cert, key string) error {
	return a.engine.RunTLS(addr, cert, key)
}
