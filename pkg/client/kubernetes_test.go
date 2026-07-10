package client_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/codesphere-cloud/managed-services-lib/pkg/client"
)

var _ = Describe("Kubernetes Client Errors", func() {
	Describe("Error Types", func() {
		It("should have distinct error types", func() {
			Expect(client.ErrResourceNotFound).NotTo(BeNil())
			Expect(client.ErrResourceConflict).NotTo(BeNil())
			Expect(client.ErrResourceInvalid).NotTo(BeNil())
			Expect(client.ErrKubernetesRequestFailed).NotTo(BeNil())
		})

		It("should allow error wrapping with Is", func() {
			wrappedErr := errors.Join(client.ErrResourceNotFound, errors.New("additional context"))
			Expect(errors.Is(wrappedErr, client.ErrResourceNotFound)).To(BeTrue())
		})
	})
})
