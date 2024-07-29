package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/gorcon/rcon"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

type Rcon struct {
	Address  string
	Password string
	conn     *rcon.Conn
}

func NewRconCommand() *cobra.Command {
	rcon := &Rcon{}

	cmd := &cobra.Command{
		Use:   "rcon",
		Short: "Minecraft RCON client",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rcon.Run()
		},
	}

	flags := cmd.Flags()

	defaultAddr := ":25575"
	if env.IsDevEnv() {
		defaultAddr = "127.0.0.2:25575"
	}

	flags.StringVarP(&rcon.Address, "address", "a", defaultAddr, "Address of the RCON server")
	flags.StringVarP(&rcon.Password, "password", "p", "x", "Password for the RCON server")

	return cmd
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

func isatty() bool {
	_, err := unix.IoctlGetTermios(syscall.Stdin, unix.TCGETS)
	return err == nil
}

func (r *Rcon) Run() error {
	if err := r.connect(r.Address, r.Password); err != nil {
		return err
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
			return err
		}

		fmt.Println(output)
	}

	return nil
}
