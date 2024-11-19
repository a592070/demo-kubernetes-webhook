package usecase

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type sidecarMutator struct {
	logger logr.Logger
}

func NewSidecarMutator(logger logr.Logger) Mutator[corev1.Pod] {
	return &sidecarMutator{
		logger: logger,
	}
}

func (s *sidecarMutator) Apply(ctx context.Context, pod corev1.Pod) (corev1.Pod, error) {
	logger := s.logger.WithValues("name", pod.Name,
		"namespace", pod.Namespace)
	logger.V(1).Info("processing pod",
		AnnotationSidecarInjectName, pod.Annotations[AnnotationSidecarInjectName],
		AnnotationSidecarInjectValue, pod.Annotations[AnnotationSidecarInjectValue])

	// Remove previous sidecar firstly
	sidecarName := AnnotationDefaultSidecarName
	if pod.Annotations[AnnotationSidecarInjectName] != "" {
		sidecarName = pod.Annotations[AnnotationSidecarInjectName]
	}
	pod.Spec.Containers = s.removeSidecar(pod.Spec.Containers, sidecarName)

	if len(pod.Annotations[AnnotationSidecarInjectValue]) == 0 {
		return pod, nil
	}

	// Append sidecar into pod's containers
	sidecar := corev1.Container{}
	err := json.Unmarshal([]byte(pod.Annotations[AnnotationSidecarInjectValue]), &sidecar)
	if err != nil {
		logger.Error(err, "failed to unmarshal sidecar container",
			AnnotationSidecarInjectName, pod.Annotations[AnnotationSidecarInjectName],
			AnnotationSidecarInjectValue, pod.Annotations[AnnotationSidecarInjectValue])
		return pod, errors.Wrapf(err, "failed to unmarshal sidecar annotation %s", AnnotationSidecarInjectValue)
	}
	sidecarName = sidecar.Name
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)

	// Assign new sidecar name to annotation
	pod.Annotations[AnnotationSidecarInjectName] = sidecarName

	return pod, nil
}

func (s *sidecarMutator) removeSidecar(containers []corev1.Container, sidecarName string) []corev1.Container {
	removeIdx := -1
	for i, container := range containers {
		if container.Name == sidecarName {
			removeIdx = i
		}
	}
	if removeIdx == -1 {
		return containers
	}
	return append(containers[:removeIdx], containers[removeIdx+1:]...)
}
