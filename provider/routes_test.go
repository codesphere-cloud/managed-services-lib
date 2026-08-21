// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/codesphere-cloud/managed-services-lib/model"
	"github.com/codesphere-cloud/managed-services-lib/provider"
)

type (
	fakeParams struct {
		Storage int `json:"storage"`
	}

	fakeConfig struct {
		Version string `json:"version"`
	}

	fakeSecrets struct {
		Password string `json:"password"`
	}

	fakeDetails struct {
		Hostname string `json:"hostname"`
		Ready    bool   `json:"ready"`
	}

	fakeUpdate struct {
		Plan *struct {
			Parameters fakeParams `json:"parameters"`
		} `json:"plan"`
		Config *fakeConfig `json:"config"`
	}

	fakeBackupConfig struct {
		Bucket string `json:"bucket"`
	}

	fakeBackupSecrets struct {
		AccessKey string `json:"accessKey"`
	}
)

type createCall struct {
	ID              model.ServiceID
	TeamID          int
	CustomSubdomain *string
	Plan            fakeParams
	Config          fakeConfig
	Secrets         fakeSecrets
	RecoverFrom     *model.RecoverFrom
}

type updateCall struct {
	ID              model.ServiceID
	TeamID          int
	CustomSubdomain *string
	Args            fakeUpdate
}

type backupCall struct {
	BackupID      model.BackupId
	MsID          model.ServiceID
	TeamID        int
	Config        fakeBackupConfig
	Secrets       fakeBackupSecrets
	RetentionDays *int
}

type fakeProvider struct {
	created  []createCall
	updated  []updateCall
	deleted  []model.ServiceID
	backedUp []backupCall
	status   map[model.ServiceID]model.ServiceStatus[fakeParams, fakeConfig, fakeDetails]
}

func (f *fakeProvider) Create(_ context.Context, id model.ServiceID, teamID int, customSubdomain *string,
	plan fakeParams, config fakeConfig, secrets fakeSecrets, recoverFrom *model.RecoverFrom) error {
	f.created = append(f.created, createCall{id, teamID, customSubdomain, plan, config, secrets, recoverFrom})
	return nil
}

func (f *fakeProvider) List(_ context.Context) ([]model.ServiceID, error) {
	return []model.ServiceID{"svc-1", "svc-2"}, nil
}

func (f *fakeProvider) GetStatus(_ context.Context, _ []model.ServiceID) (map[model.ServiceID]model.ServiceStatus[fakeParams, fakeConfig, fakeDetails], error) {
	return f.status, nil
}

func (f *fakeProvider) Update(_ context.Context, id model.ServiceID, teamID int, customSubdomain *string,
	args fakeUpdate) error {
	f.updated = append(f.updated, updateCall{id, teamID, customSubdomain, args})
	return nil
}

func (f *fakeProvider) Delete(_ context.Context, id model.ServiceID) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeProvider) TakeBackup(_ context.Context, backupID model.BackupId, msID model.ServiceID, teamID int,
	config fakeBackupConfig, secrets fakeBackupSecrets, retentionDays *int) error {
	f.backedUp = append(f.backedUp, backupCall{backupID, msID, teamID, config, secrets, retentionDays})
	return nil
}

func (f *fakeProvider) GetBackupStatus(_ context.Context, _ model.BackupId, _ model.ServiceID, _ int,
	_ fakeBackupConfig, _ fakeBackupSecrets, _ *int) (model.BackupStatus, error) {
	return model.BackupStatus{Exists: true}, nil
}

func (f *fakeProvider) DeleteBackup(_ context.Context, _ model.BackupId, _ model.ServiceID, _ int,
	_ fakeBackupConfig, _ fakeBackupSecrets, _ *int) error {
	return nil
}

var _ = Describe("Routes", func() {
	var (
		p      *fakeProvider
		router *gin.Engine
	)

	do := func(method, path, body string) *httptest.ResponseRecorder {
		var req *http.Request
		if body == "" {
			req = httptest.NewRequest(method, path, nil)
		} else {
			req = httptest.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		p = &fakeProvider{}
		router = gin.New()
		group := router.Group("/api/v1/fake")
		provider.RegisterRoutes(group, p)
		provider.RegisterBackupRoutes(group, p)
	})

	Describe("POST /", func() {
		const body = `{
			"id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			"teamId": 7,
			"customSubdomain": "my-db",
			"plan": {"parameters": {"storage": 1000}},
			"config": {"version": "14.2"},
			"secrets": {"password": "super-secret-password"}
		}`

		It("passes each service-level field as its own argument", func() {
			w := do(http.MethodPost, "/api/v1/fake", body)

			Expect(w.Code).To(Equal(http.StatusCreated))
			Expect(p.created).To(HaveLen(1))
			Expect(p.created[0].ID).To(Equal(model.ServiceID("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")))
			Expect(p.created[0].TeamID).To(Equal(7))
			Expect(p.created[0].CustomSubdomain).To(HaveValue(Equal("my-db")))
		})

		It("unwraps plan.parameters into the provider's params type", func() {
			do(http.MethodPost, "/api/v1/fake", body)

			Expect(p.created[0].Plan).To(Equal(fakeParams{Storage: 1000}))
		})

		It("decodes the provider's own config and secrets sections", func() {
			do(http.MethodPost, "/api/v1/fake", body)

			Expect(p.created[0].Config).To(Equal(fakeConfig{Version: "14.2"}))
			Expect(p.created[0].Secrets).To(Equal(fakeSecrets{Password: "super-secret-password"}))
		})

		It("rejects a malformed body", func() {
			w := do(http.MethodPost, "/api/v1/fake", `{"plan": "not-an-object"}`)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(p.created).To(BeEmpty())
		})

		It("rejects a body without an id", func() {
			w := do(http.MethodPost, "/api/v1/fake", `{"teamId": 7}`)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(p.created).To(BeEmpty())
		})

		It("rejects a body without a teamId", func() {
			w := do(http.MethodPost, "/api/v1/fake", `{"id": "svc-1"}`)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(p.created).To(BeEmpty())
		})
	})

	Describe("PATCH /:id", func() {
		It("takes the ID from the path and decodes the service fields alongside the partial payload", func() {
			w := do(http.MethodPatch, "/api/v1/fake/svc-1",
				`{"id": "svc-1", "teamId": 7, "customSubdomain": "my-db", "plan": {"parameters": {"storage": 2000}}}`)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(p.updated).To(HaveLen(1))
			Expect(p.updated[0].ID).To(Equal(model.ServiceID("svc-1")))
			Expect(p.updated[0].TeamID).To(Equal(7))
			Expect(p.updated[0].CustomSubdomain).To(HaveValue(Equal("my-db")))
			Expect(p.updated[0].Args.Plan).NotTo(BeNil())
			Expect(p.updated[0].Args.Plan.Parameters).To(Equal(fakeParams{Storage: 2000}))
		})

		It("rejects a body whose id does not match the path", func() {
			w := do(http.MethodPatch, "/api/v1/fake/svc-1", `{"id": "svc-2", "teamId": 7}`)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(p.updated).To(BeEmpty())
		})

		It("leaves sections the request omits unset", func() {
			do(http.MethodPatch, "/api/v1/fake/svc-1",
				`{"id": "svc-1", "teamId": 7, "customSubdomain": "my-db", "plan": {"parameters": {"storage": 2000}}}`)

			Expect(p.updated[0].Args.Config).To(BeNil())
		})
	})

	Describe("GET /", func() {
		It("lists service IDs when no id is given", func() {
			w := do(http.MethodGet, "/api/v1/fake", "")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal(`["svc-1","svc-2"]`))
		})

		It("exposes the wrapped plan parameters to the provider that built it", func() {
			status := provider.NewServiceStatus(
				fakeParams{Storage: 1000},
				fakeConfig{Version: "14.2"},
				fakeDetails{Ready: true},
			)

			Expect(status.Plan.Parameters).To(Equal(fakeParams{Storage: 1000}))
		})

		It("re-wraps plan parameters in the status response", func() {
			p.status = map[model.ServiceID]model.ServiceStatus[fakeParams, fakeConfig, fakeDetails]{
				"svc-1": provider.NewServiceStatus(
					fakeParams{Storage: 1000},
					fakeConfig{Version: "14.2"},
					fakeDetails{Hostname: "10.0.0.5", Ready: true},
				),
			}

			w := do(http.MethodGet, "/api/v1/fake?id=svc-1", "")

			Expect(w.Code).To(Equal(http.StatusOK))
			var got map[string]map[string]json.RawMessage
			Expect(json.Unmarshal(w.Body.Bytes(), &got)).To(Succeed())
			Expect(string(got["svc-1"]["plan"])).To(MatchJSON(`{"parameters":{"storage":1000}}`))
			Expect(string(got["svc-1"]["config"])).To(MatchJSON(`{"version":"14.2"}`))
			Expect(string(got["svc-1"]["details"])).To(MatchJSON(`{"hostname":"10.0.0.5","ready":true}`))
		})
	})

	Describe("DELETE /:id", func() {
		It("passes the path ID through", func() {
			w := do(http.MethodDelete, "/api/v1/fake/svc-1", "")

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(p.deleted).To(Equal([]model.ServiceID{"svc-1"}))
		})
	})

	Describe("PUT /backups/:id", func() {
		It("takes the backup ID from the path and the service ID from msId", func() {
			w := do(http.MethodPut, "/api/v1/fake/backups/backup-1",
				`{"msId": "svc-1", "teamId": 7, "config": {"bucket": "b"}, "secrets": {"accessKey": "k"}}`)

			Expect(w.Code).To(Equal(http.StatusAccepted))
			Expect(p.backedUp).To(Equal([]backupCall{{
				BackupID: "backup-1",
				MsID:     "svc-1",
				TeamID:   7,
				Config:   fakeBackupConfig{Bucket: "b"},
				Secrets:  fakeBackupSecrets{AccessKey: "k"},
			}}))
		})

		It("rejects a body without an msId", func() {
			w := do(http.MethodPut, "/api/v1/fake/backups/backup-1", `{"teamId": 7, "config": {"bucket": "b"}}`)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(p.backedUp).To(BeEmpty())
		})

		It("rejects a body without a teamId", func() {
			w := do(http.MethodPut, "/api/v1/fake/backups/backup-1", `{"msId": "svc-1", "config": {"bucket": "b"}}`)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(p.backedUp).To(BeEmpty())
		})
	})

	Describe("POST /backups/:id/status", func() {
		It("returns the backup status contract", func() {
			w := do(http.MethodPost, "/api/v1/fake/backups/backup-1/status", `{"msId": "svc-1", "teamId": 7}`)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(MatchJSON(`{"exists":true}`))
		})
	})
})
