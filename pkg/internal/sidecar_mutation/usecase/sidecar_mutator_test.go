package usecase

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

var _ = ginkgo.Describe("inject sidecar mutation test", func() {
	var logger logr.Logger
	var sidecarInjector *sidecarMutator

	ginkgo.BeforeEach(func() {
		logger = zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true))
		sidecarInjector = &sidecarMutator{
			logger: logger,
		}
	})

	ginkgo.It("removeSidecar", func() {
		containers := []corev1.Container{
			{
				Name:  "nginx",
				Image: "nginx:latest",
			},
			{
				Name:  "sidecar",
				Image: "busybox",
			},
			{
				Name:  "anothersidecar",
				Image: "busybox",
			},
		}

		wantedRemoveScarName := "sidecar"
		ginkgo.By("remove sidecar")
		result := sidecarInjector.removeSidecar(containers, wantedRemoveScarName)
		gomega.Expect(len(containers)).To(gomega.BeEquivalentTo(len(result) + 1))
		for _, container := range result {
			gomega.Expect(container.Name).NotTo(gomega.Equal(wantedRemoveScarName))
		}
	})

	ginkgo.It("Should remove previous injected sidecar when annotation's value is missing", func() {
		ctx := context.Background()
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					AnnotationSidecarInjectName: "sidecar",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:latest",
					},
					{
						Name:  "sidecar",
						Image: "sidecar:latest",
					},
				},
			},
		}

		ginkgo.By("Apply")
		affectedPod, err := sidecarInjector.Apply(ctx, pod)

		ginkgo.By("Should success")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(affectedPod).NotTo(gomega.BeNil())

		ginkgo.By("should remove sidecar")
		gomega.Expect(affectedPod.Spec.Containers).To(gomega.HaveLen(1))
		gomega.Expect(affectedPod.Spec.Containers).NotTo(gomega.ContainElement(corev1.Container{
			Name:  "sidecar",
			Image: "sidecar:latest",
		}))

	})

	ginkgo.It("Should inject sidecar using given annotation's value", func() {
		ctx := context.Background()
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					AnnotationSidecarInjectName:  "",
					AnnotationSidecarInjectValue: `{"name": "sidecar", "image": "sidecar:latest"}`,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:latest",
					},
				},
			},
		}

		ginkgo.By("Apply")
		affectedPod, err := sidecarInjector.Apply(ctx, pod)

		ginkgo.By("Should success")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(affectedPod).NotTo(gomega.BeNil())

		ginkgo.By("Annotation's sidecar name should be renew")
		gomega.Expect(affectedPod.Annotations[AnnotationSidecarInjectName]).Should(gomega.Equal("sidecar"))

		ginkgo.By("Should add sidecar")
		gomega.Expect(affectedPod.Spec.Containers).To(gomega.HaveLen(2))
		gomega.Expect(affectedPod.Spec.Containers).To(gomega.ContainElement(corev1.Container{
			Name:  "sidecar",
			Image: "sidecar:latest",
		}))

	})

	ginkgo.It("Should replace injected sidecar using given annotation's value", func() {
		ctx := context.Background()
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					AnnotationSidecarInjectName:  "sidecar",
					AnnotationSidecarInjectValue: `{"name": "sidecar-v2", "image": "sidecar:0.0.2"}`,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:latest",
					},
					{
						Name:  "sidecar",
						Image: "sidecar:0.0.1",
					},
				},
			},
		}

		ginkgo.By("Apply")
		affectedPod, err := sidecarInjector.Apply(ctx, pod)

		ginkgo.By("Should success")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(affectedPod).NotTo(gomega.BeNil())

		ginkgo.By("Should renew annotation's sidecar name")
		gomega.Expect(affectedPod.Annotations[AnnotationSidecarInjectName]).Should(gomega.Equal("sidecar-v2"))

		ginkgo.By("Should replace sidecar")
		gomega.Expect(affectedPod.Spec.Containers).To(gomega.HaveLen(2))
		gomega.Expect(affectedPod.Spec.Containers).To(gomega.ContainElement(corev1.Container{
			Name:  "sidecar-v2",
			Image: "sidecar:0.0.2",
		}))

	})
})

func TestInjectSidecar(t *testing.T) {
	gomega.RegisterTestingT(t)
	ginkgo.RunSpecs(t, "Test InjectSidecar use case")
}
