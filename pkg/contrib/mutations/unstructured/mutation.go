package unstructured

import (
	"context"

	mutator "github.com/thomas-maurice/goapp-mutating-webhook/pkg/webhook"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type UnstructuredMutation struct {
	mutator.BaseMutation[*unstructured.Unstructured]
}

func (m UnstructuredMutation) Mutate(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations["unstructured"] = "true"

	obj.SetAnnotations(annotations)

	return obj, nil
}
