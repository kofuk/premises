package backup

import (
	"archive/tar"
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	goFs "io/fs"
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
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/util"
	"github.com/ulikunitz/xz"
)

const (
	preserveHistoryCount = 5
)

type BackupService struct {
	s3     *s3wrap.Client
	bucket string
}

func New(awsAccessKey, awsSecretKey, s3Endpoint, bucket string) *BackupService {
	if strings.HasPrefix(s3Endpoint, "http://s3.premises.local:") {
		// When S3 endpoint is localhost, it should be a development environment on Docker.
		// We implicitly rewrite the address so that we can access S3 host.
		s3Endpoint = strings.Replace(s3Endpoint, "http://s3.premises.local", "http://host.docker.internal", 1)
	}

	s3 := s3wrap.New(awsAccessKey, awsSecretKey, s3Endpoint)

	return &BackupService{
		s3:     s3,
		bucket: bucket,
	}
}

func makeBackupName() string {
	return fmt.Sprintf("%s.tar.zst", time.Now().Format(time.DateTime))
}

func (self *BackupService) DownloadWorldData(config *runner.Config) error {
	slog.Info("Downloading world archive...")
	resp, err := self.s3.GetObject(context.Background(), self.bucket, config.World.GenerationId)
	if err != nil {
		return fmt.Errorf("Unable to download %s: %w", config.World.GenerationId, err)
	}
	defer resp.Body.Close()

	size := resp.Size
	if size < 1 {
		size = 1
	}

	progress := make(chan int)
	defer close(progress)
	go func() {
		ticker := time.NewTicker(time.Second)
		showNext := true
		var total int64
		for {
			select {
			case <-ticker.C:
				showNext = true
			case chunkSize, ok := <-progress:
				if !ok {
					return
				}

				total += int64(chunkSize)

				if showNext {
					percentage := total * 100 / size
					if err := exterior.SendMessage("serverStatus", runner.Event{
						Type: runner.EventStatus,
						Status: &runner.StatusExtra{
							EventCode: entity.EventWorldDownload,
							Progress:  int(percentage),
						},
					}); err != nil {
						slog.Error("Unable to write server status", slog.Any("error", err))
					}

					showNext = false
				}
			}
		}
	}()

	ext := getFileExtension(config.World.GenerationId)
	file, err := os.Create(fs.LocateDataFile("world" + ext))
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(&util.ProgressWriter{W: file, Ch: progress}, resp.Body); err != nil {
		return err
	}
	slog.Info("Downloading world archive...Done")

	return nil
}

func (self *BackupService) GetLatestKey(world string) (string, error) {
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

func (self *BackupService) UploadWorldData(config *runner.Config) error {
	return self.doUploadWorldData(config)
}

func SaveLastWorldHash(config *runner.Config, hash string) error {
	file, err := os.Create(fs.LocateDataFile("last_world"))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(hash); err != nil {
		return err
	}
	return nil
}

func RemoveLastWorldHash(config *runner.Config) error {
	if err := os.Remove(fs.LocateDataFile("last_world")); err != nil {
		return err
	}
	return nil
}

func GetLastWorldHash(config *runner.Config) (string, bool, error) {
	file, err := os.Open(fs.LocateDataFile("last_world"))
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

func getFileExtension(name string) string {
	index := strings.IndexRune(name, '.')
	if index < 0 {
		return ""
	}
	return name[index:]
}

func (self *BackupService) doUploadWorldData(config *runner.Config) error {
	slog.Info("Uploading world archive...")

	archivePath := fs.LocateDataFile("world.tar.zst")

	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	size := fileInfo.Size()
	if size < 1 {
		size = 1
	}
	progress := make(chan int)
	defer close(progress)

	go func() {
		ticker := time.NewTicker(time.Second)
		showNext := true
		var prevPercentage int64
		var totalUploaded int64
		for {
			select {
			case <-ticker.C:
				showNext = true
			case chunkSize, ok := <-progress:
				if !ok {
					return
				}

				totalUploaded += int64(chunkSize)

				if showNext {
					percentage := totalUploaded * 100 / size
					if percentage != prevPercentage {
						if err := exterior.SendMessage("serverStatus", runner.Event{
							Type: runner.EventStatus,
							Status: &runner.StatusExtra{
								EventCode: entity.EventWorldUpload,
								Progress:  int(percentage),
							},
						}); err != nil {
							slog.Error("Unable to write server status", slog.Any("error", err))
						}
					}
					prevPercentage = percentage

					showNext = false
				}
			}
		}
	}()

	key := fmt.Sprintf("%s/%s", config.World.Name, makeBackupName())
	if err := self.s3.PutObject(context.Background(), self.bucket, key, &util.ProgressReader{R: file, Ch: progress}, fileInfo.Size()); err != nil {
		return fmt.Errorf("Unable to upload %s: %w", key, err)
	}
	slog.Info("Uploading world archive...Done")

	if err := os.Remove(archivePath); err != nil {
		return err
	}

	if err := SaveLastWorldHash(config, key); err != nil {
		slog.Warn("Error saving last world hash", slog.Any("error", err))
	}

	return nil
}

func (self *BackupService) RemoveOldBackups(config *runner.Config) error {
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

func extractTar(r io.Reader, outDir string) error {
	tr := tar.NewReader(r)
	for {
		th, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch th.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(outDir, th.Name), 0755); err != nil {
				return err
			}

		case tar.TypeReg:
			outFile, err := os.Create(filepath.Join(outDir, th.Name))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

		default:
			return fmt.Errorf("Unsupported header type: %v", th.Typeflag)
		}
	}

	return nil
}

func extractZstWorldArchive(inFile, outDir string) error {
	file, err := os.Open(inFile)
	if err != nil {
		return err
	}

	zstReader, err := zstd.NewReader(file, zstd.WithDecoderConcurrency(runtime.NumCPU()))
	if err != nil {
		return err
	}
	defer zstReader.Close()

	if err := extractTar(zstReader, outDir); err != nil {
		return err
	}

	return nil
}

func extractXzWorldArchive(inFile, outDir string) error {
	numThreads := runtime.NumCPU() - 1
	if numThreads < 1 {
		numThreads = 1
	}

	file, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer file.Close()

	xzReader, err := xz.NewReader(file)
	if err != nil {
		return err
	}

	if err := extractTar(xzReader, outDir); err != nil {
		return err
	}

	return nil
}

func extractZipWorldArchive(inFile, outDir string) error {
	r, err := zip.OpenReader(inFile)
	if err != nil {
		if err == zip.ErrInsecurePath {
			r.Close()
		}
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/") {
			// If the name ends with "/", it is a directory.
			// (technically it can have a content, but we don't care about it)
			continue
		}

		absName := filepath.Join(outDir, f.Name)

		if err := os.MkdirAll(filepath.Dir(absName), 0755); err != nil {
			return err
		}

		outFile, err := os.Create(absName)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			rc.Close()
			outFile.Close()
			return err
		}

		rc.Close()
		outFile.Close()
	}

	return nil
}

func worldArchiveExists() bool {
	candidates := []string{
		"world.tar.zst",
		"world.tar.xz",
		"world.zip",
	}

	for _, name := range candidates {
		if _, err := os.Stat(fs.LocateDataFile(name)); err == nil {
			return true
		}
	}

	return false
}

func ExtractWorldArchiveIfNeeded() error {
	if !worldArchiveExists() {
		// No archive to extract. Continue.
		return nil
	}

	if err := fs.RemoveIfExists(fs.LocateWorldData("world")); err != nil {
		return err
	}

	slog.Info("Extracting world archive...")

	tempDir, err := os.MkdirTemp("/tmp", "premises-temp")
	if err != nil {
		return err
	}

	extractors := []struct {
		name     string
		fn       func(inFile, outDir string) error
		fileName string
	}{
		{
			name:     "zip",
			fn:       extractZipWorldArchive,
			fileName: fs.LocateDataFile("world.zip"),
		},
		{
			name:     "xz",
			fn:       extractXzWorldArchive,
			fileName: fs.LocateDataFile("world.tar.xz"),
		},
		{
			name:     "zstd",
			fn:       extractZstWorldArchive,
			fileName: fs.LocateDataFile("world.tar.zst"),
		},
	}

	for _, extractor := range extractors {
		slog.Info(fmt.Sprintf("Try extract archive using %s extractor", extractor.name))

		err := extractor.fn(extractor.fileName, tempDir)
		if err == nil {
			slog.Info("Archive extraction succeeded")
			os.Remove(extractor.fileName)
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
	}

	slog.Info("Detecting and renaming world data...")
	if err := moveWorldDataToGameDir(tempDir); err != nil {
		slog.Error("Failed to prepare world data from archive", slog.Any("error", err))
	}
	slog.Info("Detecting and renaming world data...Done")

	os.RemoveAll(tempDir)

	return nil
}

func writeTar(to io.Writer, baseDir string, dirs ...string) error {
	tw := tar.NewWriter(to)
	defer tw.Close()

	creationTime := time.Now()

	for _, dir := range dirs {
		filesystem := os.DirFS(filepath.Join(baseDir, dir))

		err := goFs.WalkDir(filesystem, ".", func(path string, d goFs.DirEntry, err error) error {
			if err != nil {
				return err
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

	outFile, err := os.Create(fs.LocateDataFile("world.tar.zst"))
	if err != nil {
		return err
	}
	defer outFile.Close()

	zstWriter, err := zstd.NewWriter(outFile, zstd.WithEncoderConcurrency(runtime.NumCPU()), zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return err
	}
	defer zstWriter.Close()

	if err := writeTar(zstWriter, fs.LocateWorldData(""), "world"); err != nil {
		return err
	}

	slog.Info("Creating world archive...Done")
	return nil
}

func PrepareUploadData() error {
	return createArchive()
}
