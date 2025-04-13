package rcon_test

import (
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/rcon"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("List command", func() {
	DescribeTable("Parse player list",
		func(input string, expected *rcon.ListOutput) {
			players, err := rcon.ParseListOutput(input)
			Expect(err).To(BeNil())
			Expect(players).To(Equal(expected))
		},
		Entry("No player", "There are 0 of a max of 20 players online: ", &rcon.ListOutput{
			MaxPlayers: 20,
			Players:    nil,
		}),
		Entry("One player", "There are 1 of a max of 20 players online: test1", &rcon.ListOutput{
			MaxPlayers: 20,
			Players:    []string{"test1"},
		}),
		Entry("Many player", "There are 2 of a max of 20 players online: test1, test2", &rcon.ListOutput{
			MaxPlayers: 20,
			Players:    []string{"test1", "test2"},
		}),
	)

	DescribeTable("Parse error case",
		func(input string) {
			_, err := rcon.ParseListOutput(input)
			Expect(err).To(HaveOccurred())
		},
		Entry("Mismatched player count", "There are 2 of a max of 20 players online: test1, test2, test3"),
		Entry("Empty input", ""),
	)
})
