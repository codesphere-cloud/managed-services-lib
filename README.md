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

## Detached jobs (backup & restore)

Backups run **detached** from the provider pod as one-shot Kubernetes Jobs. `provider.Base` exposes a `Jobs` (`client.JobRunner`) helper for dispatching and polling them, and the backup lifecycle maps onto the Job lifecycle closely. The three backup methods collapse to near one-liners:

```go
func (p *MyProvider) backupSpec(args model.TakeBackupArgs) client.JobSpec {
	return provider.ServiceJobSpec(provider.ServiceJob{
		Operation: provider.JobOpBackup, // name: backup-<backupID>, auto-labels BackupIDLabel
		MsID:      args.MsID,
		Key:       args.ID,
		Image:     p.backupImage,
		Command:   []string{"/backup"},
		// Env/Secrets are entirely yours — wire whatever your backup image reads.
		Env: map[string]string{
			"BACKUP_STORE_ENDPOINT_URL":  args.Config.EndpointURL,
			"BACKUP_STORE_ACCESS_KEY_ID": args.Config.AccessKeyID,
		},
		Secrets: map[string]string{"BACKUP_STORE_SECRET_KEY": args.Secrets.SecretKey},
		Labels:  p.labels(args.MsID),
	})
}

func (p *MyProvider) TakeBackup(ctx context.Context, args model.TakeBackupArgs) error {
	return p.Jobs.Run(ctx, p.namespace(args.MsID), p.backupSpec(args))
}

func (p *MyProvider) GetBackupStatus(ctx context.Context, backupID string, retry model.TakeBackupArgs) (provider.BackupStatus, error) {
	st, err := p.Jobs.State(ctx, p.namespace(retry.MsID), provider.BackupJobName(backupID))
	if err != nil {
		return provider.BackupStatus{}, err
	}
	if st.Phase == client.JobFailed {
		_ = p.Jobs.Replace(ctx, p.namespace(retry.MsID), p.backupSpec(retry))
	}
	return provider.BackupStatusFromJob(st), nil
}

func (p *MyProvider) DeleteBackup(ctx context.Context, args model.TakeBackupArgs) error {
	return p.Jobs.Delete(ctx, p.namespace(args.MsID), provider.BackupJobName(args.ID))
}
```

`ServiceJobSpec` owns the Job *conventions*: a deterministic name (`<operation>-<key>`, so `Run` is idempotent and `State`/`Delete` resolve the same Job) and the identity labels (always `ServiceIDLabel`; backup operations also auto-stamp `BackupIDLabel` from the key). `Env` and `Secrets` pass through verbatim — each provider's image defines its own contract. Secrets are stored in the Job's owned Secret and injected via `secretKeyRef`, never appearing in the Job or Pod manifest. `Run` is idempotent on the Job name; `State` returns a phase (`pending`/`running`/`succeeded`/`failed`/`not_found`) and, on failure, folds the failed pod's logs into the reason; `Replace` is the retry path. Jobs default to a 6h deadline, a 3-retry backoff, and a 1h finished-TTL — override via `JobSpec` fields, or use `JobSpec.Customize` for anything not modelled.

### Other detached operations

Every detached operation is the same shape, so they all go through `provider.ServiceJob` — you supply the `Operation` prefix and execution details, and get the naming and identity-label conventions for free:

| Operation | Job name | Auto-stamped label |
|-----------|----------|--------------------|
| `JobOpBackup` | `backup-<backupID>` | `ServiceIDLabel`, `BackupIDLabel` |
| `JobOpDeleteBackup` | `delete-backup-<backupID>` | `ServiceIDLabel`, `BackupIDLabel` |
| `JobOpRestore` | `restore-<msID>` | `ServiceIDLabel` |
| any provider-defined prefix | `<op>-<key>` | `ServiceIDLabel` |

The operation→label policy lives in the lib, so a Job's identity labels always exist without the caller managing them; anything else you put in `Labels` rides along underneath. `ServiceIDLabel` is always stamped. The lib keeps only name helpers (`BackupJobName`, `DeleteBackupJobName`) and the status mappers (`BackupStatusFromJob` for the backup contract type, `OperationStatusFromJob` generically) — the three backup endpoints share these so they resolve the same Job.

### Restore

Restore is **not a standalone operation** in this contract — there is no restore endpoint. It happens **inside `Create`**: the create payload carries an optional `recoverFrom` (`model.RecoverFromBackup` or `model.RecoverFromTimestamp`, which the provider embeds in its own `CreateParams`), and when it is set the provider provisions the service *and* dispatches a detached Job to pull data from the backup store. Restore is keyed on the service ID (`restore-<msID>`) — one restore per service creation — and its progress is surfaced through the service's normal `GetStatus` (e.g. `details.ready = false` until it finishes), not a separate status endpoint.

```go
func (p *MyProvider) Create(ctx context.Context, params MyCreateParams) error {
	// ... provision the service (StatefulSet, Service, Secret, ...) ...

	if params.RecoverFrom != nil {
		spec := provider.ServiceJobSpec(provider.ServiceJob{
			Operation: provider.JobOpRestore,   // name: restore-<msID>
			MsID:      params.ID,
			Key:       params.ID,
			Image:     p.restoreImage,
			Command:   []string{"/restore"},
			// recoverFrom is provider-specific; wire whatever your restore image reads.
			Env:     map[string]string{"SOURCE_BACKUP_ID": params.RecoverFrom.Backup.ID},
			Secrets: map[string]string{"SOURCE_SECRET_KEY": params.RecoverFrom.Backup.Secrets.SecretKey},
			Labels:  p.labels(params.ID),
		})
		if err := p.Jobs.Run(ctx, p.namespace(params.ID), spec); err != nil {
			return err
		}
	}
	return nil
}

// In GetStatus, fold the restore Job's progress into the service status:
//   st, _ := p.Jobs.State(ctx, ns, provider.ServiceJobName(provider.JobOpRestore, id))
//   ready := st.Phase == client.JobSucceeded || st.Phase == client.JobNotFound
```


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
