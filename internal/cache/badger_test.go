package cache_test

import (
	"io/ioutil"
	"os"

	"github.com/bsm/accord/internal/cache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Badger", func() {
	var subject cache.Cache
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "accord-cache-test")
		Expect(err).NotTo(HaveOccurred())

		subject, err = cache.OpenBadger(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(subject.Close()).To(Succeed())
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	It("should add / contain", func() {
		Expect(subject.Contains("x")).To(BeFalse())
		Expect(subject.Add("x")).To(Succeed())
		Expect(subject.Contains("x")).To(BeTrue())
	})

	It("should add in bulk", func() {
		bw, err := subject.AddBatch()
		Expect(err).NotTo(HaveOccurred())
		defer bw.Discard()

		Expect(bw.Add("x")).To(Succeed())
		Expect(bw.Add("y")).To(Succeed())
		Expect(bw.Add("z")).To(Succeed())

		Expect(subject.Contains("x")).To(BeFalse())
		Expect(subject.Contains("y")).To(BeFalse())
		Expect(subject.Contains("z")).To(BeFalse())
		Expect(bw.Flush()).To(Succeed())
		Expect(subject.Contains("x")).To(BeTrue())
		Expect(subject.Contains("y")).To(BeTrue())
		Expect(subject.Contains("z")).To(BeTrue())
	})
})
