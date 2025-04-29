package system

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Memory", func() {
	DescribeTable("parseMeminfoLine", func(input, key string, value int, expectsError bool) {
		k, v, err := parseMeminfoLine(input)

		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(k).To(Equal(key))
		Expect(v).To(Equal(value))
	},
		Entry("Line with `kB` suffix", "MemTotal:       16283152 kB", "MemTotal", 16673947648, false),
		Entry("Line without `kB` suffix", "HugePages_Total:       1", "HugePages_Total", 1, false),
		Entry("Line without colon", "foobar", "", 0, true),
		Entry("Empty line", "", "", 0, true),
		Entry("Invalid number", "MemTotal:   foo kB", "MemTotal", 0, true),
		Entry("Suffix except for kB", "MemTotal:   100 mB", "MemTotal", 0, true),
		Entry("Empty value", "MemTotal:", "MemTotal", 0, true),
		Entry("Line with newline", "MemTotal:   10 kB\n", "MemTotal", 10240, false),
		Entry("Line with newline (CRLF)", "MemTotal:   10 kB\r\n", "MemTotal", 10240, false),
	)

	DescribeTable("readTotalMemoryFromReader", func(input string, totalMem int, expectsError bool) {
		mem, err := readTotalMemoryFromReader(strings.NewReader(input))

		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(mem).To(Equal(totalMem))
	},
		Entry("Simple", `MemTotal:       16283152 kB
MemFree:         5143976 kB
MemAvailable:    7294960 kB
Buffers:          297112 kB
`, 16673947648, false),
		Entry("No MemTotal line", `MemFree:         5143976 kB
MemAvailable:    7294960 kB
Buffers:          297112 kB
`, 0, true),
		Entry("Error in non-MemTotal line", `MemFree:         5143976 kB
foobar
MemTotal:       16283152 kB
Buffers:          297112 kB
`, 16673947648, false),
		Entry("Error in MemTotal line", `MemFree:         5143976 kB
MemTotal:       16283152z kB
Buffers:          297112 kB
`, 0, true),
	)
})
