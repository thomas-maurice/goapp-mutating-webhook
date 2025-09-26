package mutator

import (
	"errors"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cached "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// Returns a RESTMapper with in-memory discovery caching. This is going to be used to
// check the Group Version Kind / Group Version Resource mappings to make sure that
// the apiserver sends us consistent data.
func NewRESTMapper(cfg *rest.Config) (*restmapper.DeferredDiscoveryRESTMapper, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	memCacheClient := cached.NewMemCacheClient(dc)

	return restmapper.NewDeferredDiscoveryRESTMapper(memCacheClient), nil
}

// CheckAndDecodeRequest decodes both the new and old objects, ensuring the GVR matches T.
func CheckAndDecodeRequest[T runtime.Object](
	ar *admissionv1.AdmissionReview,
	mapper *restmapper.DeferredDiscoveryRESTMapper,
) (newObj T, oldObj T, err error) {
	var zero T

	if ar == nil || ar.Request == nil {
		return zero, zero, errors.New("invalid AdmissionReview: nil request")
	}

	// Convert metav1 GVR/GVK -> schema types
	actual := schema.GroupVersionResource(ar.Request.Resource)
	gvk := schema.GroupVersionKind{
		Group:   ar.Request.Kind.Group,
		Version: ar.Request.Kind.Version,
		Kind:    ar.Request.Kind.Kind,
	}

	// Map GVK -> expected GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return zero, zero, fmt.Errorf("unable to resolve GVR from GVK %v: %w", gvk, err)
	}

	expected := mapping.Resource

	if actual != expected {
		return zero, zero, fmt.Errorf("invalid resource type: got %v, expected %v", actual, expected)
	}

	// For CREATE mutation webhooks, we decode the new object
	if len(ar.Request.Object.Raw) > 0 {
		var obj runtime.Object
		// This is ugly as hell but the only way to get *unstructed.Unstructured to work properly
		if _, ok := any(zero).(*unstructured.Unstructured); ok {
			// Force decode into unstructured
			u := &unstructured.Unstructured{}
			err := u.UnmarshalJSON(ar.Request.Object.Raw)
			if err != nil {
				return zero, zero, fmt.Errorf("failed to unmarshal into unstructured: %w", err)
			}

			obj = u
		} else {
			decoded, _, err := deserializer.Decode(ar.Request.Object.Raw, &gvk, nil)
			if err != nil {
				return zero, zero, fmt.Errorf("failed to decode new object: %w", err)
			}

			obj = decoded
		}

		casted, ok := obj.(T)
		if !ok {
			return zero, zero, fmt.Errorf("decoded new object is %T, not expected %T", obj, zero)
		}

		newObj = casted
	}

	// In case of UPDATE/DELETE we also want to get the old object
	if len(ar.Request.OldObject.Raw) > 0 {
		var obj runtime.Object
		// This is ugly as hell but the only way to get *unstructed.Unstructured to work properly
		if _, ok := any(zero).(*unstructured.Unstructured); ok {
			u := &unstructured.Unstructured{}
			err := u.UnmarshalJSON(ar.Request.OldObject.Raw)
			if err != nil {
				return newObj, zero, fmt.Errorf("failed to unmarshal old object into unstructured: %w", err)
			}

			obj = u
		} else {
			decoded, _, err := deserializer.Decode(ar.Request.OldObject.Raw, &gvk, nil)
			if err != nil {
				return newObj, zero, fmt.Errorf("failed to decode old object: %w", err)
			}

			obj = decoded
		}

		casted, ok := obj.(T)
		if !ok {
			return newObj, zero, fmt.Errorf("decoded old object is %T, not expected %T", obj, zero)
		}

		oldObj = casted
	}

	return newObj, oldObj, nil
}
