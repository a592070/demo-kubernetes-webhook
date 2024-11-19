package injection

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
)

const (
	name = "mutating-webhook"
)

type config struct {
	Port     int    `json:"port" validate:"required"`
	LogLevel string `json:"logLevel" validate:"required,oneof=debug info warn error"`
}

func NewConfig() (*config, error) {
	var rootCmd = &cobra.Command{
		Short: fmt.Sprintf("%s is the Kubernetes Mutating Webhook", name),
		Long:  "It is developed to inject sidecar container to kubernetes' pod by using given value.",
		Use:   name,
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
	rootCmd.PersistentFlags().String("log", "info", "Log level")

	err := rootCmd.Execute()
	if err != nil {
		return nil, errors.Wrap(err, "[NewConfig]failed to execute cmd")
	}

	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.BindPFlag("Port", rootCmd.PersistentFlags().Lookup("port")); err != nil {
		return nil, errors.Wrap(err, "[NewConfig]failed to bind flag")
	}
	if err := v.BindPFlag("LogLevel", rootCmd.PersistentFlags().Lookup("log")); err != nil {
		return nil, errors.Wrap(err, "[NewConfig]failed to bind flag")
	}

	var cfg config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "[NewConfig]failed to unmarshal config")
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, errors.Wrap(err, "[NewConfig]failed to validate config")
	}

	return &cfg, nil

}
