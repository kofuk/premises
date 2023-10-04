package backup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/kofuk/go-mega"
	"github.com/kofuk/premises/runner/config"
	log "github.com/sirupsen/logrus"
	"github.com/ulikunitz/xz"
)

const (
	preserveHistoryCount = 5
)

func makeBackupName(archiveVersion int) string {
	verStr := "latest"
	if archiveVersion != 0 {
		verStr = strconv.Itoa(archiveVersion)
	}
	return fmt.Sprintf("%s.tar.zst", verStr)
}

func makeBackupGanerationName(ver int) string {
	if ver == 0 {
		return "latest"
	} else {
		return strconv.Itoa(ver)
	}
}

func makeSureCloudFolderExists(m *mega.Mega, parent *mega.Node, name string) (*mega.Node, error) {
	children, err := m.FS.GetChildren(parent)
	if err != nil {
		return nil, err
	}
	for _, folder := range children {
		if folder.GetName() == name {
			return folder, nil
		}
	}
	result, err := m.CreateDir(name, parent)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getNodeByName(nodes []*mega.Node, name string) *mega.Node {
	for _, node := range nodes {
		if node.GetName() == name {
			return node
		}
	}
	return nil
}

func getNodeByBackupGeneration(nodes []*mega.Node, gen int) *mega.Node {
	genName := makeBackupGanerationName(gen)
	zstName := genName + ".tar.zst"
	xzName := genName + ".tar.xz"
	zipName := genName + ".zip"

	for _, node := range nodes {
		name := node.GetName()
		if name == zstName || name == xzName || name == zipName {
			return node
		}
	}
	return nil
}

func getFileExtension(name string) string {
	index := strings.IndexRune(name, '.')
	if index < 0 {
		return ""
	}
	return name[index:]
}

func getNodeByHash(nodes []*mega.Node, hash string) *mega.Node {
	for _, node := range nodes {
		if node.GetHash() == hash {
			return node
		}
	}
	return nil
}

func rotateWorldArchives(ctx *config.PMCMContext, m *mega.Mega, parent *mega.Node) error {
	nodes, err := m.FS.GetChildren(parent)
	if err != nil {
		return err
	}

	// Find the first empty slot.
	emptySlot := -1
	for i := 0; i < preserveHistoryCount; i++ {
		if getNodeByBackupGeneration(nodes, i) == nil {
			emptySlot = i
			break
		}
	}

	if emptySlot == -1 {
		if oldest := getNodeByBackupGeneration(nodes, preserveHistoryCount); oldest != nil {
			if err := m.Delete(oldest, true); err != nil {
				return err
			}
		}
		emptySlot = preserveHistoryCount
	}

	for i := emptySlot - 1; i >= 0; i-- {
		if targetNode := getNodeByBackupGeneration(nodes, i); targetNode != nil {
			newName := makeBackupGanerationName(i + 1)
			ext := getFileExtension(targetNode.GetName())
			if err := m.Rename(targetNode, newName+ext); err != nil {
				return err
			}
		}
	}
	return nil
}

func getCloudWorldFolder(ctx *config.PMCMContext, m *mega.Mega, name string) (*mega.Node, error) {
	dataRoot, err := makeSureCloudFolderExists(m, m.FS.GetRoot(), "premises")
	if err != nil {
		return nil, err
	}

	worldsFolder, err := makeSureCloudFolderExists(m, dataRoot, ctx.Cfg.Mega.FolderName)
	if err != nil {
		return nil, err
	}
	worldFolder, err := makeSureCloudFolderExists(m, worldsFolder, name)
	if err != nil {
		return nil, err
	}
	return worldFolder, nil
}

type UploadOptions struct {
	TmpFileName string
	SourceDir   string
}

func doUploadWorldData(ctx *config.PMCMContext, options *UploadOptions) error {
	if ctx.Cfg.Mega.Email == "" {
		log.Error("Cannot sync world archive because Mega credential is not set.")
		return nil
	}

	m := mega.New()
	if err := m.Login(ctx.Cfg.Mega.Email, ctx.Cfg.Mega.Password); err != nil {
		return err
	}
	defer func() {
		if err := m.Logout(); err != nil {
			log.WithError(err).Warn("Failed to logout from Mega")
		}
	}()

	worldsFolder, err := getCloudWorldFolder(ctx, m, ctx.Cfg.World.Name)
	if err != nil {
		return err
	}

	log.Info("Rotating world archives...")
	if err := rotateWorldArchives(ctx, m, worldsFolder); err != nil {
		return err
	}
	log.Info("Rotating world archives...Done")

	archivePath := ctx.LocateDataFile(options.TmpFileName)
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		return err
	}

	size := fileInfo.Size()
	progress := make(chan int)

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
						ctx.NotifyStatus(fmt.Sprintf(ctx.L("world.uploading.pct"), int(percentage)))
					}
					prevPercentage = percentage

					showNext = false
				}
			}
		}
	}()

	log.Info("Uploading world archive...")
	node, err := m.UploadFile(archivePath, worldsFolder, makeBackupName(0), &progress)
	if err != nil {
		return err
	}
	log.Info("Uploading world archive...Done")

	if err := os.Remove(ctx.LocateDataFile(options.TmpFileName)); err != nil {
		return err
	}

	if err := SaveLastWorldHash(ctx, node.GetHash()); err != nil {
		log.WithError(err).Warn("Error saving last world hash")
	}

	return nil
}

func UploadWorldData(ctx *config.PMCMContext, options UploadOptions) error {
	if options.SourceDir == "" {
		options.SourceDir = ctx.LocateWorldData("")
	}
	if options.TmpFileName == "" {
		options.TmpFileName = "world.tar.zst"
	}

	return doUploadWorldData(ctx, &options)
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

func DownloadWorldData(ctx *config.PMCMContext) error {
	if ctx.Cfg.Mega.Email == "" {
		log.Error("Cannot sync world archive because Mega credential is not set.")
		return nil
	}

	m := mega.New()
	if err := m.Login(ctx.Cfg.Mega.Email, ctx.Cfg.Mega.Password); err != nil {
		return err
	}
	defer func() {
		if err := m.Logout(); err != nil {
			log.WithError(err).Warn("Failed to logout from Mega")
		}
	}()

	worldFolder, err := getCloudWorldFolder(ctx, m, ctx.Cfg.World.Name)
	if err != nil {
		return err
	}
	nodes, err := m.FS.GetChildren(worldFolder)
	if err != nil {
		return err
	}

	archive := getNodeByHash(nodes, ctx.Cfg.World.GenerationId)
	if archive == nil {
		log.WithField("gen_id", ctx.Cfg.World.GenerationId).Error("Can't find specified world archive; will start as-is.")
		return nil
	}

	size := archive.GetSize()
	progress := make(chan int)

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
						ctx.NotifyStatus(fmt.Sprintf(ctx.L("world.downloading.pct"), int(percentage)))
					}
					prevPercentage = percentage

					showNext = false
				}
			}
		}
	}()

	log.Info("Downloading world archive...")
	ext := getFileExtension(archive.GetName())
	if err := m.DownloadFile(archive, ctx.LocateDataFile("world"+ext), &progress); err != nil {
		return err
	}
	log.Info("Downloading world archive...Done")

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
