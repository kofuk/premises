package backup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4Signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/logging"
	"github.com/klauspost/compress/zstd"
	entity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/config"
	"github.com/kofuk/premises/runner/exterior"
	log "github.com/sirupsen/logrus"
	"github.com/ulikunitz/xz"
)

const (
	preserveHistoryCount = 5
)

type BackupProvider struct {
	s3Client *s3.Client
	bucket   string
}

type noAcceptEncodingSigner struct {
	signer s3.HTTPSignerV4
}

func newNoAcceptEncodingSigner(signer s3.HTTPSignerV4) *noAcceptEncodingSigner {
	return &noAcceptEncodingSigner{
		signer: signer,
	}
}

func (self *noAcceptEncodingSigner) SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string, service string, region string, signingTime time.Time, optFns ...func(*v4Signer.SignerOptions)) error {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	r.Header.Del("Accept-Encoding")
	err := self.signer.SignHTTP(ctx, credentials, r, payloadHash, service, region, signingTime, optFns...)
	if acceptEncoding != "" {
		r.Header.Set("Accept-Encoding", acceptEncoding)
	}
	return err
}

func New(awsAccessKey, awsSecretKey, s3Endpoint, bucket string) *BackupProvider {
	if strings.HasPrefix(s3Endpoint, "http://localhost:") {
		// When S3 endpoint is localhost, it should be a development environment on Docker.
		// We implicitly rewrite the address so that we can access S3 host.
		s3Endpoint = strings.Replace(s3Endpoint, "http://localhost", "http://host.docker.internal", 1)
	}
	config := aws.Config{
		Credentials:  credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, ""),
		BaseEndpoint: &s3Endpoint,
		Logger: logging.LoggerFunc(func(classification logging.Classification, format string, v ...interface{}) {
			log.WithField("source", "aws-sdk").Debug(v...)
		}),
		ClientLogMode: aws.LogRequestWithBody | aws.LogResponseWithBody,
	}

	s3Client := s3.NewFromConfig(config, func(options *s3.Options) {
		options.UsePathStyle = true
		defSigner := v4Signer.NewSigner(func(so *v4Signer.SignerOptions) {
			so.Logger = options.Logger
			so.LogSigning = options.ClientLogMode.IsSigning()
			so.DisableURIPathEscaping = true
		})
		options.HTTPSignerV4 = newNoAcceptEncodingSigner(defSigner)
	})

	return &BackupProvider{
		s3Client: s3Client,
		bucket:   bucket,
	}
}

func makeBackupName() string {
	return fmt.Sprintf("%s.tar.zst", time.Now().Format(time.DateTime))
}

func (self *BackupProvider) DownloadWorldData(ctx *config.PMCMContext) error {
	log.Info("Downloading world archive...")
	resp, err := self.s3Client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &self.bucket,
		Key:    &ctx.Cfg.World.GenerationId,
	})
	if err != nil {
		return fmt.Errorf("Unable to download %s: %w", ctx.Cfg.World.GenerationId, err)
	}
	defer resp.Body.Close()

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
					percentage := total * 100 / *resp.ContentLength
					if err := exterior.SendMessage("serverStatus", entity.Event{
						Type: entity.EventStatus,
						Status: &entity.StatusExtra{
							EventCode: entity.EventWorldDownload,
							Progress:  int(percentage),
						},
					}); err != nil {
						log.Error(err)
					}

					showNext = false
				}
			}
		}
	}()

	ext := getFileExtension(ctx.Cfg.World.GenerationId)
	file, err := os.Create(ctx.LocateDataFile("world" + ext))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(&ProgressWriter{
		writer: file,
		notify: progress,
	}, resp.Body)
	if err != nil {
		return err
	}
	log.Info("Downloading world archive...Done")

	return nil
}

func (self *BackupProvider) UploadWorldData(ctx *config.PMCMContext, options UploadOptions) error {
	if options.SourceDir == "" {
		options.SourceDir = ctx.LocateWorldData("")
	}
	if options.TmpFileName == "" {
		options.TmpFileName = "world.tar.zst"
	}

	return self.doUploadWorldData(ctx, &options)
}

func SaveLastWorldHash(ctx *config.PMCMContext, hash string) error {
	file, err := os.Create(ctx.LocateDataFile("last_world"))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(hash); err != nil {
		return err
	}
	return nil
}

func RemoveLastWorldHash(ctx *config.PMCMContext) error {
	if err := os.Remove(ctx.LocateDataFile("last_world")); err != nil {
		return err
	}
	return nil
}

func GetLastWorldHash(ctx *config.PMCMContext) (string, bool, error) {
	file, err := os.Open(ctx.LocateDataFile("last_world"))
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

func (self *BackupProvider) doUploadWorldData(ctx *config.PMCMContext, options *UploadOptions) error {
	archivePath := ctx.LocateDataFile(options.TmpFileName)
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		return err
	}

	size := fileInfo.Size()
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
						if err := exterior.SendMessage("serverStatus", entity.Event{
							Type: entity.EventStatus,
							Status: &entity.StatusExtra{
								EventCode: entity.EventWorldUpload,
								Progress:  int(percentage),
							},
						}); err != nil {
							log.Error(err)
						}
					}
					prevPercentage = percentage

					showNext = false
				}
			}
		}
	}()

	log.Info("Uploading world archive...")
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", ctx.Cfg.World.Name, makeBackupName())
	_, err = self.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &self.bucket,
		Key:    &key,
		Body: &ProgressReader{
			reader: file,
			notify: progress,
		},
		ContentLength: aws.Int64(fileInfo.Size()),
	}, s3.WithAPIOptions(v4Signer.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
	if err != nil {
		return fmt.Errorf("Unable to upload %s: %w", key, err)
	}
	log.Info("Uploading world archive...Done")

	if err := os.Remove(ctx.LocateDataFile(options.TmpFileName)); err != nil {
		return err
	}

	if err := SaveLastWorldHash(ctx, key); err != nil {
		log.WithError(err).Warn("Error saving last world hash")
	}

	return nil
}

func (self *BackupProvider) RemoveOldBackups(ctx *config.PMCMContext) error {
	resp, err := self.s3Client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: &self.bucket,
		Prefix: aws.String(fmt.Sprintf("%s/", ctx.Cfg.World.Name)),
	})
	if err != nil {
		return err
	}

	objs := resp.Contents
	if len(objs) <= 5 {
		// We don't need to delete old backups. Exiting...
		return nil
	}

	sort.Slice(objs, func(i, j int) bool {
		return objs[i].LastModified.Unix() > objs[j].LastModified.Unix()
	})

	var objectIds []types.ObjectIdentifier
	for _, obj := range objs[5:] {
		objectIds = append(objectIds, types.ObjectIdentifier{
			Key: obj.Key,
		})
	}
	if _, err := self.s3Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: &self.bucket,
		Delete: &types.Delete{
			Objects: objectIds,
		},
	}); err != nil {
		return err
	}

	return nil
}

func extractZstWorldArchive(inFile io.Reader, outDir string) error {
	log.Println("Extracting Zstandard...")

	zstReader, err := zstd.NewReader(inFile, zstd.WithDecoderConcurrency(runtime.NumCPU()))
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

	log.Println("Extracting Zstandard...Done")

	return nil
}

func extractXzWorldArchive(inFile io.Reader, outDir string) error {
	log.Println("Extracting XZ...")

	numThreads := runtime.NumCPU() - 1
	if numThreads < 1 {
		numThreads = 1
	}

	xzReader, err := xz.NewReader(inFile)
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

	log.Println("Extracting XZ...Done")

	return nil
}

func extractZipWorldArchive(inFile, outDir string) error {
	log.Println("Extracting Zip...")

	unzipCmd := exec.Command("unzip", inFile)
	unzipCmd.Dir = outDir
	unzipCmd.Stdout = os.Stdout
	unzipCmd.Stderr = os.Stderr
	if err := unzipCmd.Run(); err != nil {
		return err
	}

	log.Println("Extracting Zip...Done")

	return nil
}

func ExtractWorldArchiveIfNeeded(ctx *config.PMCMContext) error {
	if _, err := os.Stat(ctx.LocateDataFile("world.tar.zst")); os.IsNotExist(err) {
		if _, err := os.Stat(ctx.LocateDataFile("world.tar.xz")); os.IsNotExist(err) {
			if _, err := os.Stat(ctx.LocateDataFile("world.zip")); os.IsNotExist(err) {
				log.Info("No world archive exists; continue...")
				return nil
			}
		}
	} else if err != nil {
		return err
	}

	if err := os.RemoveAll(ctx.LocateWorldData("world")); err != nil {
		return err
	}
	if _, err := os.Stat(ctx.LocateWorldData("world_nether")); err == nil {
		if err := os.RemoveAll(ctx.LocateWorldData("world_nether")); err != nil {
			return err
		}
	}
	if _, err := os.Stat(ctx.LocateWorldData("world_the_end")); err == nil {
		if err := os.RemoveAll(ctx.LocateWorldData("world_the_end")); err != nil {
			return err
		}
	}

	log.Info("Extracting world archive...")

	tempDir, err := os.MkdirTemp("/tmp", "premises-temp")
	if err != nil {
		return err
	}

	inFile, err := os.Open(ctx.LocateDataFile("world.tar.zst"))
	if err != nil {
		inFile, err := os.Open(ctx.LocateDataFile("world.tar.xz"))
		if err != nil {
			_, err := os.Stat(ctx.LocateDataFile("world.zip"))
			if err != nil {
				return err
			}

			if err := extractZipWorldArchive(ctx.LocateDataFile("world.zip"), tempDir); err != nil {
				return err
			}

			log.Info("Extracting world archive...Done")

			if err := os.Remove(ctx.LocateDataFile("world.zip")); err != nil {
				return err
			}
		} else {
			defer inFile.Close()

			if err := extractXzWorldArchive(inFile, tempDir); err != nil {
				return err
			}

			log.Info("Extracting world archive...Done")

			if err := os.Remove(ctx.LocateDataFile("world.tar.xz")); err != nil {
				return err
			}
		}
	} else {
		defer inFile.Close()

		if err := extractZstWorldArchive(inFile, tempDir); err != nil {
			return err
		}

		log.Info("Extracting world archive...Done")

		if err := os.Remove(ctx.LocateDataFile("world.tar.zst")); err != nil {
			return err
		}
	}

	log.Info("Detecting and renaming world data...")
	if err := moveWorldDataToGameDir(ctx, tempDir); err != nil {
		log.WithError(err).Error("Failed to prepare world data from archive")
	}
	log.Info("Detecting and renaming world data...Done")

	os.RemoveAll(tempDir)

	return nil
}

func doPrepareUploadData(ctx *config.PMCMContext, options *UploadOptions) error {
	log.Info("Creating world archive...")

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

	outFile, err := os.Create(ctx.LocateDataFile(options.TmpFileName))
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

	log.Info("Creating world archive...Done")
	return nil
}

func PrepareUploadData(ctx *config.PMCMContext, options UploadOptions) error {
	if options.SourceDir == "" {
		options.SourceDir = ctx.LocateWorldData("")
	}
	if options.TmpFileName == "" {
		options.TmpFileName = "world.tar.zst"
	}
	return doPrepareUploadData(ctx, &options)
}
