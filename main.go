package main

import (
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"

	"github.com/kofuk/premises/backup"
	"github.com/kofuk/premises/config"
	"github.com/kofuk/premises/conoha"
	"github.com/kofuk/premises/gameconfig"
	"github.com/kofuk/premises/monitor"
)

type serverState struct {
	statusMu         sync.Mutex
	status           monitor.StatusData
	selectedWorld    string
	monitorChan      chan *monitor.StatusData
	monitorClients   []chan *monitor.StatusData
	monitorClientsMu sync.Mutex
	worldBackupMu    sync.Mutex
	worldBackups     []backup.WorldBackup
}

var server serverState

func (s *serverState) addMonitorClient(ch chan *monitor.StatusData) {
	s.monitorClientsMu.Lock()
	defer s.monitorClientsMu.Unlock()

	s.monitorClients = append(s.monitorClients, ch)
}

func (s *serverState) removeMonitorClient(ch chan *monitor.StatusData) {
	s.monitorClientsMu.Lock()
	defer s.monitorClientsMu.Unlock()

	for i, c := range s.monitorClients {
		if c == ch {
			if i != len(s.monitorClients)-1 {
				s.monitorClients[i] = s.monitorClients[len(s.monitorClients)-1]
			}
			break
		}
	}
}

func (s *serverState) dispatchMonitorEvent() {
	for {
		status := <-s.monitorChan

		s.statusMu.Lock()
		s.status = *status
		s.statusMu.Unlock()

		s.monitorClientsMu.Lock()
		for _, ch := range s.monitorClients {
			ch <- status
		}
		s.monitorClientsMu.Unlock()
	}
}

func BuildVM(gameConfig []byte, cfg *config.Config) error {
	token, err := conoha.GetToken(cfg)
	if err != nil {
		return err
	}

	flavors, err := conoha.GetFlavors(cfg, token)
	if err != nil {
		return err
	}
	flavorID := flavors.GetIDByCondition(2, 1, 100)

	imageID, imageStatus, err := conoha.GetImageID(cfg, token, "mc-premises")
	if err != nil {
		return err
	} else if imageStatus != "active" {
		return errors.New("Image is not active")
	}

	startupScript, err := conoha.GenerateStartupScript(gameConfig, cfg)
	if err != nil {
		return err
	}

	if _, err := conoha.CreateVM(cfg, token, imageID, flavorID, startupScript); err != nil {
		return err
	}

	if err := conoha.DeleteImage(cfg, token, imageID); err != nil {
		return err
	}

	return nil
}

func DestroyVM(cfg *config.Config) error {
	token, err := conoha.GetToken(cfg)
	if err != nil {
		return err
	}

	detail, err := conoha.GetVMDetail(cfg, token, "mc-premises")
	if err != nil {
		return err
	}

	if err := conoha.StopVM(cfg, token, detail.ID); err != nil {
		return err
	}

	// Wait for VM to stop
	for {
		time.Sleep(20 * time.Second)

		detail, err := conoha.GetVMDetail(cfg, token, "mc-premises")
		if err != nil {
			return err
		}
		if detail.Status == "SHUTOFF" {
			break
		}
	}

	if err := conoha.CreateImage(cfg, token, detail.ID, "mc-premises"); err != nil {
		return err
	}

	// Wait for image to be saved
	for {
		if _, imageStatus, err := conoha.GetImageID(cfg, token, "mc-premises"); err == nil && imageStatus == "active" {
			break
		}
		time.Sleep(30 * time.Second)
	}

	if err := conoha.DeleteVM(cfg, token, detail.ID); err != nil {
		return err
	}

	return nil
}

func notifyNonRecoverableFailure() {
	server.monitorChan <- &monitor.StatusData{
		Status:   "Operation failed. Manual operation required!",
		HasError: true,
		Shutdown: true,
	}
}

func monitorServer(cfg *config.Config, gameServer GameServer) {
	if err := monitor.MonitorServer(cfg, cfg.ServerAddr, server.monitorChan); err != nil {
		log.Println(err)
	}

	if !gameServer.StopVM() {
		notifyNonRecoverableFailure()
		return
	}
	if !gameServer.SaveImage() {
		notifyNonRecoverableFailure()
		return
	}
	if !gameServer.DeleteVM() {
		notifyNonRecoverableFailure()
		return
	}

	os.Remove(filepath.Join(cfg.Prefix, "/opt/premises/monitor_key"))
}

func LaunchServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, gameServer GameServer) {
	cfg.MonitorKey = gameConfig.AuthKey
	os.WriteFile(filepath.Join(cfg.Prefix, "/opt/premises/monitor_key"), []byte(gameConfig.AuthKey), 0600)

	server.monitorChan <- &monitor.StatusData{
		Status:   "Waiting for the server to start up...",
		HasError: false,
		Shutdown: false,
	}

	if !gameServer.SetUp(gameConfig) {
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to start VM",
			HasError: true,
			Shutdown: false,
		}
		return
	}

	if !gameServer.UpdateDNS() {
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to update DNS",
			HasError: true,
			Shutdown: false,
		}
		return
	}

	if !gameServer.DeleteImage() {
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to delete outdated image",
			HasError: true,
			Shutdown: false,
		}

		return
	}

	go monitorServer(cfg, gameServer)
}

func StopServer(cfg *config.Config, gameServer GameServer) {
	if err := monitor.StopServer(cfg, cfg.ServerAddr); err != nil {
		log.Println(err)
	}
}

func isValidMemSize(memSize int) bool {
	return memSize == 1 || memSize == 2 || memSize == 4 || memSize == 8 || memSize == 16 || memSize == 32 || memSize == 64
}

func createConfigFromPostData(values url.Values, cfg *config.Config) (*gameconfig.GameConfig, error) {
	if !values.Has("game-config") {
		return nil, errors.New("Game configuration is not set")
	}
	result := gameconfig.New(values.Get("game-config"))

	if !values.Has("machine-type") {
		return nil, errors.New("Machine type is not set")
	}
	memSizeGB, err := strconv.Atoi(strings.Replace(values.Get("machine-type"), "g", "", 1))
	if err != nil {
		return nil, err
	}
	if !isValidMemSize(memSizeGB) {
		return nil, errors.New("Invalid machine type")
	}
	result.SetAllocFromAvailableMemSize(memSizeGB * 1024)
	result.GenerateAuthKey()

	if values.Get("new-world") == "on" {
		if err := result.SetLevelType(values.Get("level-type")); err != nil {
			return nil, err
		}

		if !values.Has("world") {
			return nil, errors.New("World is not set")
		}
		result.SetWorld(values.Get("world"), 0)
	} else if values.Get("migrate-world") == "on" {
		if !values.Has("migrate-from") {
			return nil, errors.New("Migratation source world is not set")
		}
		var config backup.WorldBackup
		if json.Unmarshal([]byte(values.Get("migrate-from")), &config) != nil {
			return nil, errors.New("Invalid migratation source world")
		}
		result.MigrateFromOtherConfig(config.ServerName, config.WorldName, config.Generation)
	} else {
		if !values.Has("world-gen") {
			return nil, errors.New("World generation is not set")
		}
		worldGen, err := strconv.Atoi(values.Get("world-gen"))
		if err != nil {
			return nil, errors.New("Invalid world generation")
		}
		result.SetWorld(values.Get("world"), worldGen)
	}

	result.SetOperators(cfg.Game.Operators)
	result.SetWhitelist(cfg.Game.Whitelist)
	result.SetMegaCredential(cfg.Mega.Email, cfg.Mega.Password)
	result.SetMotd(cfg.Game.Motd)

	return result, nil
}

//go:embed templates/*.html
var templates embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func guessAndHandleCurrentVMState(cfg *config.Config, gameServer GameServer) {
	if gameServer.VMExists() {
		if gameServer.VMRunning() {
			go monitorServer(cfg, gameServer)
			if gameServer.ImageExists() {
				if !gameServer.DeleteImage() {
					log.Println("Failed to delete image")
				}
			}
		} else {
			if !gameServer.ImageExists() && !gameServer.SaveImage() {
				notifyNonRecoverableFailure()
				return
			}
			if !gameServer.DeleteVM() {
				notifyNonRecoverableFailure()
				return
			}
		}
	}
}

func main() {
	debugEnv := false
	debugWeb := false

	if len(os.Getenv("PREMISES_DEBUG")) > 0 {
		for _, mod := range strings.Split(os.Getenv("PREMISES_DEBUG"), ",") {
			if mod == "all" {
				debugEnv = true
				debugWeb = true
			} else if mod == "env" {
				debugEnv = true
			} else if mod == "web" {
				debugWeb = true
			}
		}
	}

	prefix := ""
	if debugEnv {
		prefix = "/tmp/premises"
	}

	if debugWeb {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	bindAddr := ":8000"
	if len(os.Args) > 1 {
		bindAddr = os.Args[1]
	}

	cfg, err := config.LoadConfig(prefix)
	if err != nil {
		log.Fatal(err)
	}
	cfg.Prefix = prefix

	go func() {
		backups, err := backup.GetBackupList(cfg)
		if err != nil {
			log.Println(err)
			return
		}

		server.worldBackupMu.Lock()
		defer server.worldBackupMu.Unlock()
		server.worldBackups = backups
	}()

	gameServer := NewLocalDebugServer(cfg)

	server.status.Status = "Server stopped"
	server.status.Shutdown = true

	guessAndHandleCurrentVMState(cfg, gameServer)

	monitorChan := make(chan *monitor.StatusData)
	server.monitorChan = monitorChan

	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})

	template := template.New("")
	templateEntries, err := templates.ReadDir("templates")
	for _, ent := range templateEntries {
		data, err := templates.ReadFile(filepath.Join("templates", ent.Name()))
		if err != nil {
			log.Fatal(err)
		}
		template.New(ent.Name()).Parse(string(data))
	}
	r.SetHTMLTemplate(template)

	var sessionStore sessions.Store
	if debugWeb {
		sessionStore = cookie.NewStore([]byte(cfg.ControlPanel.Secret))
	} else {
		sessionStore, err = redis.NewStore(4, "tcp", cfg.ControlPanel.Redis.Address, cfg.ControlPanel.Redis.Password, []byte(cfg.ControlPanel.Secret))
		if err != nil {
			log.Fatal(err)
		}
	}

	r.Use(sessions.Sessions("session", sessionStore))

	r.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("username") != nil {
			c.Redirect(http.StatusFound, "/control")
		} else {
			c.HTML(200, "login.html", nil)
		}
	})
	r.POST("/", func(c *gin.Context) {
		if c.GetHeader("Origin") != cfg.ControlPanel.AllowedOrigin {
			c.Redirect(http.StatusFound, "/")
			return
		}

		username := c.PostForm("username")
		password := c.PostForm("password")

		hashPassword := ""
		for _, usr := range cfg.ControlPanel.Users {
			if usr.Name == username {
				hashPassword = usr.Password
				break
			}
		}
		if hashPassword == "" {
			c.Redirect(http.StatusFound, "/")
			return
		}

		session := sessions.Default(c)
		if bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(password)) == nil {
			session.Set("username", username)
			session.Save()
			c.Redirect(http.StatusFound, "/control")
			return
		}
	})
	r.GET("/logout", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Delete("username")
		session.Save()
		c.Redirect(http.StatusFound, "/")
	})

	controlPanel := r.Group("control")
	controlPanel.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("username") == nil {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
		}
	})
	{
		controlPanel.GET("/", func(c *gin.Context) {
			c.HTML(200, "control.html", nil)
		})

		api := controlPanel.Group("api")
		api.Use(func(c *gin.Context) {
			if c.GetHeader("Origin") != cfg.ControlPanel.AllowedOrigin {
				c.JSON(400, gin.H{"success": false, "message": "Invalid request (origin not allowed)"})
				c.Abort()
			}
		})
		{
			api.GET("/status", func(c *gin.Context) {
				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					log.Println(err)
					return
				}
				defer conn.Close()

				ch := make(chan *monitor.StatusData)
				server.addMonitorClient(ch)
				defer close(ch)
				defer server.removeMonitorClient(ch)

				server.statusMu.Lock()
				if err := conn.WriteJSON(server.status); err != nil {
					log.Println(err)
					return
				}
				server.statusMu.Unlock()

				for {
					status := <-ch

					if err := conn.WriteJSON(status); err != nil {
						log.Println(err)
						break
					}
				}
			})

			api.POST("/launch", func(c *gin.Context) {
				server.statusMu.Lock()
				defer server.statusMu.Unlock()

				if err := c.Request.ParseForm(); err != nil {
					log.Println(err)
					c.JSON(400, gin.H{"success": false, "message": "Form parse error"})
					return
				}

				gameConfig, err := createConfigFromPostData(c.Request.Form, cfg)
				if err != nil {
					c.JSON(400, gin.H{"success": false, "message": err.Error()})
					return
				}

				j, _ := json.Marshal(gameConfig)
				log.Println(string(j))

				go LaunchServer(gameConfig, cfg, gameServer)

				c.JSON(200, gin.H{"success": true})
			})

			api.POST("/stop", func(c *gin.Context) {
				server.statusMu.Lock()
				defer server.statusMu.Unlock()

				go StopServer(cfg, gameServer)

				c.JSON(200, gin.H{"success": true})
			})

			api.POST("/getbackups", func(c *gin.Context) {
				server.worldBackupMu.Lock()
				defer server.worldBackupMu.Unlock()

				c.JSON(200, server.worldBackups)
			})

			api.POST("/getgameconfigs", func(c *gin.Context) {
				c.JSON(200, cfg.Game.Configs)
			})
		}
	}

	go func() {
		server.dispatchMonitorEvent()
	}()

	log.Fatal(r.Run(bindAddr))

	// zoneID, err := cloudflare.GetZoneID(cfg)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// if err := cloudflare.UpdateDNS(cfg, zoneID, "2001:db8::2", 6); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := monitor.GenerateTLSKey(cfg); err != nil {
	// 	log.Fatal(err)
	// }

	// ss, err := conoha.GenerateStartupScript([]byte("hoge"), cfg)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println(ss)

	// if err := BuildVM(cfg); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := DestroyVM(cfg); err != nil {
	// 	log.Fatal(err)
	// }
}
