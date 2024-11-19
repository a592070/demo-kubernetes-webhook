package handlers

import "sigs.k8s.io/controller-runtime/pkg/webhook/admission"

type AdmissionHandler interface {
	admission.Handler
}
