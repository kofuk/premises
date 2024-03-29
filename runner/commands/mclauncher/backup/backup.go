package backup

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
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

func (self *BackupService) UploadWorldData(config *runner.Config, options UploadOptions) error {
	if options.SourceDir == "" {
		options.SourceDir = fs.LocateWorldData("")
	}
	if options.TmpFileName == "" {
		options.TmpFileName = "world.tar.zst"
	}

	return self.doUploadWorldData(config, &options)
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

type UploadOptions struct {
	TmpFileName string
	SourceDir   string
}

func (self *BackupService) doUploadWorldData(config *runner.Config, options *UploadOptions) error {
	slog.Info("Uploading world archive...")

	archivePath := fs.LocateDataFile(options.TmpFileName)

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

	if err := os.Remove(fs.LocateDataFile(options.TmpFileName)); err != nil {
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

	tarCmd := exec.Command("tar", "-x")
	tarCmd.Dir = outDir
	tarCmd.Stdin = zstReader
	tarCmd.Stderr = os.Stderr
	tarCmd.Stdout = os.Stdout

	if err := tarCmd.Start(); err != nil {
		return err
	}

	tarCmd.Wait()

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

	xzReader, err := xz.NewReader(file)
	if err != nil {
		return err
	}

	tarCmd := exec.Command("tar", "-x")
	tarCmd.Dir = outDir
	tarCmd.Stderr = os.Stderr
	tarCmd.Stdout = os.Stdout

	tarStdin, err := tarCmd.StdinPipe()
	if err != nil {
		return err
	}
	defer tarStdin.Close()

	if err := tarCmd.Start(); err != nil {
		return err
	}

	if _, err := io.Copy(tarStdin, xzReader); err != nil {
		return err
	}

	tarCmd.Wait()

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

	if err := os.RemoveAll(fs.LocateWorldData("world")); err != nil {
		return err
	}
	if _, err := os.Stat(fs.LocateWorldData("world_nether")); err == nil {
		if err := os.RemoveAll(fs.LocateWorldData("world_nether")); err != nil {
			return err
		}
	}
	if _, err := os.Stat(fs.LocateWorldData("world_the_end")); err == nil {
		if err := os.RemoveAll(fs.LocateWorldData("world_the_end")); err != nil {
			return err
		}
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

func doPrepareUploadData(options *UploadOptions) error {
	slog.Info("Creating world archive...")

	tarArgs := []string{"-c", "world"}

	// "Paper" mod server saves world data in separate dirs.
	if s, err := os.Stat(filepath.Join(options.SourceDir, "world_nether")); err == nil && s.IsDir() {
		tarArgs = append(tarArgs, "world_nether")
	}
	if s, err := os.Stat(filepath.Join(options.SourceDir, "world_the_end")); err == nil && s.IsDir() {
		tarArgs = append(tarArgs, "world_the_end")
	}

	tarCmd := exec.Command("tar", tarArgs...)
	tarCmd.Dir = options.SourceDir
	tarStdout, err := tarCmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer tarStdout.Close()

	outFile, err := os.Create(fs.LocateDataFile(options.TmpFileName))
	if err != nil {
		return err
	}
	defer outFile.Close()

	if err := tarCmd.Start(); err != nil {
		return err
	}

	zstWriter, err := zstd.NewWriter(outFile, zstd.WithEncoderConcurrency(runtime.NumCPU()), zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return err
	}
	defer zstWriter.Close()

	if _, err := io.Copy(zstWriter, tarStdout); err != nil {
		return err
	}

	tarCmd.Wait()

	slog.Info("Creating world archive...Done")
	return nil
}

func PrepareUploadData(options UploadOptions) error {
	if options.SourceDir == "" {
		options.SourceDir = fs.LocateWorldData("")
	}
	if options.TmpFileName == "" {
		options.TmpFileName = "world.tar.zst"
	}
	return doPrepareUploadData(&options)
}
