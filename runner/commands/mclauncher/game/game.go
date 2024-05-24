package game

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/world"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
	"github.com/kofuk/premises/runner/system"
)

type OnHealthyFunc func(l *Launcher)
type BeforeLaunchFunc func(l *Launcher)

type Launcher struct {
	config            *runner.Config
	world             *world.WorldService
	ctx               context.Context
	cancel            context.CancelFunc
	shouldRestart     bool
	restoringSnapshot bool
	quickUndoSlot     int
	Rcon              *Rcon
	onHealthy         OnHealthyFunc
	beforeLaunch      BeforeLaunchFunc
}

var (
	ErrRestartRequested = errors.New("restart requested")
)

func NewLauncher(config *runner.Config, world *world.WorldService) *Launcher {
	ctx, cancel := context.WithCancel(context.Background())

	rconAddr := "127.0.0.1:25575"
	if _, err := os.Stat("/.dockerenv"); err == nil {
		rconAddr = "127.0.0.2:25575"
	}

	l := &Launcher{
		config: config,
		world:  world,
		ctx:    ctx,
		cancel: cancel,
		Rcon:   NewRcon(rconAddr, "x"),
	}

	l.RegisterBeforeLaunchHook(func(l *Launcher) {
		if l.restoringSnapshot {
			if err := processQuickUndo(l.quickUndoSlot); err != nil {
				slog.Error("Error processing quick undo", slog.Any("error", err))
			}

			l.restoringSnapshot = false
		}
	})

	l.RegisterOnHealthyHook(func(l *Launcher) {
		go l.sendStartedEvent(config)

		l.AddToWhiteList(l.config.Whitelist)
		l.AddToOp(l.config.Operators)

		exterior.SendEvent(runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventRunning,
			},
		})
	})

	return l
}

func (l *Launcher) RegisterBeforeLaunchHook(fn BeforeLaunchFunc) {
	l.beforeLaunch = fn
}

func (l *Launcher) RegisterOnHealthyHook(fn OnHealthyFunc) {
	l.onHealthy = fn
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

		exterior.SendEvent(runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		})

		if err := l.world.DownloadWorldData(l.config); err != nil {
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

	exterior.SendEvent(runner.Event{
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
	exterior.SendEvent(runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	})
	if err := world.PrepareUploadData(); err != nil {
		slog.Error("Failed to create world archive", slog.Any("error", err))
		exterior.SendEvent(runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		return err
	}

	exterior.SendEvent(runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldUpload,
		},
	})
	key, err := l.world.UploadWorldData(l.config)
	if err != nil {
		slog.Error("Failed to upload world data", slog.Any("error", err))
		exterior.SendEvent(runner.Event{
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

func getAllocSizeMiB() int {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		// Docker environment.
		return 1024
	}

	totalMem, err := system.GetTotalMemory()
	if err != nil {
		slog.Error("Error retrieving total memory", slog.Any("error", err))
		return 1024
	}
	return totalMem/1024/1024 - 1024
}

func waitServerHealthy(ctx context.Context) error {
	serverAddr := "0.0.0.0:25565"
	if _, err := os.Stat("/.dockerenv"); err == nil {
		serverAddr = "127.0.0.2:25565"
	}
	for {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", serverAddr)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			slog.Debug("waitServerHealthy failed", slog.Any("error", err))

			time.Sleep(time.Second)
			continue
		}
		conn.Close()
		return nil
	}
}

func (l *Launcher) stopAfterLongInactive(ctx context.Context) {
	timeout := l.config.Server.InactiveTimeout
	if timeout < 0 {
		return
	} else if timeout < 5 {
		timeout = 5
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	lastActive := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if l.isServerActive() {
				lastActive = time.Now()
				continue
			}

			if time.Since(lastActive) > time.Duration(timeout)*time.Minute {
				l.Stop()
				return
			}
		}
	}
}

func (l *Launcher) executeServer(cmdline []string) error {
	slog.Info("Launching Minecraft server", slog.String("server_name", l.config.Server.Version), slog.Any("commandline", cmdline))

	exterior.SendEvent(runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventLoading,
		},
	})

	if l.beforeLaunch != nil {
		l.beforeLaunch(l)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := waitServerHealthy(ctx); err != nil {
			// We'll write the error to log, but don't want to treat this as an error.
			slog.Error("Failed to wait for the server to be healthy", slog.Any("error", err))
			return
		}
		l.onHealthy(l)
	}()
	go func() {
		l.stopAfterLongInactive(ctx)
	}()

	err := system.Cmd(cmdline[0], cmdline[1:], system.WithWorkingDir(fs.DataPath("gamedata")))

	cancel()

	return err
}

func (l *Launcher) startServer() error {
	allocSize := getAllocSizeMiB()

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

	var prevLaunch time.Time
	for {
		select {
		case <-l.ctx.Done():
			return nil
		default:
		}

		if err := l.executeServer(launchCommand); err == nil {
			if !l.restoringSnapshot {
				return nil
			}
		} else {
			slog.Error("Minecraft server failed", slog.Any("error", err))
			if time.Since(prevLaunch) < 10*time.Second {
				return fmt.Errorf("game seems to be crashed before the loading: %w", err)
			}
		}

		prevLaunch = time.Now()
	}
}

func (l *Launcher) StopToRestart() {
	l.shouldRestart = true
	l.Stop()
	l.cancel()
}

func cleanGameDir() error {
	ents, err := os.ReadDir(fs.DataPath("gamedata"))
	if err != nil {
		return err
	}

	var errs []error
	for _, ent := range ents {
		if ent.Name() == "server.properties" || ent.Name() == "world" || strings.HasPrefix(ent.Name(), "ss@") {
			continue
		}
		if err := os.RemoveAll(fs.DataPath(filepath.Join("gamedata", ent.Name()))); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (l *Launcher) cleanGameDirIfVersionChanged() error {
	ver, err := getLastServerVersion()
	if err != nil {
		return err
	}
	if ver == l.config.Server.Version {
		return nil
	}

	slog.Info("Different version of server selected. cleaning up...", slog.String("old", ver), slog.String("new", l.config.Server.Version))

	if err := cleanGameDir(); err != nil {
		return err
	}

	return nil
}

func (l *Launcher) prepareEnvironment() error {
	exterior.SendEvent(runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	})

	if err := l.cleanGameDirIfVersionChanged(); err != nil {
		slog.Error(err.Error())
	}

	if err := clearLastServerVersion(); err != nil {
		slog.Error(err.Error())
	}

	if err := generateServerProps(l.config); err != nil {
		return err
	}

	if err := generateEula(); err != nil {
		return err
	}

	return nil
}

func (l *Launcher) Launch() error {
	if err := l.downloadWorld(); err != nil {
		slog.Error("Unable to donwload world data", slog.Any("error", err))

		exterior.SendEvent(runner.Event{
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
		exterior.SendEvent(runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		})
		return err
	}

	if err := l.prepareEnvironment(); err != nil {
		slog.Error("Failed to prepare environment", slog.Any("error", err))
		exterior.SendEvent(runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLaunchErr,
			},
		})
		return err
	}

	if err := l.startServer(); err != nil {
		slog.Error("Failed to launch Minecraft server", slog.Any("error", err))
		exterior.SendEvent(runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLaunchErr,
			},
		})
		return err
	}

	if err := storeLastServerVersion(l.config); err != nil {
		slog.Error("Error saving last server versoin", slog.Any("error", err))
	}

	if err := l.uploadWorld(); err != nil {
		return err
	}

	if err := l.world.RemoveOldBackups(l.config); err != nil {
		slog.Error("Unable to delete outdated backups", slog.Any("error", err))
	}

	if l.shouldRestart {
		return ErrRestartRequested
	}

	return nil
}

func (l *Launcher) isServerActive() bool {
	list, err := l.Rcon.List()
	if err != nil {
		slog.Error("Failed to retrieve player list", slog.Any("error", err))
		return true
	}

	return len(list) > 0
}

func (l *Launcher) Stop() {
	exterior.DispatchEvent(runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventStopping,
		},
	})

	if err := l.Rcon.Stop(); err != nil {
		slog.Error("Failed to send stop command to server", slog.Any("error", err))
		return
	}
}

func (l *Launcher) SaveAll() error {
	if err := l.Rcon.SaveAll(); err != nil {
		return err
	}
	return nil
}

func (l *Launcher) AddToWhiteList(players []string) {
	for _, player := range players {
		if err := l.Rcon.AddToWhiteList(player); err != nil {
			slog.Error("Failed to execute whitelist command", slog.Any("error", err))
		}
	}
}

func (l *Launcher) AddToOp(players []string) {
	for _, player := range players {

		if err := l.Rcon.AddToOp(player); err != nil {
			slog.Error("Failed to execute op command", slog.Any("error", err))
		}
	}
}

func (l *Launcher) QuickUndo(slot int) error {
	exterior.DispatchEvent(runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventStopping,
		},
	})

	l.restoringSnapshot = true
	l.quickUndoSlot = slot

	l.Stop()

	return nil
}

func (l *Launcher) sendStartedEvent(config *runner.Config) {
	slog.Debug("Send Started event...")

	data := new(runner.StartedExtra)

	data.ServerVersion = config.Server.Version
	data.World.Name = config.World.Name
	seed, err := l.Rcon.Seed()
	if err != nil {
		slog.Error("Failed to retrieve seed", slog.Any("error", err))
	}
	data.World.Seed = seed

	exterior.SendEvent(runner.Event{
		Type:    runner.EventStarted,
		Started: data,
	})
}

func generateServerProps(config *runner.Config) error {
	serverProps := NewServerProperties()
	serverProps.LoadConfig(config)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		serverProps.DangerouslySetProperty("server-ip", "127.0.0.2")
	}

	out, err := os.Create(fs.DataPath("gamedata/server.properties"))
	if err != nil {
		return err
	}
	defer out.Close()

	if err := serverProps.Write(out); err != nil {
		return err
	}

	return nil
}

func generateEula() error {
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
	if err := os.Mkdir(fs.DataPath("gamedata/world"), 0755); err != nil {
		return err
	}

	if err := fs.CopyAll(fs.DataPath("gamedata", fmt.Sprintf("ss@quick%d/world", slot)), fs.DataPath("gamedata/world")); err != nil {
		return err
	}

	return nil
}
