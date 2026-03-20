package exteriord

import (
	"archive/tar"
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/klauspost/compress/zstd"
	"github.com/kofuk/premises/backend/common"
	"github.com/kofuk/premises/backend/runner/commands/exteriord/exterior"
	"github.com/kofuk/premises/backend/runner/commands/exteriord/outbound"
	"github.com/kofuk/premises/backend/runner/commands/exteriord/proc"
	"github.com/kofuk/premises/backend/runner/config"
	"github.com/kofuk/premises/backend/runner/env"
	"github.com/kofuk/premises/backend/runner/rpc"
)

func extractResources(ctx context.Context) error {
	archivePath := env.DataPath("resources.tar.zst")
	resourcesDir := env.DataPath("resources")

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		slog.InfoContext(ctx, "No resources archive found, skipping extraction")
		return nil
	}

	slog.InfoContext(ctx, "Extracting resources...")

	if err := os.RemoveAll(resourcesDir); err != nil {
		return err
	}

	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		return err
	}

	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return err
	}

	decompressor, err := zstd.NewReader(archiveFile)
	if err != nil {
		return err
	}
	defer decompressor.Close()

	unarchiver := tar.NewReader(decompressor)
	for {
		header, err := unarchiver.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		destPath := env.DataPath("resources/" + header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode&0777))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, unarchiver); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		default:
			slog.WarnContext(ctx, "Unsupported file type in resources archive", slog.String("file", header.Name))

		}
	}

	// Clean up the archive after extraction
	os.Remove(archivePath)

	return nil
}

func Run(ctx context.Context, args []string) int {
	signal.Ignore(syscall.SIGHUP)

	if err := extractResources(ctx); err != nil {
		slog.ErrorContext(ctx, "Failed to extract resources", slog.Any("error", err))
		return 1
	}

	slog.InfoContext(ctx, "Starting premises-runner...", slog.String("protocol_version", common.Version))

	config, err := config.Load()
	if err != nil {
		slog.ErrorContext(ctx, "Unable to load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, cancelFn := context.WithCancel(ctx)

	msgChan := make(chan outbound.OutboundMessage, 8)

	ob := outbound.NewServer(config.ControlPanel, config.AuthKey, msgChan)
	go ob.Start(ctx)

	stateStore := NewStateStore(NewLocalStorageStateBackend(env.DataPath("states.json")))

	rpcHandler := NewRPCHandler(rpc.DefaultServer, msgChan, stateStore, cancelFn)
	rpcHandler.Bind()

	e := exterior.New()

	setupTask := e.RegisterTask("Initialize Server",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--setup"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		))
	monitoring := e.RegisterTask("Game Monitoring Service",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--launcher"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	systemUpdate := e.RegisterTask("Keep System Up-to-date",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--update-packages"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Connector",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--connector"),
			proc.Restart(proc.RestartOnFailure),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	e.RegisterTask("Snapshot Service",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--snapshot-helper"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Clean Up",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--clean"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), monitoring, systemUpdate)

	e.Run(ctx)

	return 0
}
