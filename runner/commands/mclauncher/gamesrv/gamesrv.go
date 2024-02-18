package gamesrv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/serverprop"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
)

type ServerInstance struct {
	Name                   string
	FinishWG               sync.WaitGroup
	ShouldStop             bool
	StartupFailed          bool
	IsServerInitialized    bool
	IsGameFinished         bool
	RestartRequested       bool
	Crashed                bool
	lastActive             time.Time
	quickUndoBeforeRestart bool
	ServerPid              int
	Rcon                   *Rcon
}

var (
	activePlayerListRegexp = regexp.MustCompile("^There are ([0-9]+) of a max of [0-9]+ players online")
)

func New() *ServerInstance {
	return &ServerInstance{
		Rcon: NewRcon("127.0.0.1:25575", "x"),
	}
}

func (srv *ServerInstance) isServerActive() bool {
	resp, err := srv.Rcon.Execute("list")
	if err != nil {
		slog.Error("Failed to send list command to server", slog.Any("error", err))
	}

	if match := activePlayerListRegexp.FindStringSubmatch(resp); match != nil {
		if match[1] == "0" {
			slog.Info("Server is detected to be inactive")
			return false
		}
	}

	return true
}

func (srv *ServerInstance) Wait() {
	done := make(chan interface{})

	srv.lastActive = time.Now()
	go func() {
		ticker := time.NewTicker(time.Minute)

		for {
			select {
			case <-ticker.C:
				if srv.isServerActive() {
					srv.lastActive = time.Now()
				} else {
					if srv.lastActive.Add(30 * time.Minute).Before(time.Now()) {
						srv.Stop()
					}
				}
			case <-done:
				goto end
			}
		}

	end:
		ticker.Stop()
	}()

	srv.FinishWG.Wait()
	close(done)
}

func (srv *ServerInstance) Stop() {
	if _, err := srv.Rcon.Execute("stop"); err != nil {
		slog.Error("Failed to send stop command to server", slog.Any("error", err))
	}
}

func (srv *ServerInstance) SaveAll() error {
	if _, err := srv.Rcon.Execute("save-all"); err != nil {
		return err
	}
	return nil
}

func (srv *ServerInstance) AddToWhiteList(players []string) {
	for _, player := range players {
		_, err := srv.Rcon.Execute(fmt.Sprintf("whitelist add %s", player))
		if err != nil {
			slog.Error("Failed to execute whitelist command", slog.Any("error", err))
		}
	}
}

func (srv *ServerInstance) AddToOp(players []string) {
	for _, player := range players {
		_, err := srv.Rcon.Execute(fmt.Sprintf("op %s", player))
		if err != nil {
			slog.Error("Failed to execute op command", slog.Any("error", err))
		}
	}
}

func (srv *ServerInstance) SendChat(message string) error {
	if _, err := srv.Rcon.Execute(fmt.Sprintf("tellraw @a \"%s\"", message)); err != nil {
		return err
	}

	return nil
}

func (srv *ServerInstance) GetSeed() (string, error) {
	seed, err := srv.Rcon.Execute("seed")
	if err != nil {
		return "", err
	}

	if len(seed) < 8 || seed[:7] != "Seed: [" || seed[len(seed)-1] != ']' {
		return "", errors.New("Failed to retrieve seed")
	}

	return seed[7 : len(seed)-1], nil
}

func (srv *ServerInstance) QuickUndo() error {
	srv.quickUndoBeforeRestart = true

	proc, err := os.FindProcess(srv.ServerPid)
	if err != nil {
		return err
	}
	// go go go!!!
	if err := proc.Kill(); err != nil {
		return err
	}

	return nil
}

func LaunchInteractiveRcon() {
	rcon := NewRcon("127.0.0.1:25575", "x")

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		resp, err := rcon.Execute(scanner.Text())
		if err != nil {
			slog.Error("Failed to execute command", slog.Any("error", err))
			os.Exit(1)
		}
		fmt.Println(resp)
		fmt.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		slog.Info("Failed to scan stdin", slog.Any("error", err))
	}
}

var (
	serverLoadingRegexp         = regexp.MustCompile("^Starting ([a-z]+\\.)+Main$")
	serverLoadingProgressRegexp = regexp.MustCompile("\\]: Preparing spawn area: ([0-9]+)%")
	serverLoadedRegexp          = regexp.MustCompile("\\]: Done \\([0-9]*\\.[0-9]*s\\)! For help, type \"help\"")
	serverStoppingRegexp        = regexp.MustCompile("\\]: Stopping the server")
)

func SendStartedEvent(config *runner.Config, srv *ServerInstance) {
	slog.Debug("Send Started event...")

	data := new(runner.StartedExtra)

	data.ServerVersion = config.Server.Version
	data.World.Name = config.World.Name
	seed, err := srv.GetSeed()
	if err != nil {
		slog.Error("Failed to retrieve seed", slog.Any("error", err))
	}
	data.World.Seed = seed

	if err := exterior.SendMessage("serverStatus", runner.Event{
		Type:    runner.EventStarted,
		Started: data,
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}
}

func MonitorServer(config *runner.Config, srv *ServerInstance, stdout io.ReadCloser) error {
	reader := bufio.NewReader(stdout)
	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil && err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if isPrefix {
			continue
		}
		slog.Info("Log from Minecraft", slog.String("content", string(line)))
		if serverLoadingRegexp.Match(line) {
			if err := exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: runner.EventLoading,
				},
			}); err != nil {
				slog.Error("Unable to write send message", slog.Any("error", err))
			}
		} else if serverLoadingProgressRegexp.Match(line) {
			matches := serverLoadingProgressRegexp.FindSubmatch(line)
			if matches == nil {
				continue
			}
			progress, _ := strconv.Atoi(string(matches[1]))
			progress %= 101
			if err := exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: runner.EventLoading,
					Progress:  progress,
				},
			}); err != nil {
				slog.Error("Unable to write send message", slog.Any("error", err))
			}
		} else if serverLoadedRegexp.Match(line) {
			if err := exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: runner.EventRunning,
				},
			}); err != nil {
				slog.Error("Unable to write send message", slog.Any("error", err))
			}

			go SendStartedEvent(config, srv)

			if err := SaveLastServerVersion(config); err != nil {
				slog.Error("Error saving last server versoin", slog.Any("error", err))
			}
		} else if serverStoppingRegexp.Match(line) {
			if err := exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: runner.EventStopping,
				},
			}); err != nil {
				slog.Error("Unable to write send message", slog.Any("error", err))
			}
		}
	}
}

func generateServerProps(config *runner.Config) error {
	serverProps := serverprop.New()
	serverProps.SetMotd(config.Motd)
	serverProps.SetDifficulty(config.World.Difficulty)
	serverProps.SetLevelType(config.World.LevelType)
	serverProps.SetSeed(config.World.Seed)
	serverPropsFile, err := os.Create(fs.LocateWorldData("server.properties"))
	if err != nil {
		return err
	}
	defer serverPropsFile.Close()
	if err := serverProps.Write(serverPropsFile); err != nil {
		return err
	}
	return nil
}

func signEulaForServer() error {
	eulaFile, err := os.Create(fs.LocateWorldData("eula.txt"))
	if err != nil {
		return err
	}
	defer eulaFile.Close()
	eulaFile.WriteString("eula=true")
	return nil
}

func processQuickUndo() error {
	if err := os.RemoveAll(fs.LocateWorldData("world")); err != nil {
		return err
	}

	cmd := exec.Command("cp", "-R", "--", "ss@quick0/world", "ss@quick0/world_nether", "ss@quick0/world_the_end", ".")
	cmd.Dir = fs.LocateWorldData("")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		slog.Info("cp command returned an error (this is no problem in the most cases)", slog.Any("error", err))
	}

	return nil
}

func LaunchServer(config *runner.Config, srv *ServerInstance) error {
	if _, err := os.Stat(fs.LocateServer(config.Server.Version)); err != nil {
		return err
	}

	ver, exists, err := GetLastServerVersion()
	if err != nil {
		return err
	}
	if !exists || ver != config.Server.Version {
		slog.Info("Different version of server selected. cleaning up...", slog.String("old", ver), slog.String("new", config.Server.Version))

		ents, err := os.ReadDir(fs.LocateWorldData(""))
		if err != nil {
			return err
		}
		for _, ent := range ents {
			if ent.Name() == "server.properties" || ent.Name() == "world" || ent.Name() == "world_nether" || ent.Name() == "world_the_end" {
				continue
			}
			if err := os.RemoveAll(fs.LocateWorldData(ent.Name())); err != nil {
				return err
			}
		}
	}

	if err := RemoveLastServerVersion(); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := generateServerProps(config); err != nil {
		return err
	}

	if err := signEulaForServer(); err != nil {
		return err
	}

	srv.Name = config.Server.Version
	srv.FinishWG.Add(1)
	allocSize := config.AllocSize
	if allocSize == 0 {
		allocSize = 512
	}
	javaArgs := []string{fmt.Sprintf("-Xmx%dM", config.AllocSize), fmt.Sprintf("-Xms%dM", config.AllocSize), "-jar", fs.LocateServer(config.Server.Version), "nogui"}
	go func() {
		slog.Info("Launching Minecraft server", slog.String("server_name", config.Server.Version), slog.Any("commandline", javaArgs))
		launchCount := 0
		prevLaunch := time.Now()
		for !srv.ShouldStop && !srv.RestartRequested {
			if srv.quickUndoBeforeRestart {
				if err := processQuickUndo(); err != nil {
					slog.Error("Error processing quick undo", slog.Any("error", err))
				}

				launchCount = 0
				srv.quickUndoBeforeRestart = false
			}

			if launchCount == 5 {
				if time.Now().Sub(prevLaunch) < 3*time.Minute {
					srv.Crashed = true
					break
				}
			}
			cmd := exec.Command("java", javaArgs...)
			cmd.Dir = fs.LocateWorldData("")
			cmdStdout, _ := cmd.StdoutPipe()
			cmd.Stderr = os.Stderr
			cmd.Start()
			srv.ServerPid = cmd.Process.Pid
			MonitorServer(config, srv, cmdStdout)
			cmd.Wait()
			cmdStdout.Close()
			exitCode := cmd.ProcessState.ExitCode()
			slog.Info("Server exited", slog.Int("exit_code", exitCode))
			if exitCode == 0 {
				break
			}
			launchCount++
		}
		srv.FinishWG.Done()
	}()

	return nil
}

func SaveLastServerVersion(config *runner.Config) error {
	file, err := os.Create(fs.LocateDataFile("last_version"))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(config.Server.Version); err != nil {
		return err
	}
	return nil
}

func RemoveLastServerVersion() error {
	if err := os.Remove(fs.LocateDataFile("last_version")); err != nil {
		return err
	}
	return nil
}

func GetLastServerVersion() (string, bool, error) {
	file, err := os.Open(fs.LocateDataFile("last_version"))
	if err != nil && os.IsNotExist(err) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}
