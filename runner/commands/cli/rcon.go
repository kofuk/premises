package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/gorcon/rcon"
	"golang.org/x/sys/unix"
)

func isatty() bool {
	_, err := unix.IoctlGetTermios(syscall.Stdin, unix.TCGETS)
	return err == nil
}

type Rcon struct {
	conn *rcon.Conn
}

func (r *Rcon) execute(line string) (string, error) {
	line = strings.TrimPrefix(line, "/")

	output, err := r.conn.Execute(line)
	if err != nil {
		return "", err
	}
	return output, err
}

func (r *Rcon) connect(address, password string) error {
	conn, err := rcon.Dial(address, password)
	if err != nil {
		return err
	}
	r.conn = conn
	return nil
}

func (r *Rcon) Run(args []string) int {
	address := ":25575"
	password := "x"
	if len(args) >= 2 {
		address = args[0]
		password = args[1]
	}

	if err := r.connect(address, password); err != nil {
		fmt.Fprintln(os.Stderr, "Unable to connect to the host:", err.Error())
		return 1
	}
	defer r.conn.Close()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		if isatty() {
			fmt.Print("> ")
		}

		if !scanner.Scan() {
			break
		}

		output, err := r.execute(scanner.Text())
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error executing command:", err.Error())
			return 1
		}

		fmt.Println(output)
	}

	return 0
}
