// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/codesphere-cloud/managed-services-lib/client"
	"github.com/codesphere-cloud/managed-services-lib/model"
)

// RegisterRoutes registers CRUD routes for a managed service provider on the given router group.
func RegisterRoutes[CreateParams any, Status any, UpdateParams any](
	group *gin.RouterGroup,
	p Provider[any, Status, UpdateParams],
) {
	// GET / - List all service IDs or get detailed status
	group.GET("", func(c *gin.Context) {
		ids := c.QueryArray("id")
		if len(ids) == 0 {
			// List mode - return all service IDs
			serviceIDs, err := p.List(c.Request.Context())
			if err != nil {
				HandleError(c, err)
				return
			}
			c.JSON(http.StatusOK, serviceIDs)
			return
		}

		// Detail mode - return status for specified IDs
		// Services that don't exist are simply omitted from the result
		modelIDs := make([]model.ServiceID, len(ids))
		for i, id := range ids {
			modelIDs[i] = model.ServiceID(id)
		}
		status, err := p.GetStatus(c.Request.Context(), modelIDs)
		if err != nil {
			HandleError(c, err)
			return
		}
		c.JSON(http.StatusOK, status)
	})

	// POST / - Create a new service
	group.POST("", func(c *gin.Context) {
		params, err := parseCreate[CreateParams](c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := p.Create(c.Request.Context(), params); err != nil {
			HandleError(c, err)
			return
		}
		c.Status(http.StatusCreated)
	})

	// PATCH /:id - Update an existing service
	group.PATCH("/:id", func(c *gin.Context) {
		id, args, err := parseUpdate[UpdateParams](c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := p.Update(c.Request.Context(), id, args); err != nil {
			HandleError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// DELETE /:id - Delete a service
	group.DELETE("/:id", func(c *gin.Context) {
		id := model.ServiceID(c.Param("id"))
		if err := p.Delete(c.Request.Context(), id); err != nil {
			HandleError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// PUT /backups/:id - Take a backup
	group.PUT("/backups/:id", func(c *gin.Context) {
		var args model.TakeBackupArgs
		if err := c.ShouldBindJSON(&args); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		args.ID = c.Param("id")

		if err := p.TakeBackup(c.Request.Context(), args); err != nil {
			HandleError(c, err)
			return
		}
		c.Status(http.StatusAccepted)
	})

	// POST /backups/:id/status - Get backup status
	group.POST("/backups/:id/status", func(c *gin.Context) {
		var retryArgs model.TakeBackupArgs
		if err := c.ShouldBindJSON(&retryArgs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		retryArgs.ID = c.Param("id")

		status, err := p.GetBackupStatus(c.Request.Context(), c.Param("id"), retryArgs)
		if err != nil {
			HandleError(c, err)
			return
		}
		c.JSON(http.StatusOK, status)
	})

	// DELETE /backups/:id - Delete a backup
	group.DELETE("/backups/:id", func(c *gin.Context) {
		var args model.TakeBackupArgs
		if err := c.ShouldBindJSON(&args); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		args.ID = c.Param("id")

		if err := p.DeleteBackup(c.Request.Context(), args); err != nil {
			HandleError(c, err)
			return
		}
		c.Status(http.StatusAccepted)
	})
}

// ParseCreate binds JSON to the Service domain type.
func parseCreate[T any](c *gin.Context) (T, error) {
	var svc T
	if err := c.ShouldBindJSON(&svc); err != nil {
		return svc, err
	}
	return svc, nil
}

// ParseUpdate binds JSON to the UpdateArgs domain type.
func parseUpdate[T any](c *gin.Context) (model.ServiceID, T, error) {
	var args T
	if err := c.ShouldBindJSON(&args); err != nil {
		return "", args, err
	}
	return model.ServiceID(c.Param("id")), args, nil
}

// HandleError handles provider errors and returns appropriate HTTP responses.
// Domain errors (from providers) pass through their message since providers
// craft safe, user-facing text. Infrastructure errors use generic messages
// to avoid leaking Kubernetes internals.
func HandleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrServiceNotFound):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrBackupNotFound):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInvalidArgument):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrServiceNotHealthy):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNamespaceNotFound):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotImplemented):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusNotImplemented, gin.H{"error": err.Error()})
	case errors.Is(err, client.ErrResourceNotFound):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
	case errors.Is(err, client.ErrResourceConflict):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusConflict, gin.H{"error": "resource already exists"})
	case errors.Is(err, client.ErrResourceInvalid):
		slog.Warn("client error", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "resource invalid"})
	case errors.Is(err, client.ErrKubernetesRequestFailed):
		slog.Error("upstream error", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream service error"})
	default:
		slog.Error("internal error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
