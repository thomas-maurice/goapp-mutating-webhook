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
	mapper "github.com/thomas-maurice/goapp-mutating-webhook/pkg/restmapper"
	webhook "github.com/thomas-maurice/goapp-mutating-webhook/pkg/webhook"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type RegisterMutationsFunc func(ctx context.Context) error
type RegisterAdmissionsFunc func(ctx context.Context) error

type Api struct {
	logger                 *slog.Logger
	engine                 *gin.Engine
	restConfig             *rest.Config
	k8sClient              kubernetes.Interface
	config                 *config.Config
	restMapper             *restmapper.DeferredDiscoveryRESTMapper
	RegisterMutationsFunc  RegisterMutationsFunc
	RegisterAdmissionsFunc RegisterAdmissionsFunc
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

func NewAPI(
	logger *slog.Logger,
	cfg *config.Config,
	restConfig *rest.Config,
	regMutFunc RegisterMutationsFunc,
	regAdmFunc RegisterAdmissionsFunc,
) (*Api, error) {
	restMapper, err := mapper.GetRESTMapper(restConfig)
	if err != nil {
		return nil, err
	}

	client, err := k8sclient.GetClient(restConfig)
	if err != nil {
		return nil, err
	}

	a := &Api{
		logger:                 logger,
		engine:                 gin.New(),
		restMapper:             restMapper,
		restConfig:             restConfig,
		k8sClient:              client,
		config:                 cfg,
		RegisterMutationsFunc:  regMutFunc,
		RegisterAdmissionsFunc: regAdmFunc,
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
	ctx = k8sclient.ToContext(ctx, client)
	ctx = k8sconfig.ToContext(ctx, restConfig)
	ctx = mapper.ToContext(ctx, restMapper)
	ctx = config.ToContext(ctx, cfg)
	ctx = EngineToContext(ctx, a.engine)

	if a.RegisterMutationsFunc != nil {
		err = a.RegisterMutationsFunc(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to register mutation functions: %w", err)
		}
	}

	if a.RegisterAdmissionsFunc != nil {
		err = a.RegisterAdmissionsFunc(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to register admission functions: %w", err)
		}
	}

	return a, nil
}

func RegisterMutationHook[T runtime.Object](
	ctx context.Context,
	apiPath string,
	hooks []webhook.Mutation[T],
) error {
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

		admissionReview, err := webhook.DecodeAdmissionReview(ctx)
		if err != nil {
			ctx.Status(http.StatusBadRequest)
			fmt.Fprintf(ctx.Writer, "bad request: %s", err)

			return
		}

		obj, _, err := webhook.CheckAndDecodeRequest[T](admissionReview, restMapper)
		if err != nil {
			panic(err)
		}

		response, err := webhook.Mutate(mutCtx, admissionReview, obj, hooks)
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

func RegisterAdmissionHook[T runtime.Object](
	ctx context.Context,
	apiPath string,
	hooks []webhook.Admission[T],
) error {
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

		admissionReview, err := webhook.DecodeAdmissionReview(ctx)
		if err != nil {
			ctx.Status(http.StatusBadRequest)
			fmt.Fprintf(ctx.Writer, "bad request: %s", err)

			return
		}

		oldObj, newObj, err := webhook.CheckAndDecodeRequest[T](admissionReview, restMapper)
		if err != nil {
			panic(err)
		}

		response, err := webhook.Admit(mutCtx, admissionReview, oldObj, newObj, hooks)
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
