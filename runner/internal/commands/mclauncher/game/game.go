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

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/world"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/exterior"
	"github.com/kofuk/premises/runner/internal/fs"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/rpc/types"
	"github.com/kofuk/premises/runner/internal/system"
	"go.opentelemetry.io/otel/trace"
)

const ScopeName = "github.com/kofuk/premises/runner/internal/commands/mclauncher/game"

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

func NewLauncher(ctx context.Context, config *runner.Config, world *world.WorldService) *Launcher {
	ctx, cancel := context.WithCancel(ctx)

	l := &Launcher{
		config: config,
		world:  world,
		ctx:    ctx,
		cancel: cancel,
		Rcon:   NewRcon("127.0.0.1:25575", "x"),
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
		go l.sendStartedEvent(ctx, config)

		l.AddToWhiteList(l.config.GameConfig.Whitelist)
		l.AddToOp(l.config.GameConfig.Operators)

		exterior.SendEvent(ctx, runner.Event{
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

func getLastWorld(ctx context.Context) (string, error) {
	var value string
	if err := rpc.ToExteriord.Call(ctx, "state/get", types.StateGetInput{
		Key: "lastWorld",
	}, &value); err != nil {
		return "", err
	}

	return value, nil
}

func clearLastWorld(ctx context.Context) error {
	if err := rpc.ToExteriord.Call(ctx, "state/remove", types.StateRemoveInput{
		Key: "lastWorld",
	}, nil); err != nil {
		return err
	}

	return nil
}

func storeLastWorld(ctx context.Context, lastWorld string) error {
	if err := rpc.ToExteriord.Call(ctx, "state/save", types.StateSetInput{
		Key:   "lastWorld",
		Value: lastWorld,
	}, nil); err != nil {
		return err
	}

	return nil
}

func getLastServerVersion(ctx context.Context) (string, error) {
	var version string
	if err := rpc.ToExteriord.Call(ctx, "state/get", types.StateGetInput{
		Key: "lastVersion",
	}, &version); err != nil {
		return "", err
	}

	return version, nil
}

func clearLastServerVersion(ctx context.Context) error {
	return rpc.ToExteriord.Call(ctx, "state/remove", types.StateRemoveInput{
		Key: "lastVersion",
	}, nil)
}

func storeLastServerVersion(ctx context.Context, config *runner.Config) error {
	return rpc.ToExteriord.Call(ctx, "state/save", types.StateSetInput{
		Key:   "lastVersion",
		Value: config.GameConfig.Server.Version,
	}, nil)
}

func (l *Launcher) downloadWorld(ctx context.Context) error {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer(ScopeName)
	ctx, span := tracer.Start(ctx, "Download world")
	defer span.End()

	if l.config.GameConfig.World.ShouldGenerate {
		if err := fs.RemoveIfExists(env.DataPath("gamedata/world")); err != nil {
			slog.Error("Unable to remove world directory", slog.Any("error", err))
		}

		return nil
	}

	if l.config.GameConfig.World.GenerationId == "@/latest" {
		genId, err := l.world.GetLatestKey(ctx, l.config.GameConfig.World.Name)
		if err != nil {
			return fmt.Errorf("failed to get latest world ID: %w", err)
		}
		l.config.GameConfig.World.GenerationId = genId
	}

	lastWorld, err := getLastWorld(ctx)
	if err != nil {
		return err
	}

	if lastWorld == "" || l.config.GameConfig.World.GenerationId != lastWorld {
		if err := clearLastWorld(ctx); err != nil {
			slog.Error("Failed to remove last world hash", slog.Any("error", err))
		}

		exterior.SendEvent(ctx, runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		})

		if err := l.world.DownloadWorldData(ctx, l.config); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (l *Launcher) downloadServerJar(ctx context.Context) error {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer(ScopeName)
	ctx, span := tracer.Start(ctx, "Download server.jar")
	defer span.End()

	if l.config.GameConfig.Server.PreferDetected {
		slog.Info("Read server version from level.dat")
		if err := DetectAndUpdateVersion(ctx, l.config); err != nil {
			slog.Error("Error detecting Minecraft version", slog.Any("error", err))
		}
	}

	if _, err := os.Stat(env.LocateServer(l.config.GameConfig.Server.Version)); err == nil {
		slog.Info("No need to download server.jar")
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	exterior.SendEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventGameDownload,
		},
	})

	slog.Info("Downloading Minecraft server...", slog.String("url", l.config.GameConfig.Server.DownloadUrl))

	if err := DownloadServerJar(ctx, l.config.GameConfig.Server.DownloadUrl, env.LocateServer(l.config.GameConfig.Server.Version)); err != nil {
		return err
	}

	slog.Info("Downloading Minecraft server...Done")

	return nil
}

func (l *Launcher) uploadWorld(ctx context.Context) error {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer(ScopeName)
	ctx, span := tracer.Start(ctx, "Upload world")
	defer span.End()

	exterior.SendEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	})
	if err := world.PrepareUploadData(ctx); err != nil {
		slog.Error("Failed to create world archive", slog.Any("error", err))
		exterior.SendEvent(ctx, runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		return err
	}

	exterior.SendEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldUpload,
		},
	})
	key, err := l.world.UploadWorldData(ctx, l.config)
	if err != nil {
		slog.Error("Failed to upload world data", slog.Any("error", err))
		exterior.SendEvent(ctx, runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		return err
	}
	if err := storeLastWorld(ctx, key); err != nil {
		slog.Error("Unable to store last world key", slog.Any("error", err))
	}

	return nil
}

func getAllocSizeMiB() int {
	if env.IsDevEnv() {
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
	for {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", "127.0.0.1:32109")
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
	timeout := l.config.GameConfig.Server.InactiveTimeout
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
				l.Stop(ctx)
				return
			}
		}
	}
}

func (l *Launcher) executeServer(ctx context.Context, cmdline []string) error {
	slog.Info("Launching Minecraft server", slog.String("server_name", l.config.GameConfig.Server.Version), slog.Any("commandline", cmdline))

	exterior.SendEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventLoading,
		},
	})

	if l.beforeLaunch != nil {
		l.beforeLaunch(l)
	}

	ctx, cancel := context.WithCancel(ctx)

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

	err := system.Cmd(ctx, cmdline[0], cmdline[1:], system.WithWorkingDir(env.DataPath("gamedata")))

	cancel()

	return err
}

func (l *Launcher) startServer(ctx context.Context) error {
	allocSize := getAllocSizeMiB()

	var launchCommand []string
	if len(l.config.GameConfig.Server.CustomCommand) > 0 {
		launchCommand = l.config.GameConfig.Server.CustomCommand
		for i := 0; i < len(launchCommand); i++ {
			if launchCommand[i] == "{server_jar}" {
				launchCommand[i] = env.LocateServer(l.config.GameConfig.Server.Version)
			}
		}
	} else {
		launchCommand = []string{
			findJavaPath(ctx, l.config.GameConfig.Server.JavaVersion),
			fmt.Sprintf("-Xmx%dM", allocSize),
			fmt.Sprintf("-Xms%dM", allocSize),
			"-jar",
			env.LocateServer(l.config.GameConfig.Server.Version),
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

		if err := l.executeServer(ctx, launchCommand); err == nil {
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

func (l *Launcher) StopToRestart(ctx context.Context) {
	l.shouldRestart = true
	l.Stop(ctx)
	l.cancel()
}

func cleanGameDir() error {
	ents, err := os.ReadDir(env.DataPath("gamedata"))
	if err != nil {
		return err
	}

	var errs []error
	for _, ent := range ents {
		if ent.Name() == "server.properties" || ent.Name() == "world" || strings.HasPrefix(ent.Name(), "ss@") {
			continue
		}
		if err := os.RemoveAll(env.DataPath(filepath.Join("gamedata", ent.Name()))); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (l *Launcher) cleanGameDirIfVersionChanged(ctx context.Context) error {
	ver, err := getLastServerVersion(ctx)
	if err != nil {
		return err
	}
	if ver == l.config.GameConfig.Server.Version {
		return nil
	}

	slog.Info("Different version of server selected. cleaning up...", slog.String("old", ver), slog.String("new", l.config.GameConfig.Server.Version))

	if err := cleanGameDir(); err != nil {
		return err
	}

	return nil
}

func (l *Launcher) prepareEnvironment(ctx context.Context) error {
	exterior.SendEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	})

	if err := l.cleanGameDirIfVersionChanged(ctx); err != nil {
		slog.Error(err.Error())
	}

	if err := clearLastServerVersion(ctx); err != nil {
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

func (l *Launcher) Launch(ctx context.Context) error {
	if err := l.downloadWorld(ctx); err != nil {
		slog.Error("Unable to donwload world data", slog.Any("error", err))

		exterior.SendEvent(ctx, runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		return nil
	}

	if err := l.downloadServerJar(ctx); err != nil {
		slog.Error("Couldn't download server.jar", slog.Any("error", err))
		exterior.SendEvent(ctx, runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		})
		return err
	}

	if err := l.prepareEnvironment(ctx); err != nil {
		slog.Error("Failed to prepare environment", slog.Any("error", err))
		exterior.SendEvent(ctx, runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLaunchErr,
			},
		})
		return err
	}

	if err := l.startServer(ctx); err != nil {
		slog.Error("Failed to launch Minecraft server", slog.Any("error", err))
		exterior.SendEvent(ctx, runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLaunchErr,
			},
		})
		return err
	}

	if err := storeLastServerVersion(ctx, l.config); err != nil {
		slog.Error("Error saving last server versoin", slog.Any("error", err))
	}

	if err := l.uploadWorld(ctx); err != nil {
		return err
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

func (l *Launcher) Stop(ctx context.Context) {
	exterior.DispatchEvent(ctx, runner.Event{
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

func (l *Launcher) QuickUndo(ctx context.Context, slot int) error {
	exterior.DispatchEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventStopping,
		},
	})

	l.restoringSnapshot = true
	l.quickUndoSlot = slot

	l.Stop(ctx)

	return nil
}

func (l *Launcher) sendStartedEvent(ctx context.Context, config *runner.Config) {
	slog.Debug("Send Started event...")

	data := new(runner.StartedExtra)

	data.ServerVersion = config.GameConfig.Server.Version
	data.World.Name = config.GameConfig.World.Name
	seed, err := l.Rcon.Seed()
	if err != nil {
		slog.Error("Failed to retrieve seed", slog.Any("error", err))
	}
	data.World.Seed = seed

	exterior.SendEvent(ctx, runner.Event{
		Type:    runner.EventStarted,
		Started: data,
	})
}

func generateServerProps(config *runner.Config) error {
	serverProps := NewServerProperties()
	serverProps.LoadConfig(config)

	out, err := os.Create(env.DataPath("gamedata/server.properties"))
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
	eulaFile, err := os.Create(env.DataPath("gamedata/eula.txt"))
	if err != nil {
		return err
	}
	defer eulaFile.Close()
	eulaFile.WriteString("eula=true")
	return nil
}

func processQuickUndo(slot int) error {
	if err := os.RemoveAll(env.DataPath("gamedata/world")); err != nil {
		return err
	}
	if err := os.Mkdir(env.DataPath("gamedata/world"), 0755); err != nil {
		return err
	}

	if err := fs.CopyAll(env.DataPath("gamedata", fmt.Sprintf("ss@quick%d/world", slot)), env.DataPath("gamedata/world")); err != nil {
		return err
	}

	return nil
}
