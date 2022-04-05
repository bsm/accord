package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/bsm/accord/backend/mock"
	"github.com/bsm/accord/internal/service"
	"github.com/bsm/accord/rpc"
	. "github.com/bsm/ginkgo/v2"
	. "github.com/bsm/gomega"
)

var _ = Describe("V1Service", func() {
	var subject *service.Service
	var _ rpc.V1Server = subject
	var backend *mock.Backend
	var ctx = context.Background()
	const owner = "THEOWNER"

	BeforeEach(func() {
		backend = mock.New()
		subject = service.New(backend)
	})

	It("should acquire", func() {
		_, err := subject.Acquire(ctx, &rpc.AcquireRequest{})
		Expect(err).To(MatchError(`rpc error: code = InvalidArgument desc = invalid owner`))
		_, err = subject.Acquire(ctx, &rpc.AcquireRequest{Owner: owner})
		Expect(err).To(MatchError(`rpc error: code = InvalidArgument desc = invalid name`))

		res, err := subject.Acquire(ctx, &rpc.AcquireRequest{
			Owner:     owner,
			Namespace: "ns",
			Name:      "resource",
			Ttl:       60,
			Metadata:  map[string]string{"k": "v"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Status).To(Equal(rpc.Status_OK))
		Expect(res.Handle.Id).To(HaveLen(16))
		Expect(res.Handle.Namespace).To(Equal("ns"))
		Expect(res.Handle.Name).To(Equal("resource"))
		Expect(res.Handle.ExpTime()).To(BeTemporally("~", time.Now().Add(time.Minute), 2*time.Second))
		Expect(res.Handle.NumAcquired).To(Equal(uint32(1)))
		Expect(res.Handle.Metadata).To(Equal(map[string]string{"k": "v"}))
	})

	It("should renew", func() {
		_, err := subject.Renew(ctx, &rpc.RenewRequest{})
		Expect(err).To(MatchError(`rpc error: code = InvalidArgument desc = invalid owner`))
		_, err = subject.Renew(ctx, &rpc.RenewRequest{Owner: owner})
		Expect(err).To(MatchError(`rpc error: code = InvalidArgument desc = invalid handle ID`))

		h, err := backend.Acquire(ctx, owner, "ns", "resource", time.Now(), nil)
		Expect(err).NotTo(HaveOccurred())

		_, err = subject.Renew(ctx, &rpc.RenewRequest{Owner: owner, HandleId: h.ID[:], Ttl: 60})
		Expect(err).NotTo(HaveOccurred())

		h, err = backend.Get(ctx, h.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(h.ExpTime).To(BeTemporally("~", time.Now().Add(time.Minute), time.Second))
	})

	It("should mark done", func() {
		_, err := subject.Done(ctx, &rpc.DoneRequest{})
		Expect(err).To(MatchError(`rpc error: code = InvalidArgument desc = invalid owner`))
		_, err = subject.Done(ctx, &rpc.DoneRequest{Owner: owner})
		Expect(err).To(MatchError(`rpc error: code = InvalidArgument desc = invalid handle ID`))

		h, err := backend.Acquire(ctx, owner, "ns", "resource", time.Now(), nil)
		Expect(err).NotTo(HaveOccurred())

		_, err = subject.Done(ctx, &rpc.DoneRequest{Owner: owner, HandleId: h.ID[:]})
		Expect(err).NotTo(HaveOccurred())

		h, err = backend.Get(ctx, h.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(h.DoneTime).To(BeTemporally("~", time.Now(), time.Second))
	})

	It("should list", func() {
		Expect(backend.Acquire(ctx, owner, "", "res1", time.Now(), nil)).NotTo(BeNil())
		Expect(backend.Acquire(ctx, owner, "", "res2", time.Now(), nil)).NotTo(BeNil())

		h, err := backend.Acquire(ctx, owner, "", "res3", time.Now(), nil)
		Expect(err).NotTo(HaveOccurred())
		_, err = subject.Done(ctx, &rpc.DoneRequest{Owner: owner, HandleId: h.ID[:]})
		Expect(err).NotTo(HaveOccurred())

		mock := &mockListServer{}
		Expect(subject.List(&rpc.ListRequest{}, mock)).To(Succeed())
		Expect(mock.sent).To(HaveLen(3))

		Expect(mock.sent[0].Name).To(Equal("res3"))
		Expect(mock.sent[0].ExpTime()).To(BeTemporally("~", time.Now(), time.Second))
		Expect(mock.sent[0].DoneTime()).To(BeTemporally("~", time.Now(), time.Second))

		Expect(mock.sent[1].Name).To(Equal("res2"))
		Expect(mock.sent[1].ExpTime()).To(BeTemporally("~", time.Now(), time.Second))
		Expect(mock.sent[1].DoneTime()).To(BeZero())
	})
})

// ------------------------------------------------------------------------

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "internal/service")
}

type mockListServer struct {
	rpc.V1_ListServer
	sent []*rpc.Handle
}

func (*mockListServer) Context() context.Context { return context.Background() }
func (s *mockListServer) Send(h *rpc.Handle) error {
	s.sent = append(s.sent, h)
	return nil
}
