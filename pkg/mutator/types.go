package mutator

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

func maybe[T any](v T) *T { return &v }

type Mutation[T runtime.Object] interface {
	ShouldMutate(ctx context.Context, object T) (bool, error)
	Mutate(ctx context.Context, object T) (T, error)
}

type BaseMutation[T runtime.Object] struct{}

func (m BaseMutation[T]) ShouldMutate(ctx context.Context, object T) (bool, error) {
	return true, nil
}

type Admission[T runtime.Object] interface {
	ShouldRun(ctx context.Context, object T) (bool, error)
	Admit(ctx context.Context, newObject T, oldObject T) (T, error)
}

type BaseAdmission[T runtime.Object] struct{}

func (m BaseAdmission[T]) ShouldRun(ctx context.Context, object T) (bool, error) {
	return true, nil
}
