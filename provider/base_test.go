// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/codesphere-cloud/managed-services-lib/provider"
)

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider Suite")
}

var _ = Describe("Shared helpers", func() {
	Describe("EscapeJSONPointer", func() {
		It("escapes / per RFC 6901", func() {
			Expect(provider.EscapeJSONPointer("managed-services.codesphere.com/replicas")).
				To(Equal("managed-services.codesphere.com~1replicas"))
		})

		It("escapes ~ before /", func() {
			Expect(provider.EscapeJSONPointer("a~b/c")).To(Equal("a~0b~1c"))
		})

		It("leaves plain segments untouched", func() {
			Expect(provider.EscapeJSONPointer("plain")).To(Equal("plain"))
		})
	})

	Describe("ParseSizeMiB", func() {
		It("parses Gi quantities into MiB", func() {
			Expect(provider.ParseSizeMiB("5Gi")).To(Equal(5120))
		})

		It("parses Mi quantities into MiB", func() {
			Expect(provider.ParseSizeMiB("1024Mi")).To(Equal(1024))
		})

		It("returns 0 for an empty string", func() {
			Expect(provider.ParseSizeMiB("")).To(Equal(0))
		})

		It("returns 0 for an unparseable string", func() {
			Expect(provider.ParseSizeMiB("not-a-size")).To(Equal(0))
		})
	})

	Describe("ReplicasFromAnnotation", func() {
		It("returns the stored replica count when present", func() {
			Expect(provider.ReplicasFromAnnotation(map[string]string{"k": "5"}, "k")).To(Equal(5))
		})

		It("falls back to DefaultPausedReplicas when the annotation is missing", func() {
			Expect(provider.ReplicasFromAnnotation(map[string]string{}, "k")).
				To(Equal(provider.DefaultPausedReplicas))
		})

		It("falls back to DefaultPausedReplicas when the annotation is not an integer", func() {
			Expect(provider.ReplicasFromAnnotation(map[string]string{"k": "abc"}, "k")).
				To(Equal(provider.DefaultPausedReplicas))
		})
	})
})
