package mcserver

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/kofuk/premises/internal/fake/mcserver/logging"
)

//go:embed "server-properties-skeleton.txt"
var defServerProperties []byte

type ServerProperties map[string]string

func LoadServerProperties() (ServerProperties, error) {
	file, err := os.Open("server.properties")
	if err != nil {
		if os.IsNotExist(err) {
			logging.Log("ServerMain", "ERROR", "Failed to load properties from file: server.properties")

			fmt.Println(`java.nio.file.NoSuchFileException: server.properties
	at sun.nio.fs.UnixException.translateToIOException(UnixException.java:92) ~[?:?]
	at sun.nio.fs.UnixException.rethrowAsIOException(UnixException.java:106) ~[?:?]
	at sun.nio.fs.UnixException.rethrowAsIOException(UnixException.java:111) ~[?:?]
	at sun.nio.fs.UnixFileSystemProvider.newByteChannel(UnixFileSystemProvider.java:261) ~[?:?]
	at java.nio.file.Files.newByteChannel(Files.java:379) ~[?:?]
	at java.nio.file.Files.newByteChannel(Files.java:431) ~[?:?]
	at java.nio.file.spi.FileSystemProvider.newInputStream(FileSystemProvider.java:422) ~[?:?]
	at java.nio.file.Files.newInputStream(Files.java:159) ~[?:?]
	at ahi.b(SourceFile:62) ~[server-1.20.1.jar:?]
	at ahf.a(SourceFile:137) ~[server-1.20.1.jar:?]
	at ahg.<init>(SourceFile:12) ~[server-1.20.1.jar:?]
	at net.minecraft.server.Main.main(SourceFile:115) ~[server-1.20.1.jar:?]
	at net.minecraft.bundler.Main.lambda$run$0(Main.java:54) ~[?:?]
	at java.lang.Thread.run(Thread.java:1623) ~[?:?]`)

			if err := os.WriteFile("server.properties", defServerProperties, 0644); err != nil {
				return nil, err
			}

			return LoadServerProperties()
		}

		return nil, err
	}
	defer file.Close()

	r := bufio.NewReader(file)
	result := make(map[string]string)
	for {
		lineBytes, _, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line := strings.TrimSpace(string(lineBytes))
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		field := strings.SplitN(line, "=", 2)
		if len(field) != 2 {
			return nil, errors.New("invalid line in server.properties")
		}

		result[field[0]] = field[1]
	}

	return result, nil
}

func (sp ServerProperties) GetOr(key, defaultValue string) string {
	val, ok := sp[key]
	if !ok {
		return defaultValue
	}
	return val
}

func (sp ServerProperties) GetRconSettings() (bool, string, string, int) {
	enabled := sp.GetOr("enable-rcon", "false")
	if enabled != "true" {
		return false, "", "", 0
	}
	addr := sp.GetOr("server-ip", "0.0.0.0")
	passwd := sp.GetOr("rcon.password", "")
	port := sp.GetOr("rcon.port", "25575")
	portNum, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return true, passwd, addr, 25575
	}
	return true, passwd, addr, int(portNum)
}
