package rcon_test

import (
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/rcon"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Seed command", func() {
	It("should parse seed correctly", func() {
		input := "Seed: [-6288715796049084847]"

		expected := rcon.SeedOutput("-6288715796049084847")
		seed, err := rcon.ParseSeedOutput(input)
		Expect(err).To(BeNil())
		Expect(seed).To(Equal(expected))
	})

	It("should return an error for invalid seed", func() {
		input := ""

		_, err := rcon.ParseSeedOutput(input)
		Expect(err).To(HaveOccurred())
	})
})
