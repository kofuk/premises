package gamesrv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"github.com/gorcon/rcon"
	log "github.com/sirupsen/logrus"

	"github.com/kofuk/premises/runner/backup"
	"github.com/kofuk/premises/runner/config"
)

type ServerInstance struct {
	Name                   string
	FinishWG               sync.WaitGroup
	ShouldStop             bool
	StartupFailed          bool
	rconMu                 sync.Mutex
	IsServerInitialized    bool
	IsGameFinished         bool
	RestartRequested       bool
	Crashed                bool
	lastActive             time.Time
	quickUndoBeforeRestart bool
	ServerPid              int
}

var (
	activePlayerListRegexp = regexp.MustCompile("^There are ([0-9]+) of a max of [0-9]+ players online")
)

func (srv *ServerInstance) isServerActive() bool {
	srv.rconMu.Lock()
	defer srv.rconMu.Unlock()
	conn, err := connectToRcon(srv)
	if err != nil {
		log.WithError(err).Error("Failed to connect rcon")
		return false
	}
	defer conn.Close()

	resp, err := conn.Execute("list")
	if err != nil {
		log.WithError(err).Error("Failed to send list command to server")
	}

	if match := activePlayerListRegexp.FindStringSubmatch(resp); match != nil {
		if match[1] == "0" {
			log.Println("Server is detected to be inactive")
			return false
		}
	}

	return true
}

func (srv *ServerInstance) Wait() {
	done := make(chan interface{})

	srv.lastActive = time.Now()
	go func() {
		ticker := time.NewTicker(time.Minute)

		for {
			select {
			case <-ticker.C:
				if srv.isServerActive() {
					srv.lastActive = time.Now()
				} else {
					if srv.lastActive.Add(30 * time.Minute).Before(time.Now()) {
						srv.Stop()
					}
				}
			case <-done:
				goto end
			}
		}

	end:
		ticker.Stop()
	}()

	srv.FinishWG.Wait()
	close(done)
}

func connectToRcon(srv *ServerInstance) (*rcon.Conn, error) {
	var err error
	for i := 0; i < 500; i++ {
		var conn *rcon.Conn
		conn, err = rcon.Dial("127.0.0.1:25575", "x")
		if err == nil {
			return conn, nil
		}
		if srv != nil {
			if srv.Crashed {
				return nil, errors.New("Server is crashed")
			} else if srv.ShouldStop {
				return nil, errors.New("Server is stopped")
			}
		}
		log.WithError(err).Info("Failed to connect rcon; retrying in 1 second")
		time.Sleep(time.Second)
	}
	return nil, err
}

func (srv *ServerInstance) Stop() {
	srv.rconMu.Lock()
	defer srv.rconMu.Unlock()
	conn, err := connectToRcon(srv)
	if err != nil {
		log.WithError(err).Error("Failed to connect rcon")
		return
	}
	defer conn.Close()

	resp, err := conn.Execute("stop")
	if err != nil {
		log.WithError(err).Error("Failed to send stop command to server")
	}
	log.Info(resp)
}

func (srv *ServerInstance) SaveAll() error {
	srv.rconMu.Lock()
	defer srv.rconMu.Unlock()
	conn, err := connectToRcon(srv)
	if err != nil {
		return err
	}
	defer conn.Close()

	resp, err := conn.Execute("save-all")
	if err != nil {
		return err
	}
	log.Info(resp)

	return nil
}

func (srv *ServerInstance) AddToWhiteList(players []string) {
	srv.rconMu.Lock()
	defer srv.rconMu.Unlock()

	conn, err := connectToRcon(srv)
	if err != nil {
		log.WithError(err).Error("Failed to connect rcon")
		return
	}
	defer conn.Close()

	for _, player := range players {
		resp, err := conn.Execute(fmt.Sprintf("whitelist add %s", player))
		if err != nil {
			log.WithError(err).Error("Failed to execute whitelist command")
		}
		log.Info(resp)
	}
}

func (srv *ServerInstance) AddToOp(players []string) {
	srv.rconMu.Lock()
	defer srv.rconMu.Unlock()

	conn, err := connectToRcon(srv)
	if err != nil {
		log.WithError(err).Error("Failed to connect rcon")
		return
	}
	defer conn.Close()

	for _, player := range players {
		resp, err := conn.Execute(fmt.Sprintf("op %s", player))
		if err != nil {
			log.WithError(err).Error("Failed to execute op command")
		}
		log.Info(resp)
	}
}

func (srv *ServerInstance) SendChat(message string) error {
	srv.rconMu.Lock()
	defer srv.rconMu.Unlock()
	conn, err := connectToRcon(srv)
	if err != nil {
		return err
	}
	defer conn.Close()

	resp, err := conn.Execute(fmt.Sprintf("tellraw @a \"%s\"", message))
	if err != nil {
		return err
	}
	log.Info(resp)

	return nil
}

func (srv *ServerInstance) GetSeed() (string, error) {
	srv.rconMu.Lock()
	defer srv.rconMu.Unlock()

	conn, err := connectToRcon(srv)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	seed, err := conn.Execute("seed")
	if err != nil {
		return "", err
	}

	if len(seed) < 8 || seed[:7] != "Seed: [" || seed[len(seed)-1] != ']' {
		return "", errors.New("Failed to retrieve seed")
	}

	return seed[7 : len(seed)-1], nil
}

func (srv *ServerInstance) QuickUndo() error {
	srv.quickUndoBeforeRestart = true

	proc, err := os.FindProcess(srv.ServerPid)
	if err != nil {
		return err
	}
	// go go go!!!
	if err := proc.Kill(); err != nil {
		return err
	}

	return nil
}

func LaunchInteractiveRcon() {
	conn, err := connectToRcon(nil)
	if err != nil {
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		resp, err := conn.Execute(scanner.Text())
		if err != nil {
			log.WithError(err).Error("Failed to execute command")
			os.Exit(1)
		}
		fmt.Println(resp)
		fmt.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Println("Failed to scan stdin")
	}
}

var (
	serverLoadingRegexp  = regexp.MustCompile("^Starting ([a-z]+\\.)+Main$")
	serverLoadedRegexp   = regexp.MustCompile("\\]: Done \\([0-9]*\\.[0-9]*s\\)! For help, type \"help\"")
	serverStoppingRegexp = regexp.MustCompile("\\]: Stopping the server")
)

func MonitorServer(ctx *config.PMCMContext, stdout io.ReadCloser) error {
	reader := bufio.NewReader(stdout)
	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil && err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if isPrefix {
			continue
		}
		fmt.Println(string(line))
		if serverLoadingRegexp.Match(line) {
			ctx.NotifyStatus(ctx.L("game.loading"))
		} else if serverLoadedRegexp.Match(line) {
			ctx.NotifyStatus(ctx.L("game.running"))

			if err := SaveLastServerVersion(ctx); err != nil {
				log.WithError(err).Error("Error saving last server versoin")
			}
		} else if serverStoppingRegexp.Match(line) {
			ctx.NotifyStatus(ctx.L("game.stopping"))
		}
	}
}

func signEulaForServer(ctx *config.PMCMContext) error {
	eulaFile, err := os.Create(ctx.LocateWorldData("eula.txt"))
	if err != nil {
		return err
	}
	defer eulaFile.Close()
	eulaFile.WriteString("eula=true")
	return nil
}

func processQuickUndo(ctx *config.PMCMContext) error {
	if err := os.RemoveAll(ctx.LocateWorldData("world")); err != nil {
		return err
	}

	cmd := exec.Command("cp", "-R", "--", "ss@quick0/world", "ss@quick0/world_nether", "ss@quick0/world_the_end", ".")
	cmd.Dir = ctx.LocateWorldData("")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.WithError(err).Info("cp command returned an error (this is no problem in the most cases)")
	}

	return nil
}

func LaunchServer(ctx *config.PMCMContext, srv *ServerInstance) error {
	if _, err := os.Stat(ctx.LocateServer(ctx.Cfg.Server.Version)); err != nil {
		return err
	}

	if ctx.Cfg.World.ShouldGenerate {
		if _, err := os.Stat(ctx.LocateWorldData("world")); err == nil {
			if err := os.RemoveAll(ctx.LocateWorldData("world")); err != nil {
				log.WithError(err).Error("Failed to remove world folder")
			}
		} else if !os.IsNotExist(err) {
			log.WithError(err).Error("Failed to stat world folder")
		}
		if _, err := os.Stat(ctx.LocateWorldData("world_nether")); err == nil {
			if err := os.RemoveAll(ctx.LocateWorldData("world_nether")); err != nil {
				log.WithError(err).Error("Failed to remove world_nether folder")
			}
		} else if !os.IsNotExist(err) {
			log.WithError(err).Error("Failed to stat world_nether folder")
		}
		if _, err := os.Stat(ctx.LocateWorldData("world_the_end")); err == nil {
			if err := os.RemoveAll(ctx.LocateWorldData("world_the_end")); err != nil {
				log.WithError(err).Error("Failed to remove world_the_end folder")
			}
		} else if !os.IsNotExist(err) {
			log.WithError(err).Error("Failed to stat world_the_end folder")
		}
	} else {
		if err := backup.ExtractWorldArchiveIfNeeded(ctx); err != nil {
			return err
		}
	}

	ver, exists, err := GetLastServerVersion(ctx)
	if err != nil {
		return err
	}
	if !exists || ver != ctx.Cfg.Server.Version {
		log.WithField("old", ver).WithField("new", ctx.Cfg.Server.Version).Info("Different version of server selected. cleaning up...")

		ents, err := os.ReadDir(ctx.LocateWorldData(""))
		if err != nil {
			return err
		}
		for _, ent := range ents {
			if ent.Name() == "server.properties" || ent.Name() == "world" || ent.Name() == "world_nether" || ent.Name() == "world_the_end" {
				continue
			}
			if err := os.RemoveAll(ctx.LocateWorldData(ent.Name())); err != nil {
				return err
			}
		}
	}

	if err := RemoveLastServerVersion(ctx); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := signEulaForServer(ctx); err != nil {
		return err
	}

	srv.Name = ctx.Cfg.Server.Version
	srv.FinishWG.Add(1)
	allocSize := ctx.Cfg.AllocSize
	if allocSize == 0 {
		allocSize = 512
	}
	javaArgs := []string{fmt.Sprintf("-Xmx%dM", ctx.Cfg.AllocSize), fmt.Sprintf("-Xms%dM", ctx.Cfg.AllocSize), "-jar", ctx.LocateServer(ctx.Cfg.Server.Version), "nogui"}
	go func() {
		log.WithField("server_name", ctx.Cfg.Server.Version).WithField("commandline", javaArgs).Info("Launching Minecraft server")
		launchCount := 0
		prevLaunch := time.Now()
		for !srv.ShouldStop && !srv.RestartRequested {
			if srv.quickUndoBeforeRestart {
				if err := processQuickUndo(ctx); err != nil {
					log.WithError(err).Error("Error processing quick undo")
				}

				launchCount = 0
				srv.quickUndoBeforeRestart = false
			}

			if launchCount == 5 {
				if time.Now().Sub(prevLaunch) < 3*time.Minute {
					srv.Crashed = true
					break
				}
			}
			cmd := exec.Command("java", javaArgs...)
			cmd.Dir = ctx.LocateWorldData("")
			cmdStdout, _ := cmd.StdoutPipe()
			cmd.Stderr = os.Stderr
			cmd.Start()
			srv.ServerPid = cmd.Process.Pid
			MonitorServer(ctx, cmdStdout)
			cmd.Wait()
			cmdStdout.Close()
			exitCode := cmd.ProcessState.ExitCode()
			log.WithField("exit_code", exitCode).Info("Server exited")
			if exitCode == 0 {
				break
			}
			launchCount++
		}
		srv.FinishWG.Done()
	}()

	return nil
}

func SaveLastServerVersion(ctx *config.PMCMContext) error {
	file, err := os.Create(ctx.LocateDataFile("last_version"))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(ctx.Cfg.Server.Version); err != nil {
		return err
	}
	return nil
}

func RemoveLastServerVersion(ctx *config.PMCMContext) error {
	if err := os.Remove(ctx.LocateDataFile("last_version")); err != nil {
		return err
	}
	return nil
}

func GetLastServerVersion(ctx *config.PMCMContext) (string, bool, error) {
	file, err := os.Open(ctx.LocateDataFile("last_version"))
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
