package mutator

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/dustin/go-humanize"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/config"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/metrics"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	goMemLimitAnnotation       = "webhooks.maurice.fr/adjusted-GOMEMLIMIT"
	goMemLimitFactorAnnotation = "webhooks.maurice.fr/adjusted-GOMEMLIMIT-factor"
	goMaxProcsAnnotation       = "webhooks.maurice.fr/adjusted-GOMAXPROCS"
)

// MutatePod will mutate the pod according to the incoming spec.
// We should only get pod review to this function
func MutatePod(cfg *config.Config, admissionRequest *admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error) {
	namespace := admissionRequest.Request.Namespace

	logger := log.GetLogger().With(
		"namespace", namespace,
		"kind", admissionRequest.Request.Resource,
	)

	// This should be checked prior to calling the function but you can
	// never be too sure innit
	err := CheckRequest(admissionRequest)
	if err != nil {
		metrics.PodMutation.Inc([]string{namespace, metrics.LabelFailure})
		return nil, err
	}

	deserializer := codecs.UniversalDeserializer()

	podObject := admissionRequest.Request.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := deserializer.Decode(podObject, nil, &pod); err != nil {
		logger.Error("could not unmarshal pod", "error", err)
		metrics.PodMutation.Inc([]string{namespace, metrics.LabelFailure})
		return nil, fmt.Errorf("could not unmarshal pod: %w", err)
	}

	patches := make([]Patch, 0)

	// we create the admission response object
	// this will contain the json patch we want to apply to the object
	admissionResponse := &admissionv1.AdmissionResponse{}

	// We just replace the entire `env`, it's easier than issuing
	// a bunch of one shot patches and avoid having to take
	// into account things like "is the env list empty" and whatnot
	containerEnv := pod.Spec.Containers[0].Env
	additionalEnv := make(map[string]struct {
		Value string
		Set   bool
	})

	// Same approach as the containerEnv
	podAnnotations := pod.ObjectMeta.Annotations
	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}
	additionalAnnotations := make(map[string]struct {
		Value string
		Set   bool
	})

	// Same approach as the containerEnv
	podLabels := pod.ObjectMeta.Labels
	if podLabels == nil {
		podLabels = make(map[string]string)
	}
	additionalLabels := make(map[string]struct {
		Value string
		Set   bool
	})

	if pod.Spec.Containers[0].Resources.Requests.Cpu() != nil {
		logger.Info(
			"GOMACPROCS environment variable set",
			"requested_cpu", pod.Spec.Containers[0].Resources.Requests.Memory().MilliValue(),
			"adjusted_value", pod.Spec.Containers[0].Resources.Requests.Cpu().Value(),
		)
		additionalEnv["GOMAXPROCS"] = struct {
			Value string
			Set   bool
		}{
			Value: fmt.Sprintf("%d", pod.Spec.Containers[0].Resources.Requests.Cpu().Value()),
		}

		additionalAnnotations[goMaxProcsAnnotation] = struct {
			Value string
			Set   bool
		}{
			Value: fmt.Sprintf("%d", pod.Spec.Containers[0].Resources.Requests.Cpu().Value()),
		}
	} else {
		logger.Info("no CPU requests set -- skipping GOMAXPROCS")
	}

	if pod.Spec.Containers[0].Resources.Requests.Memory() != nil {
		// compute the value according to the factor in the configuration
		newMemLimit := int64(math.Ceil(cfg.GoMemLimitFactor * float64(pod.Spec.Containers[0].Resources.Requests.Memory().Value())))
		logger.Info(
			"GOMEMLIMIT environment variable set",
			"requested_memory", pod.Spec.Containers[0].Resources.Requests.Memory().Value(),
			"adjusted_value", newMemLimit,
			"factor", cfg.GoMemLimitFactor,
		)
		additionalEnv["GOMEMLIMIT"] = struct {
			Value string
			Set   bool
		}{
			Value: fmt.Sprintf("%d", newMemLimit),
		}

		additionalAnnotations[goMemLimitAnnotation] = struct {
			Value string
			Set   bool
		}{
			Value: humanize.Bytes(uint64(newMemLimit)),
		}

		additionalAnnotations[goMemLimitFactorAnnotation] = struct {
			Value string
			Set   bool
		}{
			Value: fmt.Sprintf("%.2f", cfg.GoMemLimitFactor),
		}
	} else {
		logger.Info("no memory requests set -- skipping GOMEMLIMIT")
	}

	// Patch the environment variables set
	for idx, env := range containerEnv {
		if _, ok := additionalEnv[env.Name]; ok {
			containerEnv[idx].Value = additionalEnv[env.Name].Value
			entry := additionalEnv[env.Name]
			entry.Set = true
			additionalEnv[env.Name] = entry
		}
	}

	for k, v := range additionalEnv {
		if !v.Set {
			containerEnv = append(containerEnv, corev1.EnvVar{
				Name:  k,
				Value: v.Value,
			})
		}
	}

	patches = append(patches, Patch{
		Op:    "add",
		Path:  "/spec/containers/0/env",
		Value: containerEnv,
	})

	// Patch the annotation set
	for idx, annotation := range podAnnotations {
		if _, ok := additionalAnnotations[annotation]; ok {
			podAnnotations[idx] = additionalAnnotations[annotation].Value
			entry := additionalAnnotations[annotation]
			entry.Set = true
			additionalAnnotations[annotation] = entry
		}
	}

	for k, v := range additionalAnnotations {
		if !v.Set {
			podAnnotations[k] = v.Value
		}
	}

	patches = append(patches, Patch{
		Op:    "add",
		Path:  "/metadata/annotations",
		Value: podAnnotations,
	})

	// Patch the label set
	for idx, label := range podLabels {
		if _, ok := additionalLabels[label]; ok {
			podLabels[idx] = additionalLabels[label].Value
			entry := additionalLabels[label]
			entry.Set = true
			additionalLabels[label] = entry
		}
	}

	for k, v := range additionalLabels {
		if !v.Set {
			podLabels[k] = v.Value
		}
	}

	patches = append(patches, Patch{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: podLabels,
	})

	bytesPatch, err := json.Marshal(&patches)
	if err != nil {
		logger.Error("failed to marshal patch", "error", err)
		metrics.PodMutation.Inc([]string{namespace, metrics.LabelFailure})
		return nil, fmt.Errorf("failed to marshal patch: %w", err)
	}

	admissionResponse.Allowed = true
	if len(patches) > 0 {
		admissionResponse.PatchType = maybe(admissionv1.PatchTypeJSONPatch)
		admissionResponse.Patch = bytesPatch
	}

	// Wrap the response in the proper object type
	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionRequest.Request.UID

	metrics.PodMutation.Inc([]string{namespace, metrics.LabelSuccess})

	return &admissionReviewResponse, nil
}
