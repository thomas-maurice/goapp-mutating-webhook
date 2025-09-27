package podlabel

import (
	"context"

	mutator "github.com/thomas-maurice/goapp-mutating-webhook/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
)

type PodMutation struct {
	mutator.BaseMutation[*corev1.Pod]
}

func (m PodMutation) Mutate(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations["some-label"] = "bonjour, hon hon, la baguette"

	return pod, nil
}
