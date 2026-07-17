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
type Provider[CreateParams any, Status any, UpdateParams any] interface {
	Create(ctx context.Context, params CreateParams) error
	List(ctx context.Context) ([]model.ServiceID, error)
	GetStatus(ctx context.Context, ids []model.ServiceID) (map[model.ServiceID]Status, error)
	Update(ctx context.Context, id model.ServiceID, args UpdateParams) error
	Delete(ctx context.Context, id model.ServiceID) error

	TakeBackup(ctx context.Context, args model.TakeBackupArgs) error
	GetBackupStatus(ctx context.Context, backupID string, retry model.TakeBackupArgs) (BackupStatus, error)
	DeleteBackup(ctx context.Context, args model.TakeBackupArgs) error
}
```

Embed `provider.Base` for the shared dependencies (Kubernetes client, logger, storage class) and helpers. Embed `provider.UnimplementedBackups` if the provider has no backups; its backup endpoints then return `501`.

## Wiring

```go
cfg, _ := config.Load()
k8s, _ := client.NewKubernetesClient(cfg.Kubeconfig)
logger := slog.Default()

routes := map[string]func(*gin.RouterGroup){
	"mysvc": func(g *gin.RouterGroup) {
		provider.RegisterRoutes(g, mysvc.NewProvider(k8s, logger))
	},
}

server, _ := api.NewServer(cfg, routes)
server.Run()
```

`RegisterRoutes` mounts CRUD and backup endpoints under `/api/v1/{name}`.

## Detached Jobs

Some operations (backups, restores, migrations) are easier to run as one-shot Kubernetes Jobs, detached from the provider pod.

- `client.JobRunner` (also on `provider.Base` as `Jobs`) — `Run` / `State` / `Delete` / `Replace` a one-shot Job, with an optional owned credentials Secret injected via `secretKeyRef`.
- `provider.ServiceJob` / `ServiceJobSpec` — build a `JobSpec` with a consistent name (`<operation>-<key>`) and identity labels; `BackupStatusFromJob` / `OperationStatusFromJob` map a Job's state to a status.

```go
spec := provider.ServiceJobSpec(provider.ServiceJob{
	Operation: provider.JobOpBackup, MsID: id, Key: backupID,
	Image: img, Command: []string{"/backup"},
	Env: env, Secrets: secrets, // whatever your image reads
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
