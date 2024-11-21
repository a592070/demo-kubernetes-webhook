package server

import (
	"context"
	"crypto/tls"
	"demo-kubernetes-webhook/pkg/internal/sidecar_mutation/handlers"
	"fmt"
	"github.com/alron/ginlogr"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"os/signal"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"syscall"
	"time"
)

type httpServer struct {
	logger logr.Logger
	engine *gin.Engine
	port   int

	tlsPort        int
	tlsEnable      bool
	tlsCertificate tls.Certificate

	sidecarMutationHandler handlers.AdmissionHandler
}

func NewHttpServer(
	logger logr.Logger,
	port int,
	tlsEnable bool,
	tlsPort int,
	tlsCertFile string,
	tlsKeyFile string,
	sidecarMutationHandler handlers.AdmissionHandler) (Server, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(ginlogr.Ginlogr(logger, time.RFC3339, true))
	engine.Use(gin.Recovery())

	var certificate tls.Certificate
	if tlsEnable {
		logger.Info("tls enabled")
		certBytes, err := os.ReadFile(tlsCertFile)
		if err != nil {
			return nil, errors.Wrap(err, "[NewHttpServer]failed to read tls cert file")
		}
		keyBytes, err := os.ReadFile(tlsKeyFile)
		if err != nil {
			return nil, errors.Wrap(err, "[NewHttpServer]failed to read tls key file")
		}
		certificate, err = tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, errors.Wrap(err, "[NewHttpServer]failed to load certificate")
		}
	}

	return &httpServer{
		logger: logger,
		engine: engine,
		port:   port,

		tlsEnable:      tlsEnable,
		tlsPort:        tlsPort,
		tlsCertificate: certificate,

		sidecarMutationHandler: sidecarMutationHandler,
	}, nil
}

func (s *httpServer) registerRoutes() {
	s.engine.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	s.engine.POST("/sidecar", gin.WrapH(&admission.Webhook{Handler: s.sidecarMutationHandler}))

}

func (s *httpServer) Run() error {
	s.registerRoutes()
	var server *http.Server
	if s.tlsEnable {
		server = &http.Server{
			Handler: s.engine,
			Addr:    fmt.Sprintf(":%d", s.tlsPort),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{s.tlsCertificate},
			},
		}

		go func(server *http.Server) {
			s.logger.Info(fmt.Sprintf("Https server is listening on port %d", s.tlsPort))
			err := server.ListenAndServeTLS("", "")
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Error(err, "Http server start failed")
			}
		}(server)
	} else {
		server = &http.Server{
			Handler: s.engine,
			Addr:    fmt.Sprintf(":%d", s.port),
		}

		go func(server *http.Server) {
			s.logger.Info(fmt.Sprintf("Http server is listening on port %d", s.port))
			err := server.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Error(err, "Http server start failed")
			}
		}(server)
	}

	ctx := context.Background()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-signals

	ctx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()
	err := server.Shutdown(ctx)
	if err != nil {
		s.logger.Error(err, "Http server shutdown")
	}
	return err
}
