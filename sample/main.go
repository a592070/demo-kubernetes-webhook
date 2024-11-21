package main

import (
	"demo-kubernetes-webhook/sample/handlers"
	"demo-kubernetes-webhook/sample/usecase"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd/api"
	"log"
	"net/http"
	"os"
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

func newRootCommand(commandName string) *cobra.Command {
	var rootCmd = &cobra.Command{
		Short: fmt.Sprintf("%s is the Kubernetes Mutating Webhook", commandName),
		Long:  "It is developed to inject sidecar container to kubernetes' pod by using given value.",
		Use:   commandName,
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			for i := range args {
				log.Printf("Running with arg: %s", args[i])
			}
		},
	}
	helpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		helpFunc(cmd, args)
		os.Exit(0)
	})

	rootCmd.PersistentFlags().Int("port", 8080, "Port")
	rootCmd.PersistentFlags().Bool("tls-enable", false, "Enable Tls")
	rootCmd.PersistentFlags().Int("tls-port", 8443, "Tls Port")
	rootCmd.PersistentFlags().String("tls-cert", "", "Tls Cert Path")
	rootCmd.PersistentFlags().String("tls-key", "", "Tls Key Path")
	return rootCmd
}

func main() {
	rootCmd := newRootCommand("sample-mutating-webhook")
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}

	scheme, err := newScheme()
	if err != nil {
		log.Fatal(err)
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

	if rootCmd.PersistentFlags().Lookup("tls-enable").Value.String() == "true" {
		port := rootCmd.PersistentFlags().Lookup("tls-port").Value.String()
		tlsCert := rootCmd.PersistentFlags().Lookup("tls-cert").Value.String()
		tlsKey := rootCmd.PersistentFlags().Lookup("tls-key").Value.String()

		err = engine.RunTLS(fmt.Sprintf(":%s", port), tlsCert, tlsKey)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		port := rootCmd.PersistentFlags().Lookup("port").Value.String()

		err = engine.Run(fmt.Sprintf(":%s", port))
		if err != nil {
			log.Fatal(err)
		}
	}

}
