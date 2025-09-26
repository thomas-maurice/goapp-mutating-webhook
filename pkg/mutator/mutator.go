package mutator

import (
	"context"
	"errors"
	"fmt"

	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/config"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/metrics"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	defaultScheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	scheme       *runtime.Scheme
	codecs       serializer.CodecFactory
	deserializer runtime.Decoder
)

func init() {
	scheme = defaultScheme.Scheme

	err := admissionv1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
	err = metav1.AddMetaToScheme(scheme)
	if err != nil {
		panic(err)
	}

	codecs = serializer.NewCodecFactory(scheme)
	deserializer = codecs.UniversalDeserializer()
}

// MutatePod will mutate the pod according to the incoming spec.
// We should only get pod review to this function
func MutateObject[T runtime.Object](cfg *config.Config, admissionRequest *admissionv1.AdmissionReview, obj T, mutations []Mutation[T]) (*admissionv1.AdmissionReview, error) {
	namespace := admissionRequest.Request.Namespace

	logger := log.GetLogger().With(
		"namespace", namespace,
		"kind", admissionRequest.Request.Resource,
	)

	// This should be checked prior to calling the function but you can
	// never be too sure innit
	/*err := CheckRequest(admissionRequest, obj)
	if err != nil {
		metrics.PodMutation.Inc([]string{namespace, metrics.LabelFailure})
		return nil, err
	}*/

	object, _, err := deserializer.Decode(admissionRequest.Request.Object.Raw, nil, nil)
	if err != nil {
		logger.Error("could not unmarshal object", "error", err)
		//nolint:errcheck
		metrics.PodMutation.Inc([]string{namespace, metrics.LabelFailure})

		return nil, fmt.Errorf("could not unmarshal object: %w", err)
	}

	casted, ok := object.(T)
	if !ok {
		return nil, errors.New("could not do type assertion")
	}

	originalObject := casted.DeepCopyObject()

	lastMutatedObject := casted.DeepCopyObject().(T)

	for _, mutation := range mutations {
		shouldMutate, err := mutation.ShouldMutate(context.Background(), casted)
		if err != nil {
			logger.Error("failed to run shouldApply", "error", err)

			return nil, err
		}

		if !shouldMutate {
			logger.Warn("skipping (skipped)")
		}

		workingCopyObject := lastMutatedObject.DeepCopyObject().(T)

		lastMutatedObject, err = mutation.Mutate(context.Background(), workingCopyObject)
		if err != nil {
			logger.Error("failed to run mutation", "error", err)

			return nil, err
		}
	}

	patch, err := ComputePatch(originalObject, lastMutatedObject)
	if err != nil {
		logger.Error("failed to compute patch", "error", err)

		return nil, err
	}

	admissionResponse := &admissionv1.AdmissionResponse{}

	admissionResponse.Allowed = true
	admissionResponse.PatchType = maybe(admissionv1.PatchTypeJSONPatch)
	admissionResponse.Patch = patch

	// Wrap the response in the proper object type
	var admissionReviewResponse admissionv1.AdmissionReview

	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionRequest.Request.UID

	//nolint:errcheck
	metrics.PodMutation.Inc([]string{namespace, metrics.LabelSuccess})

	return &admissionReviewResponse, nil
}

func Mutate[T runtime.Object](ctx context.Context, cfg *config.Config, admissionRequest *admissionv1.AdmissionReview, obj T, mutations []Mutation[T]) (*admissionv1.AdmissionReview, error) {
	namespace := admissionRequest.Request.Namespace

	logger := log.GetLogger().With(
		"namespace", namespace,
		"kind", admissionRequest.Request.Resource,
	)

	originalObject := obj.DeepCopyObject().(T)

	lastMutatedObject := obj.DeepCopyObject().(T)

	for _, mutation := range mutations {
		shouldMutate, err := mutation.ShouldMutate(ctx, obj)
		if err != nil {
			logger.Error("failed to run shouldApply", "error", err)

			return nil, err
		}

		if !shouldMutate {
			logger.Warn("skipping (skipped)")
		}

		workingCopyObject := lastMutatedObject.DeepCopyObject().(T)

		lastMutatedObject, err = mutation.Mutate(ctx, workingCopyObject)
		if err != nil {
			logger.Error("failed to run mutation", "error", err)

			return nil, err
		}
	}

	patch, err := ComputePatch(originalObject, lastMutatedObject)
	if err != nil {
		logger.Error("failed to compute patch", "error", err)

		return nil, err
	}

	admissionResponse := &admissionv1.AdmissionResponse{}

	admissionResponse.Allowed = true
	admissionResponse.PatchType = maybe(admissionv1.PatchTypeJSONPatch)
	admissionResponse.Patch = patch

	// Wrap the response in the proper object type
	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionRequest.Request.UID

	metrics.PodMutation.Inc([]string{namespace, metrics.LabelSuccess})

	return &admissionReviewResponse, nil
}
