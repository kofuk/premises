package world

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	gofs "io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/common/s3wrap"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/util"
)

const (
	preserveHistoryCount = 5
)

type WorldService struct {
	s3     *s3wrap.Client
	bucket string
}

func New(awsAccessKey, awsSecretKey, s3Endpoint, bucket string) *WorldService {
	if strings.HasPrefix(s3Endpoint, "http://s3.premises.local:") {
		// When S3 endpoint is localhost, it should be a development environment on Docker.
		// We implicitly rewrite the address so that we can access S3 host.
		s3Endpoint = strings.Replace(s3Endpoint, "http://s3.premises.local", "http://host.docker.internal", 1)
	}

	s3 := s3wrap.New(awsAccessKey, awsSecretKey, s3Endpoint)

	return &WorldService{
		s3:     s3,
		bucket: bucket,
	}
}

func makeArchiveName() string {
	return fmt.Sprintf("%s.tar.zst", time.Now().Format(time.DateTime))
}

func (self *WorldService) getExtractionPipeline(name string) (*ExtractionPipeline, error) {
	c, err := NewFileCreator(fs.DataPath("gamedata/world"))
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
	return nil, errors.New("Unsupported archive type")
}

func (self *WorldService) DownloadWorldData(config *runner.Config) error {
	slog.Info("Downloading world archive...")
	if err := fs.RemoveIfExists(fs.DataPath("gamedata/world")); err != nil {
		return err
	}

	pl, err := self.getExtractionPipeline(config.World.GenerationId)
	if err != nil {
		return err
	}

	resp, err := self.s3.GetObject(context.Background(), self.bucket, config.World.GenerationId)
	if err != nil {
		return fmt.Errorf("Unable to download %s: %w", config.World.GenerationId, err)
	}
	defer resp.Body.Close()

	reader := util.NewProgressReader(resp.Body, entity.EventWorldDownload, int(resp.Size))

	if err := pl.Run(reader); err != nil {
		return err
	}

	slog.Info("Downloading world archive...Done")

	return nil
}

func (self *WorldService) GetLatestKey(world string) (string, error) {
	objs, err := self.s3.ListObjects(context.Background(), self.bucket, s3wrap.WithPrefix(world+"/"))
	if err != nil {
		return "", err
	}
	if len(objs) == 0 {
		return "", errors.New("No backup found for world")
	}

	sort.Slice(objs, func(i, j int) bool {
		return objs[i].Timestamp.Unix() > objs[j].Timestamp.Unix()
	})

	return objs[0].Key, nil
}

func (self *WorldService) UploadWorldData(config *runner.Config) (string, error) {
	return self.doUploadWorldData(config)
}

func (self *WorldService) doUploadWorldData(config *runner.Config) (string, error) {
	slog.Info("Uploading world archive...")

	archivePath := fs.DataPath("world.tar.zst")

	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s/%s", config.World.Name, makeArchiveName())
	reader := util.NewProgressReader(file, entity.EventWorldUpload, int(fileInfo.Size())).ToSeekable()
	if err := self.s3.PutObject(context.Background(), self.bucket, key, reader, fileInfo.Size()); err != nil {
		return "", fmt.Errorf("Unable to upload %s: %w", key, err)
	}
	slog.Info("Uploading world archive...Done")

	if err := os.Remove(archivePath); err != nil {
		return "", err
	}

	return key, nil
}

func (self *WorldService) RemoveOldBackups(config *runner.Config) error {
	objs, err := self.s3.ListObjects(context.Background(), self.bucket, s3wrap.WithPrefix(config.World.Name+"/"))
	if err != nil {
		return err
	}

	if len(objs) <= 5 {
		// We don't need to delete old backups. Exiting...
		return nil
	}

	sort.Slice(objs, func(i, j int) bool {
		return objs[i].Timestamp.Unix() > objs[j].Timestamp.Unix()
	})

	var keys []string
	for _, obj := range objs[5:] {
		keys = append(keys, obj.Key)
	}
	if err := self.s3.DeleteObjects(context.Background(), self.bucket, keys); err != nil {
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
				return errors.New("Unsupported file type")
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

	outFile, err := os.Create(fs.DataPath("world.tar.zst"))
	if err != nil {
		return err
	}
	defer outFile.Close()

	zstWriter, err := zstd.NewWriter(outFile, zstd.WithEncoderConcurrency(runtime.NumCPU()), zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return err
	}
	defer zstWriter.Close()

	if err := writeTar(zstWriter, fs.DataPath("gamedata"), "world"); err != nil {
		return err
	}

	slog.Info("Creating world archive...Done")
	return nil
}

func PrepareUploadData() error {
	return createArchive()
}
