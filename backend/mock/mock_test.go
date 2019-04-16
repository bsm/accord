package mock_test

import (
	"testing"

	"github.com/bsm/accord/backend/internal/testdata"
	"github.com/bsm/accord/backend/mock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backend", func() {
	var data testdata.BehavesLikeBackendData

	BeforeEach(func() {
		data.Subject = mock.New()
	})

	Context("defaults", testdata.BehavesLikeBackend(&data))
})

// ------------------------------------------------------------------------

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "accord/backend/postgres")
}
