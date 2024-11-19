package usecase

import (
	"context"
	corev1 "k8s.io/api/core/v1"
)

type Mutator[T corev1.Pod | corev1.ConfigMap] interface {
	Apply(ctx context.Context, resource T) (T, error)
}
