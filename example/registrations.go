package main

import (
	"context"

	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/api"
	unstructuredMutation "github.com/thomas-maurice/goapp-mutating-webhook/pkg/contrib/mutations/unstructured"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/webhook"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func RegisterMutations(ctx context.Context) error {
	err := api.RegisterMutationHookContext(ctx, "/mutate", []webhook.Mutation[*unstructured.Unstructured]{
		unstructuredMutation.UnstructuredMutation{},
	})

	return err
}

func RegisterAdmissions(ctx context.Context) error {
	return nil
}
