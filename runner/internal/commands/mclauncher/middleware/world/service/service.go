package service

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/runner/internal/api"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/util"
)

type WorldService struct {
	client     *api.Client
	httpClient *http.Client
}

func NewWorldService(endpoint, authKey string, httpClient *http.Client) *WorldService {
	return &WorldService{
		client:     api.NewClient(endpoint, authKey, httpClient),
		httpClient: httpClient,
	}
}

func (w *WorldService) GetLatestResourceID(ctx context.Context, worldName string) (string, error) {
	resp, err := w.client.GetLatestWorldID(ctx, worldName)
	if err != nil {
		return "", err
	}
	return resp.WorldID, nil
}

func (w *WorldService) getExtractionPipeline(resourceID string, envProvider env.EnvProvider) (*ExtractionPipeline, error) {
	tmpDir, err := env.MkdirTemp(envProvider)
	if err != nil {
		return nil, err
	}

	c, err := NewFileCreator(envProvider.GetDataPath("gamedata/world"), tmpDir)
	if err != nil {
		return nil, err
	}

	switch {
	case strings.HasSuffix(resourceID, ".zip"):
		tmpDir, err := env.MkdirTemp(envProvider)
		if err != nil {
			return nil, err
		}
		return &ExtractionPipeline{
			U: NewZipUnarchiver(tmpDir),
			C: c,
		}, nil
	case strings.HasSuffix(resourceID, ".tar.xz"):
		return &ExtractionPipeline{
			D: NewXZDecompressor(),
			U: NewTarUnarchiver(),
			C: c,
		}, nil
	case strings.HasSuffix(resourceID, ".tar.zst"):
		return &ExtractionPipeline{
			D: NewZstdDecompressor(),
			U: NewTarUnarchiver(),
			C: c,
		}, nil
	}
	return nil, errors.New("unsupported archive type")
}

func (w *WorldService) DownloadWorld(ctx context.Context, resourceID string, envProvider env.EnvProvider) error {
	pipeline, err := w.getExtractionPipeline(resourceID, envProvider)
	if err != nil {
		return err
	}

	downloadURLResp, err := w.client.CreateWorldDownloadURL(ctx, resourceID)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURLResp.URL, nil)
	if err != nil {
		return err
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to download world data")
	}

	reader := util.NewProgressReader(ctx, resp.Body, entity.EventWorldDownload, int(resp.ContentLength))

	if err := pipeline.Run(reader); err != nil {
		return err
	}

	return nil
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

		err := fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
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

func (w *WorldService) createArchive(envProvider env.EnvProvider) (string, error) {
	tmpDir, err := env.MkdirTemp(envProvider)
	if err != nil {
		return "", err
	}

	outFile, err := os.Create(filepath.Join(tmpDir, "world.tar.zst"))
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	zstWriter, err := zstd.NewWriter(outFile, zstd.WithEncoderConcurrency(runtime.NumCPU()), zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return "", err
	}
	defer zstWriter.Close()

	if err := writeTar(zstWriter, env.DataPath("gamedata"), "world"); err != nil {
		return "", err
	}

	return outFile.Name(), nil
}

func (w *WorldService) UploadWorld(ctx context.Context, worldName string, envProvider env.EnvProvider) (string, error) {
	archivePath, err := w.createArchive(envProvider)
	if err != nil {
		return "", err
	}
	defer os.Remove(archivePath)

	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	uploadURLResp, err := w.client.CreateWorldUploadURL(ctx, worldName)
	if err != nil {
		return "", err
	}

	reader := util.NewProgressReader(ctx, file, entity.EventWorldUpload, int(fileInfo.Size())).ToSeekable()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURLResp.URL, reader)
	if err != nil {
		return "", err
	}
	req.ContentLength = fileInfo.Size()
	req.Header.Set("Content-Type", "application/zstd")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upload failed: %s", resp.Status)
	}

	io.CopyN(io.Discard, resp.Body, 10*1024)

	return uploadURLResp.WorldID, nil
}
