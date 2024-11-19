package server

import (
	"context"
	"demo-kubernetes-webhook/pkg/internal/sidecar_mutation/handlers"
	"errors"
	"fmt"
	"github.com/alron/ginlogr"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
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

	sidecarMutationHandler handlers.AdmissionHandler
}

func NewHttpServer(logger logr.Logger, port int, sidecarMutationHandler handlers.AdmissionHandler) (Server, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(ginlogr.Ginlogr(logger, time.RFC3339, true))
	engine.Use(gin.Recovery())

	return &httpServer{
		logger:                 logger,
		engine:                 engine,
		port:                   port,
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
	server := &http.Server{
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
