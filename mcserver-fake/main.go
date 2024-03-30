package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

var (
	EulaNotSigned = errors.New("Not signed to eula.txt")
)

func Log(topic, level, message string) {
	time.Sleep((time.Millisecond * time.Duration(rand.Intn(256))) << 2)

	fmt.Printf("[%s] [%s/%s]: %s\n", time.Now().Format(time.TimeOnly), topic, level, message)
}

//go:embed "server-properties-skeleton.txt"
var defServerProperties []byte

type ServerProperties map[string]string

func LoadServerProperties() (ServerProperties, error) {
	file, err := os.Open("server.properties")
	if err != nil {
		if os.IsNotExist(err) {
			Log("ServerMain", "ERROR", "Failed to load properties from file: server.properties")

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
			return nil, errors.New("Invalid line in server.properties")
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

func (sp ServerProperties) GetRconSettings() (bool, string, int) {
	enabled := sp.GetOr("enable-rcon", "false")
	if enabled != "true" {
		return false, "", 0
	}
	passwd := sp.GetOr("rcon.password", "")
	port := sp.GetOr("rcon.port", "25575")
	portNum, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return true, passwd, 25575
	}
	return true, passwd, int(portNum)
}

func SignedToEulaTxt() (bool, error) {
	file, err := os.Open("eula.txt")
	if err != nil {
		if os.IsNotExist(err) {
			Log("ServerMain", "WARN", "Failed to load eula.txt")
			Log("ServerMain", "INFO", "You need to agree to the EULA in order to run the server. Go to eula.txt for more info.")

			if err := os.WriteFile("eula.txt", []byte("eula=false\n"), 0644); err != nil {
				return false, err
			}

			return false, nil
		}

		return false, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(data), "eula=true"), nil
}

type Server struct {
	RconPassword   string
	RconPort       int
	m              sync.Mutex
	WhitelistUsers []string
	OpUsers        []string
}

type RconPacket struct {
	ID      int
	Type    int
	Payload string
}

func ReadRcon(r io.Reader) (*RconPacket, error) {
	var result RconPacket

	buf := make([]byte, 4)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(buf)

	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	result.ID = int(binary.LittleEndian.Uint32(buf))

	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	result.Type = int(binary.LittleEndian.Uint32(buf))

	payloadLength := length - 4 - 4
	buf = make([]byte, payloadLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	result.Payload = strings.TrimRight(string(buf), "\x00")

	return &result, nil
}

func (p RconPacket) WriteToStream(r io.Writer) error {
	length := 4 + 4 + 2 + len([]byte(p.Payload))
	w := bufio.NewWriter(r)
	defer w.Flush()

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(length))
	if _, err := w.Write(buf); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(buf, uint32(p.ID))
	if _, err := w.Write(buf); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(buf, uint32(p.Type))
	if _, err := w.Write(buf); err != nil {
		return err
	}
	if _, err := w.Write([]byte(p.Payload)); err != nil {
		return err
	}
	if _, err := w.Write([]byte{0, 0}); err != nil {
		return err
	}

	return nil
}

func (s *Server) authenticateRcon(p *RconPacket) error {
	if p.Type != 3 {
		return errors.New("authentication required")
	}
	if p.Payload != s.RconPassword {
		return errors.New("authentication failed")
	}
	return nil
}

type CommandHandler map[string]func(cmd []string) (string, error)

func (h CommandHandler) Run(cmd []string) (string, error) {
	if len(cmd) == 0 {
		return "", errors.New("Invalid command")
	}

	handler, ok := h[cmd[0]]
	if !ok {
		return "", errors.New("Command not found")
	}

	return handler(cmd)
}

func (s *Server) commandLoop(conn io.ReadWriter) error {
	finished := false

	handlers := CommandHandler{
		"stop": func(cmd []string) (string, error) {
			finished = true
			return "Stopping the server", nil
		},
		"whitelist": func(cmd []string) (string, error) {
			if len(cmd) != 3 || cmd[1] != "add" {
				return "", errors.New("Invalid argument")
			}
			s.m.Lock()
			defer s.m.Unlock()

			s.WhitelistUsers = append(s.WhitelistUsers, cmd[2])

			return fmt.Sprintf("Added %s to the whitelist", cmd[2]), nil
		},
		"op": func(cmd []string) (string, error) {
			if len(cmd) != 2 {
				return "", errors.New("Not enought argument")
			}
			s.m.Lock()
			defer s.m.Unlock()

			s.OpUsers = append(s.OpUsers, cmd[1])

			return fmt.Sprintf("Made %s a server operator", cmd[1]), nil

		},
		"list": func(cmd []string) (string, error) {
			return "There are 0 of a max of 20 players online:", nil
		},
		"seed": func(cmd []string) (string, error) {
			return "Seed: [2215139433894533904]", nil
		},
		"save-all": func(cmd []string) (string, error) {
			return "Saving the game (this may take a moment!)Saved the game", nil
		},
	}

	packet, err := ReadRcon(conn)
	if err != nil {
		return err
	}
	if err := s.authenticateRcon(packet); err != nil {
		if err := (RconPacket{
			ID:      -1,
			Type:    2,
			Payload: err.Error(),
		}).WriteToStream(conn); err != nil {
			return err
		}
		return err
	}
	packet = &RconPacket{
		ID:   packet.ID,
		Type: 2,
	}
	if err := packet.WriteToStream(conn); err != nil {
		return err
	}

	for !finished {
		packet, err := ReadRcon(conn)
		if err != nil {
			return err
		}

		if packet.Type != 2 {
			return errors.New("Invalid packet type")
		}

		command := strings.Split(packet.Payload, " ")

		if output, err := handlers.Run(command); err != nil {
			if err := (RconPacket{
				ID:      packet.ID,
				Type:    0,
				Payload: fmt.Sprintf("Unknown or incomplete command, see below for error%s<--[HERE]", packet.Payload),
			}).WriteToStream(conn); err != nil {
				return err
			}
		} else {
			if err := (RconPacket{
				ID:      packet.ID,
				Type:    0,
				Payload: output,
			}).WriteToStream(conn); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Server) startRcon() error {
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.RconPort))
	if err != nil {
		return err
	}
	defer l.Close()

	Log("Server thread", "INFO", fmt.Sprintf("RCON running on 0.0.0.0:%d", s.RconPort))

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		Log("RCON Listener #1", "INFO", fmt.Sprintf("Thread RCON Client /%s started", conn.RemoteAddr()))

		if err := s.commandLoop(conn); err != nil {
			if err != io.EOF {
				log.Println(err)

			}
			conn.Close()

			continue
		}
		conn.Close()
		break
	}

	return nil
}

func (s *Server) Run() {
	ctx, cancel := context.WithCancel(context.Background())

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-ctx.Done()

		e.Shutdown(context.TODO())
		wg.Done()
	}()

	go e.Start(":25565")

	if err := s.startRcon(); err != nil {
		log.Fatal(err)
	}

	cancel()

	wg.Wait()
}

func PrintStartServerLog() {
	logs := `Starting mock minecraft server version 0.10.0 (emulating 1.20.1)
Loading properties
Default game type: SURVIVAL
Generating keypair
Starting Minecraft server on *:25565
Using epoll channel type
Preparing level "world"
Preparing start region for dimension minecraft:overworld`
	for _, line := range strings.Split(logs, "\n") {
		Log("Server thread", "INFO", line)
	}
	total := 0

	start := time.Now()

	for total <= 100 {
		Log(fmt.Sprintf("Worker-Main-%d", rand.Intn(4)+1), "INFO", fmt.Sprintf("Preparing spawn area: %d%%", total))

		total += rand.Intn(256) % 20
	}

	elapsed := time.Now().Sub(start)

	Log("Server thread", "INFO", fmt.Sprintf("Time elapsed %d ms", int(elapsed.Milliseconds())))
	Log("Server thread", "INFO", "Done (16.760s)! For help, type \"help\"")
}

func PrintStopServerLog() {
	logs := `Stopping the server
Stopping server
Saving players
Saving worlds
Saving chunks for level 'ServerLevel[world]'/minecraft:overworld
Saving chunks for level 'ServerLevel[world]'/minecraft:the_end
Saving chunks for level 'ServerLevel[world]'/minecraft:the_nether
ThreadedAnvilChunkStorage (world): All chunks are saved
ThreadedAnvilChunkStorage (DIM1): All chunks are saved
ThreadedAnvilChunkStorage (DIM-1): All chunks are saved
Thread RCON Listener stopped
ThreadedAnvilChunkStorage: All dimensions are saved`
	for _, line := range strings.Split(logs, "\n") {
		Log("Server thread", "INFO", line)
	}
}

func main() {
	fmt.Println("Starting net.minecraft.server.Main")

	serverProperties, err := LoadServerProperties()
	if err != nil {
		log.Fatal(err)
	}

	if signed, err := SignedToEulaTxt(); err != nil {
		log.Fatal(err)
	} else if !signed {
		os.Exit(0)
	}

	PrintStartServerLog()

	var s Server

	if enabled, passwd, port := serverProperties.GetRconSettings(); !enabled {
		log.Fatal("You must enable rcon")
	} else {
		s.RconPassword = passwd
		s.RconPort = port
	}

	s.Run()

	PrintStopServerLog()
}
