package k8sclient

import (
	"context"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	cachedClient kubernetes.Interface
	clientMutex  = &sync.RWMutex{}
)

type KubernetesClientKey struct{}

var (
	k8sClientKey KubernetesClientKey = KubernetesClientKey{}
)

func GetClient(config *rest.Config) (kubernetes.Interface, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if cachedClient != nil {
		return cachedClient, nil
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cachedClient = clientset

	return clientset, err
}

func FromContext(ctx context.Context) (kubernetes.Interface, error) {
	cs, ok := ctx.Value(k8sClientKey).(kubernetes.Interface)
	if ok {
		return cs, nil
	}

	return nil, nil
}

func ToContext(ctx context.Context, clientset kubernetes.Interface) context.Context {
	return context.WithValue(ctx, k8sClientKey, clientset)
}
