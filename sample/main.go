package main

import (
	"demo-kubernetes-webhook/sample/handlers"
	"demo-kubernetes-webhook/sample/usecase"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd/api"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

func newDecoder(scheme *runtime.Scheme) admission.Decoder {
	return admission.NewDecoder(scheme)
}

func main() {
	scheme, err := newScheme()
	if err != nil {
		panic(err)
	}
	decoder := newDecoder(scheme)

	sidecarMutator := usecase.NewSidecarMutator()
	sidecarMutationHandler := handlers.NewSidecarMutationHandler(decoder, sidecarMutator)

	engine := gin.Default()

	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	engine.Any("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	engine.POST("/sidecar", gin.WrapH(&admission.Webhook{Handler: sidecarMutationHandler}))

	err = engine.Run(":8080")
	if err != nil {
		panic(err)
	}
}
