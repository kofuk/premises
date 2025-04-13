package serverproperties_test

import (
	"strings"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/serverproperties"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServerProperties", func() {
	var (
		sut *serverproperties.ServerPropertiesGenerator
	)

	BeforeEach(func() {
		sut = serverproperties.NewServerPropertiesGenerator()
	})

	It("should be able to set a valid property", func() {
		err := sut.Set("motd", "Hello, world!")
		Expect(err).ToNot(HaveOccurred())

		out := new(strings.Builder)
		err = sut.Write(out)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("motd=Hello, world!"))
	})

	It("should reject blocked properties", func() {
		err := sut.Set("enable-rcon", "false")
		Expect(err).To(HaveOccurred())

		out := new(strings.Builder)
		err = sut.Write(out)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).ToNot(ContainSubstring("enable-rcon=false"))
	})
})
