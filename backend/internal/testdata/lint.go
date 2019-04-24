package testdata

import (
	"context"
	"time"

	"github.com/bsm/accord"
	"github.com/bsm/accord/backend"
	"github.com/bsm/accord/rpc"
	"github.com/google/uuid"
	G "github.com/onsi/ginkgo"
	Ω "github.com/onsi/gomega"
)

// BehavesLikeBackendData contains a subject
type BehavesLikeBackendData struct {
	Subject backend.Backend
}

// BehavesLikeBackend shared block
func BehavesLikeBackend(data *BehavesLikeBackendData) func() {
	var ctx = context.Background()
	const (
		owner1    = "THEOWNER"
		owner2    = "OTHERONE"
		namespace = "name:space"
		name      = "my.resource"
		minute    = time.Minute
	)

	return func() {
		var subject backend.Backend
		var now time.Time

		G.BeforeEach(func() {
			subject = data.Subject
			now = time.Now()
		})

		G.AfterEach(func() {
			Ω.Expect(subject.Close()).To(Ω.Succeed())
		})

		G.It("should acquire", func() {
			h, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), map[string]string{"k": "v"})
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(h.ID.String()).To(Ω.HaveLen(36))
			Ω.Expect(h.Namespace).To(Ω.Equal(namespace))
			Ω.Expect(h.Name).To(Ω.Equal(name))
			Ω.Expect(h.Owner).To(Ω.Equal(owner1))
			Ω.Expect(h.ExpTime).To(Ω.BeTemporally("~", now.Add(minute), time.Second))
			Ω.Expect(h.DoneTime).To(Ω.BeZero())
			Ω.Expect(h.NumAcquired).To(Ω.Equal(1))
			Ω.Expect(h.Metadata).To(Ω.Equal(map[string]string{"k": "v"}))
		})

		G.It("should acquire (once)", func() {
			// try to acquire 2x
			_, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), nil)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			_, err = subject.Acquire(ctx, owner1, namespace, name, now.Add(2*minute), nil)
			Ω.Expect(err).To(Ω.Equal(accord.ErrAcquired))

			// try to acquire as someone else
			_, err = subject.Acquire(ctx, owner2, namespace, name, now.Add(2*minute), nil)
			Ω.Expect(err).To(Ω.Equal(accord.ErrAcquired))
		})

		G.It("should not allow acquire when done", func() {
			h, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), nil)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(subject.Done(ctx, owner1, h.ID, nil)).To(Ω.Succeed())

			_, err = subject.Acquire(ctx, owner1, namespace, name, now.Add(2*minute), nil)
			Ω.Expect(err).To(Ω.Equal(accord.ErrDone))
		})

		G.It("should allow (re-)acquire when expired", func() {
			h1, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), nil)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(subject.Renew(ctx, owner1, h1.ID, now.Add(-time.Second), nil)).To(Ω.Succeed())

			h2, err := subject.Acquire(ctx, owner2, namespace, name, now.Add(minute), nil)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(h2.ID).NotTo(Ω.Equal(h1.ID))
			Ω.Expect(h2.Owner).To(Ω.Equal(owner2))
		})

		G.It("should allow to renew", func() {
			h1, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), map[string]string{"k": "v"})
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(subject.Renew(ctx, owner1, h1.ID, now.Add(2*minute), map[string]string{"l": "w"})).To(Ω.Succeed())

			h2, err := subject.Get(ctx, h1.ID)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(h2.ExpTime).To(Ω.BeTemporally("~", now.Add(2*minute), time.Second))
			Ω.Expect(h2.Metadata).To(Ω.Equal(map[string]string{"k": "v", "l": "w"}))
		})

		G.It("should not allow renew when done", func() {
			h, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), nil)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(subject.Done(ctx, owner1, h.ID, nil)).To(Ω.Succeed())
			Ω.Expect(subject.Renew(ctx, owner1, h.ID, now.Add(2*minute), nil)).To(Ω.Equal(backend.ErrInvalidHandle))
		})

		G.It("should not allow renew when owned by someone else", func() {
			h, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), nil)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())

			// try to acquire from a 2nd process
			Ω.Expect(subject.Renew(ctx, owner2, h.ID, now.Add(2*minute), nil)).To(Ω.Equal(backend.ErrInvalidHandle))
		})

		G.It("should mark as done (once)", func() {
			h1, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), map[string]string{"k": "v"})
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(subject.Done(ctx, owner1, h1.ID, map[string]string{"l": "w"})).To(Ω.Succeed())
			Ω.Expect(subject.Done(ctx, owner1, h1.ID, map[string]string{"m": "x"})).To(Ω.Equal(backend.ErrInvalidHandle))

			_, err = subject.Acquire(ctx, owner1, namespace, name, now.Add(2*minute), nil)
			Ω.Expect(err).To(Ω.Equal(accord.ErrDone))

			h2, err := subject.Get(ctx, h1.ID)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(h2.IsDone()).To(Ω.BeTrue())
			Ω.Expect(h2.Metadata).To(Ω.Equal(map[string]string{"k": "v", "l": "w"}))
		})

		G.It("should get by ID", func() {
			h1, err := subject.Acquire(ctx, owner1, namespace, name, now.Add(minute), nil)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())

			h2, err := subject.Get(ctx, h1.ID)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(h2).To(Ω.Equal(h1))

			h3, err := subject.Get(ctx, uuid.New())
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			Ω.Expect(h3).To(Ω.BeNil())
		})

		G.It("should list", func() {
			var results []*backend.HandleData

			// Acquire 3 resources
			h1, err := subject.Acquire(ctx, owner1, "a/b", "r1", now.Add(minute), nil)
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			h2, err := subject.Acquire(ctx, owner1, "a/b/c", "r2", now.Add(minute), map[string]string{"a": "1"})
			Ω.Expect(err).NotTo(Ω.HaveOccurred())
			h3, err := subject.Acquire(ctx, owner1, "a/x", "r3", now.Add(minute), map[string]string{"a": "1", "b": "2"})
			Ω.Expect(err).NotTo(Ω.HaveOccurred())

			// Mark 2+3 as done
			_ = h1
			Ω.Expect(subject.Done(ctx, owner1, h2.ID, nil)).To(Ω.Succeed())
			Ω.Expect(subject.Done(ctx, owner1, h3.ID, nil)).To(Ω.Succeed())

			// List all
			results = results[:0]
			Ω.Expect(subject.List(ctx, nil, func(h *backend.HandleData) error {
				results = append(results, h)
				return nil
			})).To(Ω.Succeed())
			Ω.Expect(results).To(Ω.HaveLen(3))
			Ω.Expect(results[0].Name).To(Ω.Equal("r3"))
			Ω.Expect(results[1].Name).To(Ω.Equal("r2"))
			Ω.Expect(results[2].Name).To(Ω.Equal("r1"))

			// List done
			results = results[:0]
			Ω.Expect(subject.List(ctx, &rpc.ListRequest{Filter: &rpc.ListRequest_Filter{Status: rpc.ListRequest_Filter_DONE}}, func(h *backend.HandleData) error {
				results = append(results, h)
				return nil
			})).To(Ω.Succeed())
			Ω.Expect(results).To(Ω.HaveLen(2))

			// With namespace
			results = results[:0]
			Ω.Expect(subject.List(ctx, &rpc.ListRequest{Filter: &rpc.ListRequest_Filter{Prefix: "a/b"}}, func(h *backend.HandleData) error {
				results = append(results, h)
				return nil
			})).To(Ω.Succeed())
			Ω.Expect(results).To(Ω.HaveLen(2))

			// With metadata #1
			results = results[:0]
			Ω.Expect(subject.List(ctx, &rpc.ListRequest{Filter: &rpc.ListRequest_Filter{Metadata: map[string]string{"a": "1"}}}, func(h *backend.HandleData) error {
				results = append(results, h)
				return nil
			})).To(Ω.Succeed())
			Ω.Expect(results).To(Ω.HaveLen(2))

			// With metadata #2
			results = results[:0]
			Ω.Expect(subject.List(ctx, &rpc.ListRequest{Filter: &rpc.ListRequest_Filter{Metadata: map[string]string{"b": "2"}}}, func(h *backend.HandleData) error {
				results = append(results, h)
				return nil
			})).To(Ω.Succeed())
			Ω.Expect(results).To(Ω.HaveLen(1))

			// No namespace match
			results = results[:0]
			Ω.Expect(subject.List(ctx, &rpc.ListRequest{Filter: &rpc.ListRequest_Filter{Prefix: "a/x/y"}}, func(h *backend.HandleData) error {
				results = append(results, h)
				return nil
			})).To(Ω.Succeed())
			Ω.Expect(results).To(Ω.BeEmpty())

			// Stop after first
			results = results[:0]
			Ω.Expect(subject.List(ctx, &rpc.ListRequest{Filter: &rpc.ListRequest_Filter{Prefix: "a"}}, func(h *backend.HandleData) error {
				results = append(results, h)
				return backend.ErrIteratorDone
			})).To(Ω.Succeed())
			Ω.Expect(results).To(Ω.HaveLen(1))

			// With offset
			results = results[:0]
			Ω.Expect(subject.List(ctx, &rpc.ListRequest{Offset: 2}, func(h *backend.HandleData) error {
				results = append(results, h)
				return nil
			})).To(Ω.Succeed())
			Ω.Expect(results).To(Ω.HaveLen(1))
			Ω.Expect(results[0].Name).To(Ω.Equal("r1"))
		})
	}
}
