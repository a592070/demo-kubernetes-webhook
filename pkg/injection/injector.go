package injection

import (
	"demo-kubernetes-webhook/pkg/internal/sidecar_mutation/handlers"
	"demo-kubernetes-webhook/pkg/internal/sidecar_mutation/usecase"
	"demo-kubernetes-webhook/pkg/server"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type DependenciesInjector struct {
	Server server.Server
}

func NewDependenciesInjector() (*DependenciesInjector, error) {
	cfg, err := NewConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not create config")
	}

	logger, err := NewLogger(cfg.LogLevel, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logger")
	}
	log.SetLogger(logger)

	scheme, err := NewScheme()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create scheme")
	}
	decoder := admission.NewDecoder(scheme)

	sidecarMutator := usecase.NewSidecarMutator(logger)

	sidecarMutationHandler := handlers.NewSidecarMutationHandler(logger, decoder, sidecarMutator)

	httpServer, err := server.NewHttpServer(
		logger,
		cfg.Port,
		cfg.Tls.Enable,
		cfg.Tls.Port,
		cfg.Tls.CertFile,
		cfg.Tls.KeyFile,
		sidecarMutationHandler)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http server")
	}

	return &DependenciesInjector{
		Server: httpServer,
	}, nil
}

func NewLogger(level string, skip int) (logr.Logger, error) {
	convertLevel := func(logLevel string) zapcore.Level {
		switch logLevel {
		case "debug":
			return zap.DebugLevel
		case "info":
			return zap.InfoLevel
		case "warn":
			return zap.WarnLevel
		case "error":
			return zap.ErrorLevel
		default:
			return zap.InfoLevel
		}
	}

	c := zap.NewDevelopmentConfig()
	c.Level = zap.NewAtomicLevelAt(convertLevel(level))
	c.EncoderConfig.FunctionKey = "F"

	zapLog, err := c.Build(
		zap.AddCallerSkip(skip), // traverse call depth for more useful log lines
		zap.AddCaller())
	if err != nil {
		return logr.Logger{}, err
	}
	return zapr.NewLogger(zapLog), nil
}

func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := api.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}

func NewKubernetesClient(logger logr.Logger, isDebugMode bool, scheme *runtime.Scheme) (client.Client, error) {
	if isDebugMode {
		logger.Info("Debug mode enabled. Using fake Kubernetes API client")
		return fake.NewFakeClient(), nil
	}
	clientCfg, err := clientConfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "[NewClient]Create NewClient failed")
	}

	return client.New(clientCfg, client.Options{Scheme: scheme})
}
