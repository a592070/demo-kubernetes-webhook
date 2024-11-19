package handlers

import (
	"context"
	"demo-kubernetes-webhook/pkg/internal/sidecar_mutation/usecase"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd/api"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func newScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := api.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}

var _ = Describe("handle request", func() {
	var handler *sidecarMutationHandler
	var logger logr.Logger

	BeforeEach(func() {
		logger = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))
		scheme, err := newScheme()
		Expect(err).NotTo(HaveOccurred())
		decoder := admission.NewDecoder(scheme)
		handler = &sidecarMutationHandler{
			logger:         logger,
			decoder:        decoder,
			sidecarMutator: usecase.NewSidecarMutator(logger),
		}
	})
	It("Bad request", func() {
		request := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{},
		}
		By("invoking handler")
		resp := handler.Handle(context.Background(), request)

		By("Should not allow")
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Code).To(BeEquivalentTo(http.StatusBadRequest))
	})

	It("Unable to find matched annotations", func() {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
		}
		encoded, err := json.Marshal(pod)
		Expect(err).NotTo(HaveOccurred())

		request := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UID:       "test",
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Object: runtime.RawExtension{
					Raw: encoded,
				},
			},
		}

		By("invoking handler")
		resp := handler.Handle(context.Background(), request)

		By("Should allow")
		Expect(resp.Allowed).To(BeTrue())

		By("Nothing affected")
		Expect(resp.Patches).To(HaveLen(0))
	})

	It("Should remove previous injected sidecar when annotation's value is missing", func() {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					usecase.AnnotationSidecarInjectValue: "",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:latest",
					},
					{
						Name:  "auto-inject-sidecar",
						Image: "sidecar:latest",
					},
				},
			},
		}
		encoded, err := json.Marshal(pod)
		Expect(err).NotTo(HaveOccurred())

		request := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UID:       "test",
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Object: runtime.RawExtension{
					Raw: encoded,
				},
			},
		}

		By("invoking handler")
		resp := handler.Handle(context.Background(), request)

		By("Should allow")
		Expect(resp.Allowed).To(BeTrue())

		By("Previous sidecar should be removed")
		Expect(resp.Patches).To(HaveLen(1))
		Expect(resp.Patches[0].Operation).To(BeEquivalentTo("remove"))
	})

	It("Inject sidecar", func() {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					usecase.AnnotationSidecarInjectValue: `{"name": "sidecar", "image": "sidecar:latest"}`,
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
		encoded, err := json.Marshal(pod)
		Expect(err).NotTo(HaveOccurred())

		request := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UID:       "test",
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Object: runtime.RawExtension{
					Raw: encoded,
				},
			},
		}
		requestBytes, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		fmt.Println(string(requestBytes))

		By("invoking handler")
		resp := handler.Handle(context.Background(), request)

		By("Should allow")
		Expect(resp.Allowed).To(BeTrue())

		By("Should inject sidecar")
		Expect(resp.Patches).To(HaveLen(2))
		for _, patch := range resp.Patches {
			if patch.Path == "/metadata/annotations/sidecar.example~1inject-name" {
				Expect(patch.Operation).To(BeEquivalentTo("add"))
				Expect(patch.Value).To(BeEquivalentTo("sidecar"))
			} else if patch.Path == "/spec/containers/1" {
				Expect(patch.Operation).To(BeEquivalentTo("add"))
			}
		}
	})

})

func Test_sidecarMutationHandler_Handle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test sidecar mutation handler")
}
