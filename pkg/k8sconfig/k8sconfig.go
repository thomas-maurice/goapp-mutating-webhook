package k8sconfig

import (
	"context"
	"sync"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	cachedConfig *rest.Config
	configMutex  = &sync.RWMutex{}
)

type KubernetesConfigKey struct{}

var (
	k8sConfigKey KubernetesConfigKey = KubernetesConfigKey{}
)

func GetConfig(inCluster bool, kubeconfig string) (*rest.Config, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	if cachedConfig != nil {
		return cachedConfig, nil
	}

	var (
		config *rest.Config
		err    error
	)

	if inCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	cachedConfig = config

	return config, err
}

func FromContext(ctx context.Context) (*rest.Config, error) {
	cs, ok := ctx.Value(k8sConfigKey).(*rest.Config)
	if ok {
		return cs, nil
	}

	return nil, nil
}

func ToContext(ctx context.Context, config *rest.Config) context.Context {
	return context.WithValue(ctx, k8sConfigKey, config)
}
