package game

import (
	"bufio"
	"context"
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
	"github.com/kofuk/premises/runner/commands/mclauncher/backup"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
	"github.com/kofuk/premises/runner/systemutil"
)

type Launcher struct {
	config                 *runner.Config
	world                  *backup.BackupService
	ctx                    context.Context
	cancel                 context.CancelFunc
	shouldRestart          bool
	shouldStop             bool
	FinishWG               sync.WaitGroup
	lastActive             time.Time
	quickUndoBeforeRestart bool
	quickUndoSlot          int
	serverPid              int
	Rcon                   *Rcon
}

var (
	RestartRequested = errors.New("Restart requested")
)

var (
	activePlayerListRegexp = regexp.MustCompile("^There are ([0-9]+) of a max of [0-9]+ players online")
)

func NewLauncher(config *runner.Config, backup *backup.BackupService) *Launcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &Launcher{
		config: config,
		world:  backup,
		ctx:    ctx,
		cancel: cancel,
		Rcon:   NewRcon("127.0.0.1:25575", "x"),
	}
}

func getLastWorld() (string, error) {
	var value string
	if err := rpc.ToExteriord.Call("state/get", types.StateGetInput{
		Key: "lastWorld",
	}, &value); err != nil {
		return "", err
	}

	return value, nil
}

func clearLastWorld() error {
	if err := rpc.ToExteriord.Call("state/remove", types.StateRemoveInput{
		Key: "lastWorld",
	}, nil); err != nil {
		return err
	}

	return nil
}

func storeLastWorld(lastWorld string) error {
	if err := rpc.ToExteriord.Call("state/save", types.StateSetInput{
		Key:   "lastWorld",
		Value: lastWorld,
	}, nil); err != nil {
		return err
	}

	return nil
}

func getLastServerVersion() (string, error) {
	var version string
	if err := rpc.ToExteriord.Call("state/get", types.StateGetInput{
		Key: "lastVersion",
	}, &version); err != nil {
		return "", err
	}

	return version, nil
}

func clearLastServerVersion() error {
	return rpc.ToExteriord.Call("state/remove", types.StateRemoveInput{
		Key: "lastVersion",
	}, nil)
}

func storeLastServerVersion(config *runner.Config) error {
	return rpc.ToExteriord.Call("state/save", types.StateSetInput{
		Key:   "lastVersion",
		Value: config.Server.Version,
	}, nil)
}

func (l *Launcher) downloadWorld() error {
	if l.config.World.ShouldGenerate {
		if err := fs.RemoveIfExists(fs.DataPath("gamedata/world")); err != nil {
			slog.Error("Unable to remove world directory", slog.Any("error", err))
		}

		return nil
	}

	if l.config.World.GenerationId == "@/latest" {
		genId, err := l.world.GetLatestKey(l.config.World.Name)
		if err != nil {
			return err
		}
		l.config.World.GenerationId = genId
	}

	lastWorld, err := getLastWorld()
	if err != nil {
		return err
	}

	if lastWorld == "" || l.config.World.GenerationId != lastWorld {
		if err := clearLastWorld(); err != nil {
			slog.Error("Failed to remove last world hash", slog.Any("error", err))
		}

		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		})

		if err := l.world.DownloadWorldData(l.config); err != nil {
			return err
		}

		if err := backup.ExtractWorldArchiveIfNeeded(); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (l *Launcher) downloadServerJar() error {
	if _, err := os.Stat(fs.LocateServer(l.config.Server.Version)); err == nil {
		slog.Info("No need to download server.jar")
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventGameDownload,
		},
	})

	slog.Info("Downloading Minecraft server...", slog.String("url", l.config.Server.DownloadUrl))

	if err := DownloadServerJar(l.config.Server.DownloadUrl, fs.LocateServer(l.config.Server.Version)); err != nil {
		return err
	}

	slog.Info("Downloading Minecraft server...Done")

	return nil
}

func (l *Launcher) uploadWorld() error {
	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	})
	if err := backup.PrepareUploadData(); err != nil {
		slog.Error("Failed to create world archive", slog.Any("error", err))
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		return err
	}

	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldUpload,
		},
	})
	key, err := l.world.UploadWorldData(l.config)
	if err != nil {
		slog.Error("Failed to upload world data", slog.Any("error", err))
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		return err
	}
	if err := storeLastWorld(key); err != nil {
		slog.Error("Unable to store last world key", slog.Any("error", err))
	}

	return nil
}

func (l *Launcher) startServer() error {
	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	})

	if _, err := os.Stat(fs.LocateServer(l.config.Server.Version)); err != nil {
		return err
	}

	ver, err := getLastServerVersion()
	if err != nil {
		return err
	}
	if ver != l.config.Server.Version {
		slog.Info("Different version of server selected. cleaning up...", slog.String("old", ver), slog.String("new", l.config.Server.Version))

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

	if err := clearLastServerVersion(); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := generateServerProps(l.config); err != nil {
		return err
	}

	if err := signEulaForServer(); err != nil {
		return err
	}

	l.FinishWG.Add(1)
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
	if len(l.config.Server.CustomCommand) > 0 {
		launchCommand = l.config.Server.CustomCommand
		for i := 0; i < len(launchCommand); i++ {
			if launchCommand[i] == "{server_jar}" {
				launchCommand[i] = fs.LocateServer(l.config.Server.Version)
			}
		}
	} else {
		launchCommand = []string{
			findJavaPath(l.config.Server.JavaVersion),
			fmt.Sprintf("-Xmx%dM", allocSize),
			fmt.Sprintf("-Xms%dM", allocSize),
			"-jar",
			fs.LocateServer(l.config.Server.Version),
			"nogui",
		}
	}

	go func() {
		slog.Info("Launching Minecraft server", slog.String("server_name", l.config.Server.Version), slog.Any("commandline", launchCommand))
		launchCount := 0
		prevLaunch := time.Now()
		for !l.shouldStop {
			select {
			case <-l.ctx.Done():
				break
			default:
			}

			if l.quickUndoBeforeRestart {
				if err := processQuickUndo(l.quickUndoSlot); err != nil {
					slog.Error("Error processing quick undo", slog.Any("error", err))
				}

				launchCount = 0
				l.quickUndoBeforeRestart = false
			}

			if launchCount == 5 {
				if time.Now().Sub(prevLaunch) < 3*time.Minute {
					break
				}
			}
			cmd := exec.Command(launchCommand[0], launchCommand[1:]...)
			cmd.Dir = fs.DataPath("gamedata")
			cmdStdout, _ := cmd.StdoutPipe()
			cmd.Stderr = os.Stderr
			cmd.Start()
			l.serverPid = cmd.Process.Pid
			MonitorServer(l.config, l, cmdStdout)
			cmd.Wait()
			cmdStdout.Close()
			exitCode := cmd.ProcessState.ExitCode()
			slog.Info("Server exited", slog.Int("exit_code", exitCode))
			if exitCode == 0 {
				break
			}
			launchCount++
		}
		l.FinishWG.Done()
	}()

	return nil
}

func (l *Launcher) StopToRestart() {
	l.shouldRestart = true
	l.Stop()
	l.cancel()
}

func (l *Launcher) Launch() error {
	if err := l.downloadWorld(); err != nil {
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		return nil
	}

	if l.config.Server.PreferDetected {
		slog.Info("Read server version from level.dat")
		if err := DetectAndUpdateVersion(l.config); err != nil {
			slog.Error("Error detecting Minecraft version", slog.Any("error", err))
		}
	}

	if err := l.downloadServerJar(); err != nil {
		slog.Error("Couldn't download server.jar", slog.Any("error", err))
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		})
		return err
	}

	if err := l.startServer(); err != nil {
		slog.Error("Failed to launch Minecraft server", slog.Any("error", err))
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLaunchErr,
			},
		})
		return err
	}

	l.AddToWhiteList(l.config.Whitelist)
	l.AddToOp(l.config.Operators)

	l.Wait()

	if err := l.uploadWorld(); err != nil {
		return err
	}

	if err := l.world.RemoveOldBackups(l.config); err != nil {
		slog.Error("Unable to delete outdated backups", slog.Any("error", err))
	}

	if l.shouldRestart {
		return RestartRequested
	}

	return nil
}

func (l *Launcher) isServerActive() bool {
	resp, err := l.Rcon.Execute("list")
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

func (l *Launcher) Wait() {
	done := make(chan interface{})

	l.lastActive = time.Now()
	go func() {
		ticker := time.NewTicker(time.Minute)

		for {
			select {
			case <-ticker.C:
				if l.isServerActive() {
					l.lastActive = time.Now()
				} else {
					if l.lastActive.Add(30 * time.Minute).Before(time.Now()) {
						l.Stop()
					}
				}
			case <-done:
				goto end
			}
		}

	end:
		ticker.Stop()
	}()

	l.FinishWG.Wait()
	close(done)
}

func (l *Launcher) Stop() {
	exterior.DispatchMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventStopping,
		},
	})

	l.shouldStop = true

	if _, err := l.Rcon.Execute("stop"); err != nil {
		slog.Error("Failed to send stop command to server", slog.Any("error", err))
	}
}

func (l *Launcher) SaveAll() error {
	if _, err := l.Rcon.Execute("save-all"); err != nil {
		return err
	}
	return nil
}

func (l *Launcher) AddToWhiteList(players []string) {
	for _, player := range players {
		_, err := l.Rcon.Execute(fmt.Sprintf("whitelist add %s", player))
		if err != nil {
			slog.Error("Failed to execute whitelist command", slog.Any("error", err))
		}
	}
}

func (l *Launcher) AddToOp(players []string) {
	for _, player := range players {
		_, err := l.Rcon.Execute(fmt.Sprintf("op %s", player))
		if err != nil {
			slog.Error("Failed to execute op command", slog.Any("error", err))
		}
	}
}

func (l *Launcher) SendChat(message string) error {
	if _, err := l.Rcon.Execute(fmt.Sprintf("tellraw @a \"%s\"", message)); err != nil {
		return err
	}

	return nil
}

func (l *Launcher) GetSeed() (string, error) {
	seed, err := l.Rcon.Execute("seed")
	if err != nil {
		return "", err
	}

	if len(seed) < 8 || seed[:7] != "Seed: [" || seed[len(seed)-1] != ']' {
		return "", errors.New("Failed to retrieve seed")
	}

	return seed[7 : len(seed)-1], nil
}

func (l *Launcher) QuickUndo(slot int) error {
	exterior.DispatchMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventStopping,
		},
	})

	l.quickUndoBeforeRestart = true
	l.quickUndoSlot = slot

	proc, err := os.FindProcess(l.serverPid)
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

func SendStartedEvent(config *runner.Config, srv *Launcher) {
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

func MonitorServer(config *runner.Config, srv *Launcher, stdout io.ReadCloser) error {
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

			if err := storeLastServerVersion(config); err != nil {
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
	serverProps := NewServerProperties()
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
