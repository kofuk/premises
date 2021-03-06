package backup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/kofuk/go-mega"
	"github.com/kofuk/premises/mcmanager/config"
	log "github.com/sirupsen/logrus"
)

const (
	preserveHistoryCount = 5
)

func MakeBackupName(archiveVersion int) string {
	verStr := "latest"
	if archiveVersion != 0 {
		verStr = strconv.Itoa(archiveVersion)
	}
	return fmt.Sprintf("%s.tar.xz", verStr)
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
		if getNodeByName(nodes, MakeBackupName(i)) == nil {
			emptySlot = i
			break
		}
	}

	if emptySlot == -1 {
		if oldest := getNodeByName(nodes, MakeBackupName(preserveHistoryCount)); oldest != nil {
			if err := m.Delete(oldest, true); err != nil {
				return err
			}
		}
		emptySlot = preserveHistoryCount
	}

	for i := emptySlot - 1; i >= 0; i-- {
		oldName := MakeBackupName(i)
		newName := MakeBackupName(i + 1)
		if targetNode := getNodeByName(nodes, oldName); targetNode != nil {
			if err := m.Rename(targetNode, newName); err != nil {
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

	var worldFolderName string
	if ctx.Debug {
		worldFolderName = "worlds.dev"
	} else {
		worldFolderName = "worlds"
	}

	worldsFolder, err := makeSureCloudFolderExists(m, dataRoot, worldFolderName)
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
	TmpFileName   string
	SourceDir string
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
	node, err := m.UploadFile(archivePath, worldsFolder, MakeBackupName(0), &progress)
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
		options.TmpFileName = "world.tar.xz"
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
	if err := m.DownloadFile(archive, ctx.LocateDataFile("world.tar.xz"), &progress); err != nil {
		return err
	}
	log.Info("Downloading world archive...Done")

	return nil
}

func ExtractWorldArchiveIfNeeded(ctx *config.PMCMContext) error {
	if _, err := os.Stat(ctx.LocateDataFile("world.tar.xz")); os.IsNotExist(err) {
		log.Info("No world archive exists; continue...")
		return nil
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

	inFile, err := os.Open(ctx.LocateDataFile("world.tar.xz"))
	if err != nil {
		return err
	}
	defer inFile.Close()

	numThreads := runtime.NumCPU() - 1
	if numThreads < 1 {
		numThreads = 1
	}
	xzCmd := exec.Command("xz", "--decompress", "--threads", strconv.Itoa(numThreads))
	xzStdout, err := xzCmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer xzStdout.Close()
	xzStdin, err := xzCmd.StdinPipe()
	if err != nil {
		return err
	}
	// We should close this explicitly after input file written.
	defer func() {
		if xzStdin != nil {
			xzStdin.Close()
		}
	}()
	xzCmd.Stderr = os.Stderr

	tarCmd := exec.Command("tar", "-x")
	tarCmd.Dir = ctx.LocateWorldData("")
	tarCmd.Stdin = xzStdout
	tarCmd.Stderr = os.Stderr
	tarCmd.Stdout = os.Stdout

	if err := xzCmd.Start(); err != nil {
		return err
	}
	if err := tarCmd.Start(); err != nil {
		return err
	}

	if _, err := io.Copy(xzStdin, inFile); err != nil {
		return err
	}
	xzStdin.Close()
	xzStdin = nil

	xzCmd.Wait()
	tarCmd.Wait()

	log.Info("Extracting world archive...Done")

	if err := os.Remove(ctx.LocateDataFile("world.tar.xz")); err != nil {
		return err
	}

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

	numThreads := runtime.NumCPU() - 1
	if numThreads < 1 {
		numThreads = 1
	}
	xzCmd := exec.Command("xz", "--compress", "--threads", strconv.Itoa(numThreads))
	xzCmd.Stdin = tarStdout
	xzStdout, err := xzCmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer xzStdout.Close()

	outFile, err := os.Create(ctx.LocateDataFile(options.TmpFileName))
	if err != nil {
		return err
	}
	defer outFile.Close()

	if err := tarCmd.Start(); err != nil {
		return err
	}
	if err := xzCmd.Start(); err != nil {
		return err
	}

	if _, err := io.Copy(outFile, xzStdout); err != nil {
		return err
	}

	tarCmd.Wait()
	xzCmd.Wait()

	log.Info("Creating world archive...Done")
	return nil
}

func PrepareUploadData(ctx *config.PMCMContext, options UploadOptions) error {
	if options.SourceDir == "" {
		options.SourceDir = ctx.LocateWorldData("")
	}
	if options.TmpFileName == "" {
		options.TmpFileName = "world.tar.xz"
	}
	return doPrepareUploadData(ctx, &options);
}
