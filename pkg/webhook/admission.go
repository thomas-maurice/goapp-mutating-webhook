package webhook

import (
	"context"

	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/metrics"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Admit[T runtime.Object](ctx context.Context, admissionRequest *admissionv1.AdmissionReview, oldObj T, newObj T, admissions []Admission[T]) (*admissionv1.AdmissionReview, error) {
	namespace := admissionRequest.Request.Namespace

	logger := log.GetLogger().With(
		"namespace", namespace,
		"kind", admissionRequest.Request.Resource,
	)

	allowed := true
	var lastErr error

	for _, admission := range admissions {
		shouldMutate, err := admission.ShouldRun(ctx, oldObj)
		if err != nil {
			logger.Error("failed to run shouldRun", "error", err)

			return nil, err
		}

		if !shouldMutate {
			logger.Warn("skipping (skipped)")
			continue
		}

		// In case people do a little trolling in the handlers
		oldObjCopy := oldObj.DeepCopyObject().(T)
		newObjCopy := newObj.DeepCopyObject().(T)

		_, err = admission.Admit(ctx, oldObjCopy, newObjCopy)
		if err != nil {
			logger.Error("failed to run admission", "error", err)
			lastErr = err
			allowed = false
		}
	}

	admissionResponse := &admissionv1.AdmissionResponse{}

	admissionResponse.Allowed = allowed
	if !allowed {
		admissionResponse.Result.Message = lastErr.Error()
	}

	// Wrap the response in the proper object type
	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionRequest.Request.UID

	// nolint:errcheck
	metrics.PodMutation.Inc([]string{namespace, metrics.LabelSuccess})

	return &admissionReviewResponse, nil
}
