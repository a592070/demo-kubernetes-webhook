package usecase

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	AnnotationSidecarInjectValue = "sidecar.example/inject-value"
	AnnotationSidecarInjectName  = "sidecar.example/inject-name"
	AnnotationDefaultSidecarName = "auto-inject-sidecar"
)

type SidecarMutator struct {
}

func NewSidecarMutator() *SidecarMutator {
	return &SidecarMutator{}
}

func (s *SidecarMutator) Apply(ctx context.Context, pod corev1.Pod) (corev1.Pod, error) {
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
		return pod, errors.Wrapf(err, "failed to unmarshal sidecar annotation %s", AnnotationSidecarInjectValue)
	}
	sidecarName = sidecar.Name
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)

	// Assign new sidecar name to annotation
	pod.Annotations[AnnotationSidecarInjectName] = sidecarName

	return pod, nil
}

func (s *SidecarMutator) removeSidecar(containers []corev1.Container, sidecarName string) []corev1.Container {
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
