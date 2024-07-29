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
	"strings"
	"time"

	"github.com/kofuk/premises/internal/fake/mcserver"
	"github.com/kofuk/premises/internal/fake/mcserver/logging"
	"github.com/kofuk/premises/internal/mc/protocol"
)

func SignedToEulaTxt() (bool, error) {
	file, err := os.Open("eula.txt")
	if err != nil {
		if os.IsNotExist(err) {
			logging.Log("ServerMain", "WARN", "Failed to load eula.txt")
			logging.Log("ServerMain", "INFO", "You need to agree to the EULA in order to run the server. Go to eula.txt for more info.")

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
	RconPassword string
	RconAddr     string
	RconPort     int
	State        *mcserver.State
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
		return "", errors.New("invalid command")
	}

	handler, ok := h[cmd[0]]
	if !ok {
		return "", errors.New("command not found")
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
				return "", errors.New("invalid argument")
			}
			s.State.AddToWhitelist(cmd[2])
			return fmt.Sprintf("Added %s to the whitelist", cmd[2]), nil
		},
		"op": func(cmd []string) (string, error) {
			if len(cmd) != 2 {
				return "", errors.New("not enought argument")
			}
			s.State.AddToOp(cmd[1])
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
			return errors.New("invalid packet type")
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
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.RconAddr, s.RconPort))
	if err != nil {
		return err
	}
	defer l.Close()

	logging.Log("Server thread", "INFO", fmt.Sprintf("RCON running on %s:%d", s.RconAddr, s.RconPort))

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		logging.Log("RCON Listener #1", "INFO", fmt.Sprintf("Thread RCON Client /%s started", conn.RemoteAddr()))

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

func (s *Server) createLevelDat() error {
	if _, err := os.Stat("world/level.dat"); err == nil {
		// If there's level.dat already, do nothing.
		return nil
	}

	if err := os.MkdirAll("world", 0755); err != nil {
		return err
	}

	f, err := os.Create("world/level.dat")
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func (s *Server) Run(addr, port string) {
	if err := s.createLevelDat(); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("tcp", addr+":"+port)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := s.startRcon(); err != nil {
			log.Fatal(err)
		}

		cancel()
		l.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			defer conn.Close()

			h := protocol.NewHandler(conn)

			hs, err := h.ReadHandshake()
			if err != nil {
				log.Println(err)
				return
			}

			if hs.NextState != 1 {
				log.Println("This server only supports handshake")
				return
			}

			status := protocol.Status{}
			status.Version.Name = "0.10.0+fake"
			status.Version.Protocol = hs.Version
			status.Players.Max = 0
			status.Players.Online = 0
			status.Description.Text = "Fake Minecraft Server!"
			status.EnforcesSecureChat = true
			status.CustomData = s.State.ToPublicState()

			if err := h.ReadStatusRequest(); err != nil {
				log.Println(err)
				return
			}

			if err := h.WriteStatus(status); err != nil {
				log.Println(err)
				return
			}
			if err := h.HandlePingPong(); err != nil {
				log.Println(err)
				return
			}
		}()
	}
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
		logging.Log("Server thread", "INFO", line)
	}
	total := 0

	start := time.Now()

	for total <= 100 {
		logging.Log(fmt.Sprintf("Worker-Main-%d", rand.Intn(4)+1), "INFO", fmt.Sprintf("Preparing spawn area: %d%%", total))

		total += rand.Intn(256) % 20
	}

	elapsed := time.Since(start)

	logging.Log("Server thread", "INFO", fmt.Sprintf("Time elapsed %d ms", int(elapsed.Milliseconds())))
	logging.Log("Server thread", "INFO", "Done (16.760s)! For help, type \"help\"")
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
		logging.Log("Server thread", "INFO", line)
	}
}

func main() {
	fmt.Println("Starting net.minecraft.server.Main")

	serverProperties, err := mcserver.LoadServerProperties()
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

	if enabled, passwd, addr, port := serverProperties.GetRconSettings(); !enabled {
		log.Fatal("You must enable rcon")
	} else {
		s.RconPassword = passwd
		s.RconAddr = addr
		s.RconPort = port
	}

	state, err := mcserver.CreateState(serverProperties)
	if err != nil {
		log.Fatal(err)
	}
	s.State = state

	s.Run(serverProperties.GetOr("server-ip", "0.0.0.0"), serverProperties.GetOr("server-port", "25565"))

	if err := state.Save(); err != nil {
		log.Fatal(err)
	}

	PrintStopServerLog()
}
