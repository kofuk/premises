package world

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	gofs "io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/api"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/fs"
	"github.com/kofuk/premises/runner/internal/util"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/kofuk/premises/runner/internal/commands/mclauncher/world")

type WorldService struct {
	client *api.Client
}

func New(endpoint, authKey string) *WorldService {
	return &WorldService{
		client: api.New(endpoint, authKey, otelhttp.DefaultClient),
	}
}

func (w *WorldService) getExtractionPipeline(name string) (*ExtractionPipeline, error) {
	c, err := NewFileCreator(env.DataPath("gamedata/world"))
	if err != nil {
		return nil, err
	}

	switch {
	case strings.HasSuffix(name, ".zip"):
		return &ExtractionPipeline{
			U: (*ZipUnarchiver)(nil),
			C: c,
		}, nil
	case strings.HasSuffix(name, ".tar.xz"):
		return &ExtractionPipeline{
			D: (*XZDecompressor)(nil),
			U: (*TarUnarchiver)(nil),
			C: c,
		}, nil
	case strings.HasSuffix(name, ".tar.zst"):
		return &ExtractionPipeline{
			D: (*ZstdDecompressor)(nil),
			U: (*TarUnarchiver)(nil),
			C: c,
		}, nil
	}
	return nil, errors.New("unsupported archive type")
}

func (w *WorldService) getDownloadURL(ctx context.Context, genID string) (string, error) {
	resp, err := w.client.CreateWorldDownloadURL(ctx, genID)
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

func (w *WorldService) DownloadWorldData(ctx context.Context, config *runner.Config) error {
	slog.Info("Downloading world archive...")
	if err := fs.RemoveIfExists(env.DataPath("gamedata/world")); err != nil {
		return err
	}

	pl, err := w.getExtractionPipeline(config.GameConfig.World.GenerationId)
	if err != nil {
		return err
	}

	url, err := w.getDownloadURL(ctx, config.GameConfig.World.GenerationId)
	if err != nil {
		return fmt.Errorf("unable to get download URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to download %s: %w", config.GameConfig.World.GenerationId, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to download %s: %s", config.GameConfig.World.GenerationId, resp.Status)
	}

	reader := util.NewProgressReader(ctx, resp.Body, entity.EventWorldDownload, int(resp.ContentLength))

	if err := pl.Run(reader); err != nil {
		return err
	}

	slog.Info("Downloading world archive...Done")

	return nil
}

func (w *WorldService) GetLatestKey(ctx context.Context, world string) (string, error) {
	resp, err := w.client.GetLatestWorldID(ctx, world)
	if err != nil {
		return "", err
	}

	return resp.WorldID, nil
}

func (w *WorldService) UploadWorldData(ctx context.Context, config *runner.Config) (string, error) {
	return w.doUploadWorldData(ctx, config)
}

func (w *WorldService) getUploadURL(ctx context.Context, worldName string) (string, string, error) {
	resp, err := w.client.CreateWorldUploadURL(ctx, worldName)
	if err != nil {
		return "", "", err
	}

	return resp.URL, resp.WorldID, nil
}

func (w *WorldService) doUploadWorldData(ctx context.Context, config *runner.Config) (string, error) {
	slog.Info("Uploading world archive...")

	archivePath := env.DataPath("world.tar.zst")

	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	url, key, err := w.getUploadURL(ctx, config.GameConfig.World.Name)
	if err != nil {
		return "", err
	}

	reader := util.NewProgressReader(ctx, file, entity.EventWorldUpload, int(fileInfo.Size())).ToSeekable()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, reader)
	if err != nil {
		return "", err
	}
	req.ContentLength = fileInfo.Size()
	req.Header.Set("Content-Type", "application/zstd")

	resp, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upload failed: %s", resp.Status)
	}

	io.CopyN(io.Discard, resp.Body, 10*1024)

	slog.Info("Uploading world archive...Done")

	if err := os.Remove(archivePath); err != nil {
		return "", err
	}

	return key, nil
}

func writeTar(to io.Writer, baseDir string, dirs ...string) error {
	tw := tar.NewWriter(to)
	defer tw.Close()

	creationTime := time.Now()

	for _, dir := range dirs {
		levelDatWritten := false
		if f, err := os.Open(filepath.Join(baseDir, dir, "level.dat")); err == nil {
			hdr := &tar.Header{
				Typeflag:   tar.TypeDir,
				Name:       dir,
				Size:       0,
				Mode:       0755,
				Uid:        1000,
				Gid:        1000,
				ModTime:    creationTime,
				AccessTime: creationTime,
				ChangeTime: creationTime,
				Format:     tar.FormatGNU,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				f.Close()
				return err
			}

			stat, err := f.Stat()
			if err != nil {
				f.Close()
				return err
			}
			hdr = &tar.Header{
				Typeflag:   tar.TypeReg,
				Name:       filepath.Join(dir, "level.dat"),
				Size:       stat.Size(),
				Mode:       0644,
				Uid:        1000,
				Gid:        1000,
				ModTime:    creationTime,
				AccessTime: creationTime,
				ChangeTime: creationTime,
				Format:     tar.FormatGNU,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				f.Close()
				return err
			}
			if _, err := io.Copy(tw, f); err != nil {
				f.Close()
				return err
			}
			f.Close()

			levelDatWritten = true
		}

		filesystem := os.DirFS(filepath.Join(baseDir, dir))

		err := gofs.WalkDir(filesystem, ".", func(path string, d gofs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if levelDatWritten && (path == "." || path == "level.dat") {
				return nil
			}

			hdr := &tar.Header{
				Typeflag:   tar.TypeDir,
				Name:       filepath.Join(dir, path),
				Size:       0,
				Uid:        1000,
				Gid:        1000,
				ModTime:    creationTime,
				AccessTime: creationTime,
				ChangeTime: creationTime,
				Format:     tar.FormatGNU,
			}

			switch {
			case d.Type().IsDir():
				hdr.Typeflag = tar.TypeDir
				hdr.Mode = 0755
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}

			case d.Type().IsRegular():
				file, err := os.Open(filepath.Join(baseDir, dir, path))
				if err != nil {
					return err
				}

				stat, err := file.Stat()
				if err != nil {
					file.Close()
					return err
				}

				hdr.Typeflag = tar.TypeReg
				hdr.Mode = 0644
				hdr.Size = stat.Size()
				if err := tw.WriteHeader(hdr); err != nil {
					file.Close()
					return err
				}

				if _, err := io.Copy(tw, file); err != nil {
					file.Close()
					return err
				}
				file.Close()

			default:
				return errors.New("unsupported file type")
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func createArchive() error {
	slog.Info("Creating world archive...")

	outFile, err := os.Create(env.DataPath("world.tar.zst"))
	if err != nil {
		return err
	}
	defer outFile.Close()

	zstWriter, err := zstd.NewWriter(outFile, zstd.WithEncoderConcurrency(runtime.NumCPU()), zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return err
	}
	defer zstWriter.Close()

	if err := writeTar(zstWriter, env.DataPath("gamedata"), "world"); err != nil {
		return err
	}

	slog.Info("Creating world archive...Done")
	return nil
}

func PrepareUploadData(ctx context.Context) error {
	_, span := tracer.Start(ctx, "Create archive")
	defer span.End()

	return createArchive()
}
