package gamesrv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/serverprop"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
	"github.com/kofuk/premises/runner/systemutil"
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
	quickUndoSlot          int
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
	exterior.DispatchMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventStopping,
		},
	})

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

func (srv *ServerInstance) QuickUndo(slot int) error {
	exterior.DispatchMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventStopping,
		},
	})

	srv.quickUndoBeforeRestart = true
	srv.quickUndoSlot = slot

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

func LaunchInteractiveRcon(args []string) int {
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

	return 0
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

	exterior.SendMessage("serverStatus", runner.Event{
		Type:    runner.EventStarted,
		Started: data,
	})
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
			exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: entity.EventLoading,
				},
			})
		} else if serverLoadingProgressRegexp.Match(line) {
			matches := serverLoadingProgressRegexp.FindSubmatch(line)
			if matches == nil {
				continue
			}
			progress, _ := strconv.Atoi(string(matches[1]))
			progress %= 101
			exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: entity.EventLoading,
					Progress:  progress,
				},
			})
		} else if serverLoadedRegexp.Match(line) {
			exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: entity.EventRunning,
				},
			})

			go SendStartedEvent(config, srv)

			if err := SaveLastServerVersion(config); err != nil {
				slog.Error("Error saving last server versoin", slog.Any("error", err))
			}
		} else if serverStoppingRegexp.Match(line) {
			exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: entity.EventStopping,
				},
			})
		}
	}
}

func generateServerProps(config *runner.Config) error {
	serverProps := serverprop.New()
	serverProps.SetMotd(config.Motd)
	serverProps.SetDifficulty(config.World.Difficulty)
	serverProps.SetLevelType(config.World.LevelType)
	serverProps.SetSeed(config.World.Seed)
	serverProps.OverrideProperties(config.Server.ServerPropOverride)
	serverPropsFile, err := os.Create(fs.DataPath("gamedata/server.properties"))
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
	eulaFile, err := os.Create(fs.DataPath("gamedata/eula.txt"))
	if err != nil {
		return err
	}
	defer eulaFile.Close()
	eulaFile.WriteString("eula=true")
	return nil
}

func processQuickUndo(slot int) error {
	if err := os.RemoveAll(fs.DataPath("gamedata/world")); err != nil {
		return err
	}

	cmd := exec.Command("cp", "-R", "--", fmt.Sprintf("ss@quick%d/world", slot), ".")
	cmd.Dir = fs.DataPath("gamedata")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		slog.Info("cp command returned an error", slog.Any("error", err))
	}

	return nil
}

func getJavaPathFromInstalledVersion(version int) (string, error) {
	output, err := systemutil.CmdOutput("update-alternatives", []string{"--list", "java"})
	if err != nil {
		return "", err
	}

	candidates := strings.Split(strings.TrimRight(output, "\r\n"), "\n")
	slog.Debug("Installed java versions", slog.Any("versions", candidates))

	for _, path := range candidates {
		if strings.Index(path, fmt.Sprintf("-%d-", version)) >= 0 {
			return path, nil
		}
	}

	return "", errors.New("Not found")
}

func findJavaPath(version int) string {
	if version == 0 {
		slog.Info("Version not specified. Using the system default")
		return "java"
	}

	path, err := getJavaPathFromInstalledVersion(version)
	if err != nil {
		slog.Warn("Error finding java installation. Using the system default", slog.Any("error", err))
		return "java"
	}

	slog.Info("Found java installation matching requested version", slog.String("path", path), slog.Int("requested_version", version))

	return path
}

func LaunchServer(config *runner.Config, srv *ServerInstance) error {
	if _, err := os.Stat(fs.LocateServer(config.Server.Version)); err != nil {
		return err
	}

	ver, err := GetLastServerVersion()
	if err != nil {
		return err
	}
	if ver != config.Server.Version {
		slog.Info("Different version of server selected. cleaning up...", slog.String("old", ver), slog.String("new", config.Server.Version))

		ents, err := os.ReadDir(fs.DataPath("gamedata"))
		if err != nil {
			return err
		}
		for _, ent := range ents {
			if ent.Name() == "server.properties" || ent.Name() == "world" || strings.HasPrefix(ent.Name(), "ss@") {
				continue
			}
			if err := os.RemoveAll(fs.DataPath(filepath.Join("gamedata", ent.Name()))); err != nil {
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
	allocSize := 1024
	if _, err := os.Stat("/.dockerenv"); err != nil {
		// It is non-dev environment. Guess allocSize from total memory
		totalMem, err := systemutil.GetTotalMemory()
		if err != nil {
			slog.Error("Error retrieving total memory", slog.Any("error", err))
		} else {
			totalMemMiB := totalMem / 1024 / 1024
			allocSize = totalMemMiB - 1024
		}
	}

	var launchCommand []string
	if len(config.Server.CustomCommand) > 0 {
		launchCommand = config.Server.CustomCommand
		for i := 0; i < len(launchCommand); i++ {
			if launchCommand[i] == "{server_jar}" {
				launchCommand[i] = fs.LocateServer(config.Server.Version)
			}
		}
	} else {
		launchCommand = []string{
			findJavaPath(config.Server.JavaVersion),
			fmt.Sprintf("-Xmx%dM", allocSize),
			fmt.Sprintf("-Xms%dM", allocSize),
			"-jar",
			fs.LocateServer(config.Server.Version),
			"nogui",
		}
	}
	go func() {
		slog.Info("Launching Minecraft server", slog.String("server_name", config.Server.Version), slog.Any("commandline", launchCommand))
		launchCount := 0
		prevLaunch := time.Now()
		for !srv.ShouldStop && !srv.RestartRequested {
			if srv.quickUndoBeforeRestart {
				if err := processQuickUndo(srv.quickUndoSlot); err != nil {
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
			cmd := exec.Command(launchCommand[0], launchCommand[1:]...)
			cmd.Dir = fs.DataPath("gamedata")
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
	return rpc.ToExteriord.Call("state/save", types.StateSetInput{
		Key:   "lastVersion",
		Value: config.Server.Version,
	}, nil)
}

func RemoveLastServerVersion() error {
	return rpc.ToExteriord.Call("state/remove", types.StateRemoveInput{
		Key: "lastVersion",
	}, nil)
}

func GetLastServerVersion() (string, error) {
	var version string
	if err := rpc.ToExteriord.Call("state/get", types.StateGetInput{
		Key: "lastVersion",
	}, &version); err != nil {
		return "", err
	}

	return version, nil
}
