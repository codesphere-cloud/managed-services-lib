// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/codesphere-cloud/managed-services-lib/model"
)

// UnimplementedBackups provides backup methods that report ErrNotImplemented.
// Embed it in a provider that does not support backups.
type UnimplementedBackups struct{}

// TakeBackup reports that backups are not implemented for this provider.
func (UnimplementedBackups) TakeBackup(_ context.Context, _ model.TakeBackupArgs) error {
	return ErrNotImplemented
}

// GetBackupStatus reports that backups are not implemented for this provider.
func (UnimplementedBackups) GetBackupStatus(_ context.Context, _ string, _ model.TakeBackupArgs) (BackupStatus, error) {
	return BackupStatus{}, ErrNotImplemented
}

// DeleteBackup reports that backups are not implemented for this provider.
func (UnimplementedBackups) DeleteBackup(_ context.Context, _ model.TakeBackupArgs) error {
	return ErrNotImplemented
}
