package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/config"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/k8sclient"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/metrics"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/mutator"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type Api struct {
	logger *slog.Logger
	Engine *gin.Engine
	config *config.Config

	restMapper *restmapper.DeferredDiscoveryRESTMapper

	RegisterHooksFunc func(ctx context.Context) error
}

func NewAPI(logger *slog.Logger, config *config.Config, restConfig *rest.Config) (*Api, error) {
	restMapper, err := mutator.NewRESTMapper(restConfig)
	if err != nil {
		return nil, err
	}

	a := &Api{
		logger:     logger,
		Engine:     gin.New(),
		config:     config,
		restMapper: restMapper,
	}

	a.Engine.Use(gin.Recovery())
	a.Engine.Use(sloggin.NewWithConfig(logger, sloggin.Config{
		WithRequestBody:    false,
		WithUserAgent:      false,
		WithRequestHeader:  false,
		WithResponseBody:   false,
		WithResponseHeader: false,
	}))

	/*RegisterMutationHook(a.Engine, a.config, restMapper, "/mutate", []mutator.Mutation[*corev1.Pod]{
		mutator.PodMutation{},
	})*/

	/*RegisterMutationHook(a.Engine, a.config, restMapper, "/mutate", []mutator.Mutation[*unstructured.Unstructured]{
		mutator.UnstructuredMutation{},
	})*/

	//nolint:staticcheck
	ctx := context.WithValue(context.Background(), "xxx-config-xxx", config)
	//nolint:staticcheck
	ctx = context.WithValue(ctx, "xxx-restmapper-xxx", restMapper)
	//nolint:staticcheck
	ctx = context.WithValue(ctx, "xxx-engine-xxx", a.Engine)

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

	a.Engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{})
	})

	metrics.MetricsMiddleware.Use(a.Engine)

	return a, nil
}

func RegisterMutationHook[T runtime.Object](
	engine *gin.Engine,
	cfg *config.Config,
	restMapper *restmapper.DeferredDiscoveryRESTMapper,
	apiPath string,
	hooks []mutator.Mutation[T],
) error {
	k8sClient, err := k8sclient.GetClient(true, "")
	if err != nil {
		panic(err)
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
			//nolint:errcheck
			fmt.Fprintf(ctx.Writer, "failed to marshal response: %s", err)

			return
		}

		//nolint:errcheck
		ctx.Writer.Write(b)
	})

	return nil
}

func RegisterMutationHookContext[T runtime.Object](
	ctx context.Context,
	apiPath string,
	hooks []mutator.Mutation[T],
) error {
	k8sClient, err := k8sclient.GetClient(true, "")
	if err != nil {
		panic(err)
	}

	cfgCtx := ctx.Value("xxx-config-xxx")

	cfg, ok := cfgCtx.(*config.Config)
	if !ok {
		return errors.New("failed to extract config from context")
	}

	restMapperCtx := ctx.Value("xxx-restmapper-xxx")

	restMapper, ok := restMapperCtx.(*restmapper.DeferredDiscoveryRESTMapper)
	if !ok {
		return errors.New("failed to extract config from context")
	}

	engineCtx := ctx.Value("xxx-engine-xxx")

	engine, ok := engineCtx.(*gin.Engine)
	if !ok {
		return errors.New("failed to extract config from context")
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
	return a.Engine.RunTLS(addr, cert, key)
}
