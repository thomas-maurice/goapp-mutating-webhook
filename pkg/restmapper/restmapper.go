package restmapper

import (
	"context"
	"sync"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

var (
	cachedMapper *restmapper.DeferredDiscoveryRESTMapper
	mapperMutex  = &sync.RWMutex{}
)

type RESTMapperKey struct{}

var (
	mapperKey RESTMapperKey = RESTMapperKey{}
)

// Returns a RESTMapper with in-memory discovery caching. This is going to be used to
// check the Group Version Kind / Group Version Resource mappings to make sure that
// the apiserver sends us consistent data.
func GetRESTMapper(cfg *rest.Config) (*restmapper.DeferredDiscoveryRESTMapper, error) {
	mapperMutex.Lock()
	defer mapperMutex.Unlock()

	if cachedMapper != nil {
		return cachedMapper, nil
	}

	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	memCacheClient := memory.NewMemCacheClient(dc)

	cachedMapper = restmapper.NewDeferredDiscoveryRESTMapper(memCacheClient)

	return cachedMapper, nil
}

func FromContext(ctx context.Context) (*restmapper.DeferredDiscoveryRESTMapper, error) {
	rm, ok := ctx.Value(mapperKey).(*restmapper.DeferredDiscoveryRESTMapper)
	if ok {
		return rm, nil
	}

	return nil, nil
}

func ToContext(ctx context.Context, mapper *restmapper.DeferredDiscoveryRESTMapper) context.Context {
	return context.WithValue(ctx, mapperKey, mapper)
}
