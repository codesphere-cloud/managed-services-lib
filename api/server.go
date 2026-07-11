// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/codesphere-cloud/managed-services-lib/config"
	"github.com/codesphere-cloud/managed-services-lib/middleware"
)

const apiVersion = "v1"

// Server represents the HTTP server.
type Server struct {
	cfg            *config.Config
	router         *gin.Engine
	httpServer     *http.Server
	logger         *slog.Logger
	providerRoutes map[string]func(*gin.RouterGroup)
}

// NewServer creates a new HTTP server. Each entry in providerRoutes mounts one
// provider's routes on the group at its key (e.g. "opensearch" -> /api/v1/opensearch).
func NewServer(cfg *config.Config, providerRoutes map[string]func(*gin.RouterGroup)) (*Server, error) {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	logger := slog.Default()

	router := gin.New()

	// Middleware
	router.Use(middleware.ErrorHandler(logger))
	router.Use(middleware.Logger(logger))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS())

	// Auth middleware if API key is set
	if cfg.APIKey != "" {
		router.Use(middleware.Auth(cfg.APIKey))
	}

	server := &Server{
		cfg:            cfg,
		router:         router,
		logger:         logger,
		providerRoutes: providerRoutes,
	}

	if err := server.setupRoutes(); err != nil {
		return nil, err
	}

	return server, nil
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() error {
	// Health check
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/ready", s.readyHandler)

	// API routes
	api := s.router.Group(fmt.Sprintf("/api/%s", apiVersion))
	{
		// Root endpoint
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"version": apiVersion,
				"message": "Managed Services API",
			})
		})

		for path, register := range s.providerRoutes {
			register(api.Group("/" + path))
		}
	}

	// 404 handler
	s.router.NoRoute(middleware.NotFound())
	return nil
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (s *Server) readyHandler(c *gin.Context) {
	// TODO: Add readiness checks (e.g., Kubernetes connectivity)
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
