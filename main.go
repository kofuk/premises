package main

import (
	"embed"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/kofuk/premises/backup"
	"github.com/kofuk/premises/config"
	"github.com/kofuk/premises/gameconfig"
	"github.com/kofuk/premises/mcversions"
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
	serverVersions   []mcversions.MCVersion
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
			go func(ch chan *monitor.StatusData) {
				defer func() {
					if err := recover(); err != nil {
						log.WithField("error", err).Error("Recovering previous error")
					}
				}()

				ch <- status
			}(ch)
		}
		s.monitorClientsMu.Unlock()
	}
}

func notifyNonRecoverableFailure() {
	server.monitorChan <- &monitor.StatusData{
		Status:   "Operation failed. Manual operation required!",
		HasError: true,
		Shutdown: true,
	}
}

func monitorServer(cfg *config.Config, gameServer GameServer) {
	server.monitorChan <- &monitor.StatusData{
		Status: "Waiting for the server...",
	}

	if err := monitor.MonitorServer(cfg, cfg.ServerAddr, server.monitorChan); err != nil {
		log.WithError(err).Error("Failed to monitor server")
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

	os.Remove(cfg.Locate("monitor_key"))

	gameServer.RevertDNS()

	server.monitorChan <- &monitor.StatusData{
		Status:   "Server stopped",
		Shutdown: true,
	}
}

func LaunchServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, gameServer GameServer, memSizeGB int) {
	if err := monitor.GenerateTLSKey(cfg); err != nil {
		log.WithError(err).Error("Failed to generate TLS key")
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to generate TLS key",
			HasError: true,
			Shutdown: true,
		}
		return
	}

	cfg.MonitorKey = gameConfig.AuthKey
	os.WriteFile(cfg.Locate("monitor_key"), []byte(gameConfig.AuthKey), 0600)

	server.monitorChan <- &monitor.StatusData{
		Status:   "Waiting for the server to start up...",
		HasError: false,
		Shutdown: false,
	}

	if !gameServer.SetUp(gameConfig, memSizeGB) {
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
		log.WithError(err).Error("Failed to request stopping server")
	}
}

func isValidMemSize(memSize int) bool {
	return memSize == 1 || memSize == 2 || memSize == 4 || memSize == 8 || memSize == 16 || memSize == 32 || memSize == 64
}

func createConfigFromPostData(values url.Values, cfg *config.Config) (*gameconfig.GameConfig, error) {
	if !values.Has("server-version") {
		return nil, errors.New("Server version is not set")
	}
	result := gameconfig.New()
	if err := result.SetServerVersion(values.Get("server-version")); err != nil {
		return nil, err
	}

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

	if values.Get("world-source") == "backups" {
		if !values.Has("world-name") {
			return nil, errors.New("World name is not set")
		} else if !values.Has("backup-generation") {
			return nil, errors.New("Backup generation is not set")
		}
		result.SetWorld(values.Get("world-name"), values.Get("backup-generation"))
	} else {
		//TODO: generate a new world
	}

	result.SetOperators(cfg.Game.Operators)
	result.SetWhitelist(cfg.Game.Whitelist)
	result.SetMegaCredential(cfg.Mega.Email, cfg.Mega.Password)
	result.SetMotd(cfg.Game.Motd)

	return result, nil
}

//go:embed templates/*.html
var templates embed.FS

//go:embed gen/*.js
var jsFiles embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// already checked by middleware
	CheckOrigin: func(*http.Request) bool { return true },
}

func guessAndHandleCurrentVMState(cfg *config.Config, gameServer GameServer) {
	if gameServer.VMExists() {
		if gameServer.VMRunning() {
			monitorKey, err := os.ReadFile(cfg.Locate("monitor_key"))
			if err != nil {
				log.WithError(err).Info("Failed to read previous monitor key")
				return
			}
			cfg.MonitorKey = string(monitorKey)

			if gameServer.ImageExists() {
				log.Info("Server seems to be running, but remote image exists")
				if !gameServer.DeleteImage() {
					log.Error("Failed to delete image")
				}
			}

			gameServer.UpdateDNS()

			log.Info("Start monitoring server")
			go monitorServer(cfg, gameServer)
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
	if err := godotenv.Load(); err != nil {
		log.WithError(err).Info("Failed to load .env file. If you want to use real envvars, you can ignore this diag safely.")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load config")
	}

	if cfg.Debug.Env {
		if err := os.Mkdir("/tmp/premises", 0755); err != nil {
			log.WithError(err).Info("Cannot create directory for debug environment")
		}
	}

	if cfg.Debug.Web {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	bindAddr := ":8000"
	if len(os.Args) > 1 {
		bindAddr = os.Args[1]
	}

	var gameServer GameServer
	if cfg.Debug.Runner {
		gameServer = NewLocalDebugServer(cfg)
	} else {
		gameServer = NewConohaServer(cfg)
	}

	server.status.Status = "Server stopped"
	server.status.Shutdown = true

	monitorChan := make(chan *monitor.StatusData)
	server.monitorChan = monitorChan

	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})

	template := template.New("")
	templateEntries, err := templates.ReadDir("templates")
	for _, ent := range templateEntries {
		data, err := templates.ReadFile(filepath.Join("templates", ent.Name()))
		if err != nil {
			log.WithError(err).Fatal("Failed to load templates")
		}
		template.New(ent.Name()).Parse(string(data))
	}
	r.SetHTMLTemplate(template)

	var sessionStore sessions.Store
	if cfg.Debug.Web {
		sessionStore = cookie.NewStore([]byte(cfg.ControlPanel.Secret))
	} else {
		sessionStore, err = redis.NewStore(4, "tcp", cfg.ControlPanel.Redis.Address, cfg.ControlPanel.Redis.Password, []byte(cfg.ControlPanel.Secret))
		if err != nil {
			log.WithError(err).Fatal("Failed to initialize Redis store")
		}
		redis.SetKeyPrefix(sessionStore, "session:")
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
			fields := strings.Split(usr, ":")
			if len(fields) != 2 {
				log.Error("Unexpected field count of controlPanel.users")
				continue
			}
			if fields[0] == username {
				hashPassword = fields[1]
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
	r.GET("/login.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript")

		var file io.Reader
		var err error
		if cfg.Debug.Web {
			file, err = os.Open("gen/login.js")
		} else {
			file, err = jsFiles.Open("gen/login.js")
		}
		if err != nil {
			log.WithError(err).Error("Failed to embedded file")
			c.Status(http.StatusInternalServerError)
			return
		}
		io.Copy(c.Writer, file)
	})
	r.GET("/control.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript")

		var file io.Reader
		var err error
		if cfg.Debug.Web {
			file, err = os.Open("gen/control.js")
		} else {
			file, err = jsFiles.Open("gen/control.js")
		}
		if err != nil {
			log.WithError(err).Error("Failed to embedded file")
			c.Status(http.StatusInternalServerError)
			return
		}
		io.Copy(c.Writer, file)
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
			if c.Request.Method == http.MethodPost || (c.Request.Method == http.MethodGet && c.GetHeader("Upgrade") == "WebSocket") {
				if c.GetHeader("Origin") == cfg.ControlPanel.AllowedOrigin {
					return
				}
				c.JSON(400, gin.H{"success": false, "message": "Invalid request (origin not allowed)"})
				c.Abort()
			}
		})
		{
			api.GET("/status", func(c *gin.Context) {
				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					log.WithError(err).Error("Failed to upgrade protocol to WebSocket")
					return
				}
				defer conn.Close()

				ch := make(chan *monitor.StatusData)
				server.addMonitorClient(ch)
				defer close(ch)
				defer server.removeMonitorClient(ch)

				server.statusMu.Lock()
				if err := conn.WriteJSON(server.status); err != nil {
					log.WithError(err).Error("Failed to write data")
					return
				}
				server.statusMu.Unlock()

				for {
					status := <-ch

					if err := conn.WriteJSON(status); err != nil {
						log.WithError(err).Error("Failed to write data")
						break
					}
				}
			})

			api.POST("/launch", func(c *gin.Context) {
				server.statusMu.Lock()
				defer server.statusMu.Unlock()

				if err := c.Request.ParseForm(); err != nil {
					log.WithError(err).Error("Failed to parse form")
					c.JSON(400, gin.H{"success": false, "message": "Form parse error"})
					return
				}

				gameConfig, err := createConfigFromPostData(c.Request.Form, cfg)
				if err != nil {
					c.JSON(400, gin.H{"success": false, "message": err.Error()})
					return
				}

				memSizeGB, _ := strconv.Atoi(strings.Replace(c.PostForm("machine-type"), "g", "", 1))

				go LaunchServer(gameConfig, cfg, gameServer, memSizeGB)

				c.JSON(200, gin.H{"success": true})
			})

			api.POST("/stop", func(c *gin.Context) {
				server.statusMu.Lock()
				defer server.statusMu.Unlock()

				go StopServer(cfg, gameServer)

				c.JSON(200, gin.H{"success": true})
			})

			api.GET("/backups", func(c *gin.Context) {
				if len(server.worldBackups) == 0 {
					backups, err := backup.GetBackupList(cfg)
					if err != nil {
						log.WithError(err).Error("Failed to retrive backup list")
						c.Status(http.StatusInternalServerError)
						return
					}
					server.worldBackupMu.Lock()
					server.worldBackups = backups
					server.worldBackupMu.Unlock()
				}
				c.JSON(http.StatusOK, server.worldBackups)
			})

			api.GET("/gameconfigs", func(c *gin.Context) {
				c.JSON(200, cfg.GetGameConfigs())
			})

			api.GET("/mcversions", func(c *gin.Context) {
				if len(server.serverVersions) == 0 {
					versions, err := mcversions.GetVersions()
					if err != nil {
						log.WithError(err).Error("Failed to retrive Minecraft versions")
						c.Status(http.StatusInternalServerError)
						return
					}
					server.serverVersions = versions
				}
				c.JSON(http.StatusOK, server.serverVersions)
			})
		}
	}

	go func() {
		server.dispatchMonitorEvent()
	}()

	guessAndHandleCurrentVMState(cfg, gameServer)

	log.Fatal(r.Run(bindAddr))
}
