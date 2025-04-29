package server

import (
	"testing"

	"github.com/kofuk/premises/controlpanel/internal/conoha"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConoHa", func() {
	var flavors = []conoha.Flavor{
		{ID: "1", RAM: 8192},
		{ID: "2", RAM: 4096},
		{ID: "3", RAM: 1024},
		{ID: "4", RAM: 2048},
	}

	It("should return flavor ID if matching flavor is found", func() {
		id, err := findMatchingFlavor(flavors, 2048)
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("4"))
	})

	It("should raise an error if matching flavor is not found", func() {
		_, err := findMatchingFlavor(flavors, 3000)
		Expect(err).To(HaveOccurred())
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}
