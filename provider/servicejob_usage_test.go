// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/codesphere-cloud/managed-services-lib/client"
	"github.com/codesphere-cloud/managed-services-lib/client/mocks"
	"github.com/codesphere-cloud/managed-services-lib/model"
	"github.com/codesphere-cloud/managed-services-lib/provider"
)

func runningJob(name, namespace string) *unstructured.Unstructured {
	start := metav1.Now()
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&batchv1.Job{
		TypeMeta:   metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Status:     batchv1.JobStatus{Active: 1, StartTime: &start},
	})
	Expect(err).NotTo(HaveOccurred())
	return &unstructured.Unstructured{Object: m}
}

// This suite is a worked example of how a provider dispatches detached work:
// build the spec with ServiceJob (which applies the naming + label conventions),
// then Run/State/Delete it through Base.Jobs (a client.JobRunner). It covers the
// three operations a provider actually runs as Jobs — take backup, delete
// backup, and restore-on-create — plus polling and cleanup.
var _ = Describe("Dispatching operations with ServiceJob + JobRunner", func() {
	const namespace = "rg-7" // provider.NamespaceForTeam(7)

	var (
		ctx     context.Context
		kube    *mocks.MockKubernetesClient
		runner  client.JobRunner
		created *unstructured.Unstructured
	)

	BeforeEach(func() {
		ctx = context.Background()
		kube = new(mocks.MockKubernetesClient)
		runner = client.NewJobRunner(kube)
		created = nil
	})

	// stubDispatch stubs the create path (owned secret -> job -> adopt) and
	// captures the Job object that reaches the cluster.
	stubDispatch := func() {
		kube.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).Return(&corev1.Secret{}, nil).Maybe()
		kube.On("Patch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		withUID := &unstructured.Unstructured{Object: map[string]any{}}
		withUID.SetUID("job-uid") // returned to Run for secret adoption
		kube.On("Create", mock.Anything, model.JobAPI, mock.Anything, mock.Anything).
			Run(func(a mock.Arguments) { created = a.Get(3).(*unstructured.Unstructured) }).
			Return(withUID, nil)
	}

	It("TakeBackup: dispatches backup-<backupID>, labelled by service and backup", func() {
		stubDispatch()

		spec := provider.ServiceJobSpec(provider.ServiceJob{
			Operation: provider.JobOpBackup,
			MsID:      "svc-42",
			Key:       "bkp-7", // the backup ID
			Image:     "backup-image:1",
			Command:   []string{"/backup"},
			Env:       map[string]string{"BACKUP_STORE_ENDPOINT_URL": "https://s3.example.com"},
			Secrets:   map[string]string{"BACKUP_STORE_SECRET_KEY": "shh"},
			Labels:    map[string]string{model.TeamIDLabel: "7"},
		})
		Expect(runner.Run(ctx, namespace, spec)).To(Succeed())

		Expect(created.GetName()).To(Equal("backup-bkp-7"))
		Expect(created.GetLabels()).To(HaveKeyWithValue(model.ServiceIDLabel, "svc-42"))
		Expect(created.GetLabels()).To(HaveKeyWithValue(model.BackupIDLabel, "bkp-7"))
		Expect(created.GetLabels()).To(HaveKeyWithValue(model.TeamIDLabel, "7"))
	})

	It("DeleteBackup: dispatches delete-backup-<backupID> with the same backup identity", func() {
		stubDispatch()

		spec := provider.ServiceJobSpec(provider.ServiceJob{
			Operation: provider.JobOpDeleteBackup,
			MsID:      "svc-42",
			Key:       "bkp-7",
			Image:     "backup-image:1",
			Command:   []string{"/delete-backup"},
			Secrets:   map[string]string{"BACKUP_STORE_SECRET_KEY": "shh"},
		})
		Expect(runner.Run(ctx, namespace, spec)).To(Succeed())

		Expect(created.GetName()).To(Equal("delete-backup-bkp-7"))
		Expect(created.GetName()).NotTo(Equal(provider.BackupJobName("bkp-7"))) // distinct from take-backup
		Expect(created.GetLabels()).To(HaveKeyWithValue(model.ServiceIDLabel, "svc-42"))
		Expect(created.GetLabels()).To(HaveKeyWithValue(model.BackupIDLabel, "bkp-7"))
	})

	It("Restore-on-create: dispatches restore-<msID>, keyed on the service", func() {
		stubDispatch()

		// A provider's Create(params) builds this when params.recoverFrom is set.
		spec := provider.ServiceJobSpec(provider.ServiceJob{
			Operation: provider.JobOpRestore,
			MsID:      "svc-42",
			Key:       "svc-42", // whole-service op: keyed on the service, not a backup
			Image:     "restore-image:1",
			Command:   []string{"/restore"},
			Env:       map[string]string{"SOURCE_BACKUP_ID": "bkp-7"},
			Secrets:   map[string]string{"SOURCE_SECRET_KEY": "shh"},
		})
		Expect(runner.Run(ctx, namespace, spec)).To(Succeed())

		Expect(created.GetName()).To(Equal("restore-svc-42"))
		Expect(created.GetLabels()).To(HaveKeyWithValue(model.ServiceIDLabel, "svc-42"))
		// Restore is not backup-scoped, so it carries no backup-id label.
		Expect(created.GetLabels()).NotTo(HaveKey(model.BackupIDLabel))
	})

	It("GetBackupStatus: polls the backup job and maps it to the contract status", func() {
		kube.On("Get", mock.Anything, mock.MatchedBy(func(r model.ResourceRef) bool {
			return r.APIResource == model.JobAPI && r.Name == "backup-bkp-7"
		})).Return(runningJob("backup-bkp-7", namespace), nil)

		st, err := runner.State(ctx, namespace, provider.BackupJobName("bkp-7"))
		Expect(err).NotTo(HaveOccurred())
		Expect(provider.BackupStatusFromJob(st).Phase).To(Equal(provider.BackupPhaseRunning))
	})

	It("DeleteBackup cleanup: removes the backup job and its owned secret", func() {
		kube.On("Delete", mock.Anything, mock.MatchedBy(func(r model.ResourceRef) bool {
			return r.APIResource == model.JobAPI && r.Name == "backup-bkp-7"
		})).Return(nil)
		kube.On("DeleteSecret", mock.Anything, namespace, "backup-bkp-7").Return(nil)

		Expect(runner.Delete(ctx, namespace, provider.BackupJobName("bkp-7"))).To(Succeed())
	})
})
