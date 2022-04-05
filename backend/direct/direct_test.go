package direct_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/bsm/accord/backend/direct"
	"github.com/bsm/accord/backend/mock"
	"github.com/bsm/accord/rpc"
	. "github.com/bsm/ginkgo/v2"
	. "github.com/bsm/gomega"
	"github.com/google/uuid"
)

var _ = Describe("ServerBypass", func() {
	var subject rpc.V1Client
	var backend *mock.Backend
	var ctx = context.Background()
	const owner = "THEOWNER"

	BeforeEach(func() {
		backend = mock.New()
		subject = direct.Connect(backend)
	})

	It("should proxy RPC calls", func() {
		res, err := subject.Acquire(ctx, &rpc.AcquireRequest{
			Owner: owner,
			Name:  "resource",
			Ttl:   60,
		})
		Expect(err).NotTo(HaveOccurred())

		stored, err := backend.Get(ctx, uuid.Must(uuid.FromBytes(res.Handle.Id)))
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.Name).To(Equal("resource"))
	})

	It("should proxy streaming RPC calls", func() {
		_, err := backend.Acquire(ctx, owner, "", "res1", time.Now(), nil)
		Expect(err).NotTo(HaveOccurred())
		h2, err := backend.Acquire(ctx, owner, "", "res2", time.Now(), nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(backend.Done(ctx, owner, h2.ID, nil)).To(Succeed())
		h3, err := backend.Acquire(ctx, owner, "", "res3", time.Now(), nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(backend.Done(ctx, owner, h3.ID, nil)).To(Succeed())

		iter, err := subject.List(ctx, &rpc.ListRequest{
			Filter: &rpc.ListRequest_Filter{
				Status: rpc.ListRequest_Filter_DONE,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		found := 0
		for {
			_, err := iter.Recv()
			if err == io.EOF {
				break
			}
			Expect(err).NotTo(HaveOccurred())
			found++
		}
		Expect(found).To(Equal(2))
	})
})

// ------------------------------------------------------------------------

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "accord/backend/direct")
}
