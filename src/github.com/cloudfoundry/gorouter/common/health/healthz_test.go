package health_test

import (
	"code.cloudfoundry.org/gorouter/common/health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthz", func() {
	It("has a Value", func() {
		healthz := &health.Healthz{}
		ok := healthz.Value()
		Expect(ok).To(Equal("ok"))
	})
})
