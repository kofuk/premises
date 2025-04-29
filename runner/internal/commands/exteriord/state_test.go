package exteriord_test

import (
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/exteriord"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type inMemoryBackend struct {
	s map[string]string
}

func (b *inMemoryBackend) LoadStates() (map[string]string, error) {
	return b.s, nil
}

func (b *inMemoryBackend) SaveStates(s map[string]string) error {
	b.s = s
	return nil
}

var _ = Describe("StateStore", func() {
	var (
		backend *inMemoryBackend
		sut     *exteriord.StateStore
	)

	BeforeEach(func() {
		backend = &inMemoryBackend{
			s: make(map[string]string),
		}
		sut = exteriord.NewStateStore(backend)
	})

	It("should store value when Set is called", func() {
		err := sut.Set("foo", "111")
		Expect(err).NotTo(HaveOccurred())
		Expect(backend.s).To(Equal(map[string]string{"foo": "111"}))

		err = sut.Set("foo", "222")
		Expect(err).NotTo(HaveOccurred())
		Expect(backend.s).To(Equal(map[string]string{"foo": "222"}))
	})

	It("should return value when Get is called", func() {
		value, err := sut.Get("foo")
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(""))

		sut.Set("foo", "111")
		value, err = sut.Get("foo")
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal("111"))
	})

	It("should remove value when Remove is called", func() {
		err := sut.Remove("foo")
		Expect(err).NotTo(HaveOccurred())

		sut.Set("foo", "111")
		sut.Set("bar", "222")

		err = sut.Remove("foo")
		Expect(err).NotTo(HaveOccurred())
		Expect(backend.s).To(Equal(map[string]string{"bar": "222"}))
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rcon Suite")
}
