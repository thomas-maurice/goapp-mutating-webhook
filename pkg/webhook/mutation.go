package webhook

import (
	"context"

	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/metrics"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Mutate[T runtime.Object](ctx context.Context, admissionRequest *admissionv1.AdmissionReview, obj T, mutations []Mutation[T]) (*admissionv1.AdmissionReview, error) {
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

	// nolint:errcheck
	metrics.PodMutation.Inc([]string{namespace, metrics.LabelSuccess})

	return &admissionReviewResponse, nil
}
