package handlers

import (
	"context"
	"demo-kubernetes-webhook/pkg/internal/sidecar_mutation/usecase"
	"encoding/json"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type sidecarMutationHandler struct {
	logger         logr.Logger
	decoder        admission.Decoder
	sidecarMutator usecase.Mutator[corev1.Pod]
}

func NewSidecarMutationHandler(logger logr.Logger, decoder admission.Decoder, sidecarMutator usecase.Mutator[corev1.Pod]) AdmissionHandler {
	return &sidecarMutationHandler{
		logger:         logger,
		decoder:        decoder,
		sidecarMutator: sidecarMutator,
	}
}

func (h *sidecarMutationHandler) Handle(ctx context.Context, request admission.Request) admission.Response {
	h.logger.Info("processing request",
		"namespace", request.Namespace,
		"name", request.Name,
		"kind", request.Kind,
		"operation", request.Operation,
		"uid", request.UID,
		"request_kind", request.RequestKind,
		"request_resource", request.RequestResource)

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
