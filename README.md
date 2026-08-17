# managed-services-lib

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://github.com/codesphere-cloud/managed-services-lib/actions/workflows/ci.yml/badge.svg)](https://github.com/codesphere-cloud/managed-services-lib/actions/workflows/ci.yml)
[![Security](https://github.com/codesphere-cloud/managed-services-lib/actions/workflows/security.yml/badge.svg)](https://github.com/codesphere-cloud/managed-services-lib/actions/workflows/security.yml)

Core library for building [Codesphere managed-service provider backends](https://docs.codesphere.com/managed-services/create-custom-rest-backend). Implement one interface; the library serves it over the Codesphere REST contract.

Not a runnable service. Start from [managed-services-template](https://github.com/codesphere-cloud/managed-services-template) for a working server with an example provider, Dockerfile, and CI.

## Install

```bash
go get github.com/codesphere-cloud/managed-services-lib
```

Go 1.26+.

## Provider interface

```go
type Provider[PlanParams, Config, Secrets, Details, UpdateParams any] interface {
	Create(ctx context.Context, id model.ServiceID, teamID int, customSubdomain string,
		plan PlanParams, config Config, secrets Secrets) error
	List(ctx context.Context) ([]model.ServiceID, error)
	GetStatus(ctx context.Context, ids []model.ServiceID) (map[model.ServiceID]ServiceStatus[PlanParams, Config, Details], error)
	Update(ctx context.Context, id model.ServiceID, teamID int, customSubdomain string,
		args UpdateParams) error
	Delete(ctx context.Context, id model.ServiceID) error
}
```

The split between the platform and your provider is visible in the signatures — you never declare
the Codesphere-supplied fields or the contract's envelopes yourself:

| Codesphere provides | You define |
|---|---|
| `id`, `teamId`, `customSubdomain` — passed as arguments | `PlanParams` — contents of `plan.parameters` |
| the `plan: {parameters: …}` wrapper, unwrapped on the way in and re-wrapped on the way out | `Config` — contents of `config` |
| `msId` on backup requests | `Secrets` — contents of `secrets` |
| the `{plan, config, details}` status envelope (`ServiceStatus`) | `Details` — read-only status data (hostnames, ports, readiness) |
| HTTP status codes and error mapping | `UpdateParams` — your partial `PATCH` payload |

`PATCH` bodies are partial, so make `UpdateParams` fields pointers to tell "not sent" from "sent
empty". Alias the instantiation once so the five type parameters stay out of your way:

```go
type MyProvider = provider.Provider[Params, Config, Secrets, Details, UpdateParams]
```

Build status values with `provider.NewServiceStatus(plan, config, details)`. Embed
`provider.Base` for the shared dependencies (Kubernetes client, logger) and helpers.

Backups are an **opt-in capability**, generic over the provider's own backup-store schemas:

```go
type Backups[BackupConfig, BackupSecrets any] interface {
	TakeBackup(ctx context.Context, backupID model.BackupId, msID model.ServiceID,
		config BackupConfig, secrets BackupSecrets) error
	GetBackupStatus(ctx context.Context, backupID model.BackupId, msID model.ServiceID,
		config BackupConfig, secrets BackupSecrets) (BackupStatus, error)
	DeleteBackup(ctx context.Context, backupID model.BackupId, msID model.ServiceID,
		config BackupConfig, secrets BackupSecrets) error
}
```

A provider that supports backups implements `Backups` and calls `RegisterBackupRoutes`.

## Wiring

```go
cfg, _ := config.Load()
k8s, _ := client.NewKubernetesClient(cfg.Kubeconfig)
logger := slog.Default()

routes := map[string]func(*gin.RouterGroup){
	"mysvc": func(g *gin.RouterGroup) {
		p := mysvc.NewProvider(k8s, logger)
		provider.RegisterRoutes(g, p)       // CRUD
		provider.RegisterBackupRoutes(g, p) // backups
	},
}

server, _ := api.NewServer(cfg, routes)
server.Run()
```

`RegisterRoutes` mounts the CRUD endpoints under `/api/v1/{name}`; `RegisterBackupRoutes` adds the `/backups` endpoints for providers that implement `Backups`.

## Detached Jobs

Some operations (backups, restores, migrations) are easier to run as one-shot Kubernetes Jobs, detached from the provider pod.

- `client.JobRunner` (also on `provider.Base` as `Jobs`) — `Run` / `State` / `Delete` / `Replace` a one-shot Job, with an optional owned credentials Secret injected via `secretKeyRef`.
- `provider.ServiceJob` / `ServiceJobSpec` — build a `JobSpec` with a consistent name (`<operation>-<key>`) and identity labels; `BackupStatusFromJob` / `OperationStatusFromJob` map a Job's state to a status.

```go
spec := provider.ServiceJobSpec(provider.ServiceJob{
	Operation: provider.JobOpBackup, MsID: id, Key: backupID,
	Image: img, Command: []string{"/backup"},
	Env: env, Secrets: secrets, // whatever your image reads
	ImagePullSecrets: []string{"regcred"}, // for a private registry
})
err := p.Jobs.Run(ctx, ns, spec)
```

See the package docs and `provider/servicejob_usage_test.go` for details.

## Configuration

`config.Load()` reads these environment variables:

| Variable | Default | |
|----------|---------|--|
| `PORT` | `8080` | HTTP port |
| `API_KEY` | — | auth key (off if unset) |
| `KUBECONFIG` | — | kubeconfig path (in-cluster if unset) |
| `ENVIRONMENT` | `development` | `development` / `production` |

This is framework config only. Provider-specific config (storage class, credentials, image versions) belongs in your provider's constructor.

## Development

`make test`, `make lint`, `make mocks`. `make all` runs everything.
