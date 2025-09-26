package mutator

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type PodMutation struct {
	BaseMutation[*corev1.Pod]
}

func (m PodMutation) Mutate(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations["foo"] = "coucoumdr"

	return pod, nil
}
