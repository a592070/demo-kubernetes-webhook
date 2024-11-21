package handlers

import (
	"context"
	"demo-kubernetes-webhook/simple/usecase"
	"encoding/json"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type SidecarMutationHandler struct {
	decoder        admission.Decoder
	sidecarMutator *usecase.SidecarMutator
}

func NewSidecarMutationHandler(decoder admission.Decoder, sidecarMutator *usecase.SidecarMutator) *SidecarMutationHandler {
	return &SidecarMutationHandler{
		decoder:        decoder,
		sidecarMutator: sidecarMutator,
	}
}

func (h *SidecarMutationHandler) Handle(ctx context.Context, request admission.Request) admission.Response {
	pod := corev1.Pod{}
	err := h.decoder.Decode(request, &pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	affectedPod, err := h.sidecarMutator.Apply(ctx, pod)
	if err != nil {
		res := admission.Errored(http.StatusForbidden, err)
		res.Allowed = true
		return res
	}

	marshaledPod, err := json.Marshal(affectedPod)
	if err != nil {
		res := admission.Errored(http.StatusForbidden, errors.Wrap(err, "failed to marshal pod"))
		res.Allowed = true
		return res
	}

	return admission.PatchResponseFromRaw(request.Object.Raw, marshaledPod)
}
