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

// serviceFields are the service-level fields the contract defines on create and
// update payloads, as opposed to the provider's own sections. On update the ID
// comes from the path instead.
type serviceFields struct {
	ID              model.ServiceID `json:"id"`
	TeamID          int             `json:"teamId"`
	CustomSubdomain *string         `json:"customSubdomain"`
}

// createBody is the create payload: the service fields plus the provider's own
// sections.
type createBody[PlanParams, Config, Secrets any] struct {
	serviceFields
	Plan        model.PlanSpec[PlanParams] `json:"plan"`
	Config      Config                     `json:"config"`
	Secrets     Secrets                    `json:"secrets"`
	RecoverFrom *model.RecoverFrom         `json:"recoverFrom,omitempty"`
}

// backupBody is the backup payload; the backup ID comes from the path.
type backupBody[Config, Secrets any] struct {
	MsID          model.ServiceID `json:"msId"`
	TeamID        int             `json:"teamId"`
	Config        Config          `json:"config"`
	Secrets       Secrets         `json:"secrets"`
	RetentionDays *int            `json:"retentionDays,omitempty"`
}

// RegisterRoutes registers CRUD routes for a managed service provider on the given router group.
func RegisterRoutes[PlanParams, Config, Secrets, Details, UpdateParams any](
	group *gin.RouterGroup,
	p Provider[PlanParams, Config, Secrets, Details, UpdateParams],
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
		var body createBody[PlanParams, Config, Secrets]
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if body.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}
		if body.TeamID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "teamId must be a positive integer"})
			return
		}

		if err := p.Create(c.Request.Context(), body.ID, body.TeamID, body.CustomSubdomain,
			body.Plan.Parameters, body.Config, body.Secrets, body.RecoverFrom); err != nil {
			HandleError(c, err)
			return
		}
		c.Status(http.StatusCreated)
	})

	// PATCH /:id - Update an existing service
	group.PATCH("/:id", func(c *gin.Context) {
		var fields serviceFields
		if err := c.ShouldBindBodyWithJSON(&fields); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var args UpdateParams
		if err := c.ShouldBindBodyWithJSON(&args); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id := model.ServiceID(c.Param("id"))
		if fields.ID != id {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id in body must match path"})
			return
		}
		if err := p.Update(c.Request.Context(), id, fields.TeamID, fields.CustomSubdomain,
			args); err != nil {
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
}

// RegisterBackupRoutes mounts backup endpoints.
func RegisterBackupRoutes[BackupConfig, BackupSecrets any](
	group *gin.RouterGroup,
	b Backups[BackupConfig, BackupSecrets],
) {
	// PUT /backups/:id - Take a backup
	group.PUT("/backups/:id", func(c *gin.Context) {
		backupID, body, err := parseBackup[BackupConfig, BackupSecrets](c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := b.TakeBackup(c.Request.Context(), backupID, body.MsID, body.TeamID,
			body.Config, body.Secrets, body.RetentionDays); err != nil {
			HandleError(c, err)
			return
		}
		c.Status(http.StatusAccepted)
	})

	// POST /backups/:id/status - Get backup status
	group.POST("/backups/:id/status", func(c *gin.Context) {
		backupID, body, err := parseBackup[BackupConfig, BackupSecrets](c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		status, err := b.GetBackupStatus(c.Request.Context(), backupID, body.MsID, body.TeamID,
			body.Config, body.Secrets, body.RetentionDays)
		if err != nil {
			HandleError(c, err)
			return
		}
		c.JSON(http.StatusOK, status)
	})

	// DELETE /backups/:id - Delete a backup
	group.DELETE("/backups/:id", func(c *gin.Context) {
		backupID, body, err := parseBackup[BackupConfig, BackupSecrets](c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := b.DeleteBackup(c.Request.Context(), backupID, body.MsID, body.TeamID,
			body.Config, body.Secrets, body.RetentionDays); err != nil {
			HandleError(c, err)
			return
		}
		c.Status(http.StatusAccepted)
	})
}

func parseBackup[Config, Secrets any](c *gin.Context) (model.BackupId, backupBody[Config, Secrets], error) {
	var body backupBody[Config, Secrets]
	if err := c.ShouldBindJSON(&body); err != nil {
		return "", body, err
	}
	if body.MsID == "" {
		return "", body, errors.New("msId is required")
	}
	if body.TeamID <= 0 {
		return "", body, errors.New("teamId must be a positive integer")
	}
	return model.BackupId(c.Param("id")), body, nil
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
