package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	redisSess "github.com/gin-contrib/sessions/redis"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/language"

	"github.com/kofuk/premises/backup"
	"github.com/kofuk/premises/config"
	"github.com/kofuk/premises/gameconfig"
	"github.com/kofuk/premises/mcversions"
	"github.com/kofuk/premises/monitor"
)

//go:embed i18n/*.json
var i18nData embed.FS

var localizeBundle *i18n.Bundle

func L(locale, msgId string) string {
	if localizeBundle == nil {
		return msgId
	}

	localizer := i18n.NewLocalizer(localizeBundle, locale)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgId})
	if err != nil {
		log.WithError(err).Error("Error loading localized message. Fallback to \"en\"")
		localizer := i18n.NewLocalizer(localizeBundle, "en")
		msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgId})
		if err != nil {
			log.WithError(err).Error("Error loading localized message (fallback)")
			return msgId
		}
		return msg
	}
	return msg
}

func loadI18nData() error {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	ents, err := i18nData.ReadDir("i18n")
	if err != nil {
		return err
	}
	for _, ent := range ents {
		if _, err := bundle.LoadMessageFileFS(i18nData, "i18n/"+ent.Name()); err != nil {
			return err
		}
	}
	localizeBundle = bundle
	return nil
}

type serverState struct {
	statusMu         sync.Mutex
	status           monitor.StatusData
	selectedWorld    string
	monitorChan      chan *monitor.StatusData
	monitorClients   []chan *monitor.StatusData
	monitorClientsMu sync.Mutex
	machineType      string
}

var server serverState

const (
	CacheKeyBackups          = "backups"
	CacheKeyMCVersions       = "mcversions"
	CacheKeySystemInfoPrefix = "system-info"
)

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
			s.monitorClients = s.monitorClients[:len(s.monitorClients)-1]
			break
		}
	}
}

func (s *serverState) dispatchMonitorEvent(rdb *redis.Client) {
	for {
		status := <-s.monitorChan

		s.statusMu.Lock()
		s.status = *status
		s.statusMu.Unlock()

		if status.Shutdown {
			if _, err := rdb.Del(context.Background(), CacheKeyBackups).Result(); err != nil {
				log.WithError(err).Error("Failed to delete backup list cache")
			}
		}

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

func notifyNonRecoverableFailure(locale string) {
	server.monitorChan <- &monitor.StatusData{
		Status:   L(locale, "monitor.unrecoverable"),
		HasError: true,
		Shutdown: true,
	}
}

func monitorServer(cfg *config.Config, gameServer GameServer) {
	locale := cfg.ControlPanel.Locale

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "monitor.connecting"),
	}

	if err := monitor.MonitorServer(cfg, cfg.ServerAddr, server.monitorChan); err != nil {
		log.WithError(err).Error("Failed to monitor server")
	}

	if !gameServer.StopVM() {
		notifyNonRecoverableFailure(locale)
		return
	}
	if !gameServer.SaveImage() {
		notifyNonRecoverableFailure(locale)
		return
	}
	if !gameServer.DeleteVM() {
		notifyNonRecoverableFailure(locale)
		return
	}

	os.Remove(cfg.Locate("monitor_key"))

	gameServer.RevertDNS()

	server.monitorChan <- &monitor.StatusData{
		Status:   L(locale, "monitor.stopped"),
		Shutdown: true,
	}
}

func LaunchServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, gameServer GameServer, memSizeGB int) {
	locale := cfg.ControlPanel.Locale

	if err := monitor.GenerateTLSKey(cfg); err != nil {
		log.WithError(err).Error("Failed to generate TLS key")
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "monitor.tls_keygen.error"),
			HasError: true,
			Shutdown: true,
		}
		return
	}

	cfg.MonitorKey = gameConfig.AuthKey
	os.WriteFile(cfg.Locate("monitor_key"), []byte(gameConfig.AuthKey), 0600)

	server.monitorChan <- &monitor.StatusData{
		Status:   L(locale, "monitor.waiting"),
		HasError: false,
		Shutdown: false,
	}

	if !gameServer.SetUp(gameConfig, memSizeGB) {
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.start.error"),
			HasError: true,
			Shutdown: false,
		}
		return
	}

	if !gameServer.UpdateDNS() {
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.dns.error"),
			HasError: true,
			Shutdown: false,
		}
		return
	}

	if !gameServer.DeleteImage() {
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.image.delete.error"),
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

func ReconfigureServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, gameServer GameServer) {
	if err := monitor.ReconfigureServer(gameConfig, cfg, cfg.ServerAddr); err != nil {
		log.WithError(err).Error("Failed to reconfigure server")
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
		result.UseCache(values.Get("use-cache") == "true")
	} else {
		if !values.Has("world-name") {
			return nil, errors.New("World name is not set")
		}
		result.GenerateWorld(values.Get("world-name"), values.Get("seed"))
		if err := result.SetLevelType(values.Get("level-type")); err != nil {
			return nil, err
		}
	}

	result.SetOperators(cfg.Game.Operators)
	result.SetWhitelist(cfg.Game.Whitelist)
	result.SetMegaCredential(cfg.Mega.Email, cfg.Mega.Password)
	result.SetMotd(cfg.Game.Motd)
	result.SetLocale(cfg.ControlPanel.Locale)

	return result, nil
}

//go:embed templates/*.html
var templates embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// already checked by middleware
	CheckOrigin: func(*http.Request) bool { return true },
}

func guessAndHandleCurrentVMState(cfg *config.Config, gameServer GameServer) {
	locale := cfg.ControlPanel.Locale

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
				gameServer.DeleteImage()
			}

			gameServer.UpdateDNS()

			log.Info("Start monitoring server")
			go monitorServer(cfg, gameServer)
		} else {
			if !gameServer.ImageExists() && !gameServer.SaveImage() {
				notifyNonRecoverableFailure(locale)
				return
			}
			if !gameServer.DeleteVM() {
				notifyNonRecoverableFailure(locale)
				return
			}
		}
	}
}

func main() {
	log.SetReportCaller(true)
	if err := loadI18nData(); err != nil {
		log.Fatal(err)
	}

	if err := godotenv.Load(); err != nil {
		log.WithError(err).Info("Failed to load .env file. If you want to use real envvars, you can ignore this diag safely.")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load config")
	}

	if cfg.Debug.Env {
		if err := os.MkdirAll("/tmp/premises/gamedata", 0755); err != nil {
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

	server.status.Status = L(cfg.ControlPanel.Locale, "monitor.stopped")
	server.status.Shutdown = true

	monitorChan := make(chan *monitor.StatusData)
	server.monitorChan = monitorChan

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.ControlPanel.Redis.Address,
		Password: cfg.ControlPanel.Redis.Password,
	})

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

	sessionStore, err := redisSess.NewStore(4, "tcp", cfg.ControlPanel.Redis.Address, cfg.ControlPanel.Redis.Password, []byte(cfg.ControlPanel.Secret))
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize Redis store")
	}
	redisSess.SetKeyPrefix(sessionStore, "session:")
	r.Use(sessions.Sessions("session", sessionStore))

	r.NoRoute(static.Serve("/", static.LocalFile("gen", false)))

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

				closeChan := make(chan struct{})

				go func() {
					for {
						var v struct{}
						if err := conn.ReadJSON(&v); err != nil {
							log.Info("Connection closed")
							close(closeChan)
							break
						}
					}
				}()

				for {
					select {
					case status := <-ch:
						if err := conn.WriteJSON(status); err != nil {
							log.WithError(err).Error("Failed to write data")
							break
						}
					case <-closeChan:
						goto end
					}
				}
			end:
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

				machineType := c.PostForm("machine-type")
				server.machineType = machineType
				memSizeGB, _ := strconv.Atoi(strings.Replace(machineType, "g", "", 1))

				go LaunchServer(gameConfig, cfg, gameServer, memSizeGB)

				c.JSON(200, gin.H{"success": true})
			})

			api.POST("/reconfigure", func(c *gin.Context) {
				if err := c.Request.ParseForm(); err != nil {
					log.WithError(err).Error("Failed to parse form")
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Form parse error"})
					return
				}

				formValues := c.Request.Form
				formValues.Set("machine-type", server.machineType)

				gameConfig, err := createConfigFromPostData(formValues, cfg)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
					return
				}
				// Use previously generated key.
				gameConfig.AuthKey = cfg.MonitorKey

				go ReconfigureServer(gameConfig, cfg, gameServer)

				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			api.POST("/stop", func(c *gin.Context) {
				server.statusMu.Lock()
				defer server.statusMu.Unlock()

				go StopServer(cfg, gameServer)

				c.JSON(200, gin.H{"success": true})
			})

			api.GET("/backups", func(c *gin.Context) {
				if _, ok := c.GetQuery("reload"); ok {
					if _, err := rdb.Del(context.Background(), CacheKeyBackups).Result(); err != nil {
						log.WithError(err).Error("Failed to delete backup list cache")
					}
				}

				if val, err := rdb.Get(context.Background(), CacheKeyBackups).Result(); err == nil {
					c.Header("Content-Type", "application/json")
					c.Writer.Write([]byte(val))
					return
				} else if err != redis.Nil {
					log.WithError(err).Error("Error retriving mcversions cache")
				}

				log.WithField("cache_key", CacheKeyBackups).Info("cache miss")

				backups, err := backup.GetBackupList(cfg)
				if err != nil {
					log.WithError(err).Error("Failed to retrive backup list")
					c.Status(http.StatusInternalServerError)
					return
				}

				jsonData, err := json.Marshal(backups)
				if err != nil {
					log.WithError(err).Error("Failed to marshal backpu list")
					c.Status(http.StatusInternalServerError)
					return
				}

				if _, err := rdb.Set(context.Background(), CacheKeyBackups, jsonData, 24*time.Hour).Result(); err != nil {
					log.WithError(err).Error("Failed to store backup list")
				}

				c.Header("Content-Type", "application/json")
				c.Writer.Write(jsonData)
			})

			api.GET("/mcversions", func(c *gin.Context) {
				if _, ok := c.GetQuery("reload"); ok {
					if _, err := rdb.Del(context.Background(), CacheKeyMCVersions).Result(); err != nil {
						log.WithError(err).Error("Failed to delete mcversions cache")
					}
				}

				if val, err := rdb.Get(context.Background(), CacheKeyMCVersions).Result(); err == nil {
					c.Header("Content-Type", "application/json")
					c.Writer.Write([]byte(val))
					return
				} else if err != redis.Nil {
					log.WithError(err).Error("Error retriving mcversions cache")
				}

				log.WithField("cache_key", CacheKeyMCVersions).Info("cache miss")

				versions, err := mcversions.GetVersions()
				if err != nil {
					log.WithError(err).Error("Failed to retrive Minecraft versions")
					c.Status(http.StatusInternalServerError)
					return
				}

				jsonData, err := json.Marshal(versions)
				if err != nil {
					log.WithError(err).Error("Failed to marshal mcversions")
					c.Status(http.StatusInternalServerError)
					return
				}

				if _, err := rdb.Set(context.Background(), CacheKeyMCVersions, jsonData, 7*24*time.Hour).Result(); err != nil {
					log.WithError(err).Error("Failed to cache mcversions")
				}

				c.Header("Content-Type", "application/json")
				c.Writer.Write(jsonData)
			})

			api.GET("/systeminfo", func(c *gin.Context) {
				if cfg.ServerAddr == "" {
					c.Status(http.StatusTooEarly)
					return
				}

				cacheKey := fmt.Sprintf("%s:%s", CacheKeySystemInfoPrefix, cfg.ServerAddr)

				if _, ok := c.GetQuery("reload"); ok {
					if _, err := rdb.Del(context.Background(), cacheKey).Result(); err != nil {
						log.WithError(err).WithField("server_addr", cfg.ServerAddr).Error("Failed to delete system info cache")
					}
				}

				if val, err := rdb.Get(context.Background(), cacheKey).Result(); err == nil {
					c.Header("Content-Type", "application/json")
					c.Writer.Write([]byte(val))
					return
				} else if err != redis.Nil {
					log.WithError(err).WithField("server_addr", cfg.ServerAddr).Error("Error retriving system info cache")
				}

				log.WithField("cache_key", cacheKey).Info("cache miss")

				data, err := monitor.GetSystemInfoData(cfg, cfg.ServerAddr)
				if err != nil {
					c.Status(http.StatusInternalServerError)
					return
				}

				if _, err := rdb.Set(context.Background(), cacheKey, data, 24*time.Hour).Result(); err != nil {
					log.WithError(err).WithField("server_addr", cfg.ServerAddr).Error("Failed to cache mcversions")
				}

				c.Header("Content-Type", "application/json")
				c.Writer.Write(data)
			})

			api.GET("/worldinfo", func(c *gin.Context) {
				if cfg.ServerAddr == "" {
					c.Status(http.StatusTooEarly)
					return
				}

				data, err := monitor.GetWorldInfoData(cfg, cfg.ServerAddr)
				if err != nil {
					c.Status(http.StatusInternalServerError)
					return
				}

				c.Header("Content-Type", "application/json")
				c.Writer.Write(data)
			})

			api.POST("/snapshot", func(c *gin.Context) {
				if cfg.ServerAddr == "" {
					c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
					return
				}

				if err := monitor.TakeSnapshot(cfg, cfg.ServerAddr); err != nil {
					c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"success": true})
			})
		}
	}

	go func() {
		server.dispatchMonitorEvent(rdb)
	}()

	guessAndHandleCurrentVMState(cfg, gameServer)

	log.Fatal(r.Run(bindAddr))
}
