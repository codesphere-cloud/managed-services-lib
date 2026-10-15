// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package model_test

import (
	"strings"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/codesphere-cloud/managed-services-lib/model"
)

var _ = Describe("BackupStore", func() {
	valid := func() model.BackupStore {
		return model.BackupStore{
			Type:        "s3",
			EndpointURL: "https://s3.example.com",
			Bucket:      "backups",
			Prefix:      "team/7",
			AccessKeyID: "key",
		}
	}

	It("accepts a valid store", func() {
		Expect(binding.Validator.ValidateStruct(valid())).To(Succeed())
	})

	DescribeTable("rejects an invalid store",
		func(mutate func(*model.BackupStore)) {
			store := valid()
			mutate(&store)
			Expect(binding.Validator.ValidateStruct(store)).NotTo(Succeed())
		},
		Entry("unsupported type", func(s *model.BackupStore) { s.Type = "gcs" }),
		Entry("endpoint not a URL", func(s *model.BackupStore) { s.EndpointURL = "s3.example.com" }),
		Entry("bucket too long", func(s *model.BackupStore) { s.Bucket = strings.Repeat("a", 256) }),
		Entry("prefix with leading slash", func(s *model.BackupStore) { s.Prefix = "/team" }),
		Entry("prefix with trailing slash", func(s *model.BackupStore) { s.Prefix = "team/" }),
		Entry("missing access key", func(s *model.BackupStore) { s.AccessKeyID = "" }),
	)

	It("rejects secrets without a secret key", func() {
		Expect(binding.Validator.ValidateStruct(model.BackupStoreSecrets{})).NotTo(Succeed())
	})

	DescribeTable("Path",
		func(prefix, want string) {
			store := valid()
			store.Prefix = prefix
			Expect(store.Path()).To(Equal(want))
		},
		Entry("bucket only", "", "backups"),
		Entry("bucket and prefix", "team/7", "backups/team/7"),
		Entry("prefix is not cleaned", "team/../other", "backups/team/../other"),
	)
})
