package accord_test

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	"github.com/bsm/accord"
	"github.com/bsm/accord/backend/direct"
	"github.com/bsm/accord/backend/mock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var backend *mock.Backend
	var subject accord.Client
	var handle *accord.Handle
	var tempDir string
	var metadata = map[string]string{"a": "2", "b": "1"}
	var ctx = context.Background()

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "accord-client-test")
		Expect(err).NotTo(HaveOccurred())

		backend = mock.New()
		subject, err = accord.RPCClient(ctx, direct.Connect(backend), &accord.ClientOptions{
			Dir:       tempDir,
			Owner:     "testclient",
			Namespace: "test",
		})

		handle, err = subject.Acquire(ctx, "resource", metadata)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		handle.Discard()
		Expect(subject.Close()).To(Succeed())
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	It("should acquire", func() {
		Expect(handle.ID()).To(HaveLen(16))
		Expect(handle.Metadata()).To(Equal(metadata))

		stored, err := backend.Get(ctx, handle.ID())
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.ID).To(Equal(handle.ID()))
		Expect(stored.Namespace).To(Equal("test"))
		Expect(stored.Name).To(Equal("resource"))
		Expect(stored.Owner).To(Equal("testclient"))
		Expect(stored.ExpTime).To(BeTemporally("~", time.Now().Add(10*time.Minute), time.Second))
		Expect(stored.IsDone()).To(BeFalse())
		Expect(stored.Metadata).To(Equal(metadata))
	})

	It("should renew", func() {
		stored, err := backend.Get(ctx, handle.ID())
		Expect(err).NotTo(HaveOccurred())
		expTime := stored.ExpTime

		Expect(handle.Renew(ctx, map[string]string{"c": "3"})).To(Succeed())

		stored, err = backend.Get(ctx, handle.ID())
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.ExpTime).To(BeTemporally(">", expTime))
		Expect(stored.Metadata).To(Equal(map[string]string{"a": "2", "b": "1", "c": "3"}))
	})

	It("should discard", func() {
		Expect(handle.Discard()).To(Succeed())

		stored, err := backend.Get(ctx, handle.ID())
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.ExpTime).To(BeTemporally("~", time.Now(), time.Second))
		Expect(stored.IsDone()).To(BeFalse())

		Expect(handle.Discard()).To(Equal(accord.ErrClosed))
		Expect(handle.Renew(ctx, nil)).To(Equal(accord.ErrClosed))
		Expect(handle.Done(ctx, nil)).To(Equal(accord.ErrClosed))
	})

	It("should mark as done", func() {
		Expect(handle.Done(ctx, map[string]string{"c": "3"})).To(Succeed())

		stored, err := backend.Get(ctx, handle.ID())
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.IsDone()).To(BeTrue())
		Expect(stored.Metadata).To(Equal(map[string]string{"a": "2", "b": "1", "c": "3"}))

		Expect(handle.Discard()).To(Equal(accord.ErrClosed))
		Expect(handle.Renew(ctx, nil)).To(Equal(accord.ErrClosed))
		Expect(handle.Done(ctx, nil)).To(Equal(accord.ErrClosed))
	})
})
