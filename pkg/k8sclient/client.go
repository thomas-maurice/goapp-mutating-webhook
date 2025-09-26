package k8sclient

import (
	"context"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	cachedClient kubernetes.Interface
	clientMutex  = &sync.RWMutex{}
)

type KubernetesClientKey struct{}

var (
	k8sClientKey KubernetesClientKey = KubernetesClientKey{}
)

func GetConfig(inCluster bool, kubeconfig string) (*rest.Config, error) {
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

	return config, err
}

func GetClient(inCluster bool, kubeconfig string) (kubernetes.Interface, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if cachedClient != nil {
		return cachedClient, nil
	}

	config, err := GetConfig(inCluster, kubeconfig)
	if err != nil {
		return nil, err
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
