package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"

	"chronoscoper.com/premises/backup"
	"chronoscoper.com/premises/config"
	"chronoscoper.com/premises/conoha"
	"chronoscoper.com/premises/gameconfig"
	"chronoscoper.com/premises/monitor"
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

	imageID, err := conoha.GetImageID(cfg, token, "mc-premises")
	if err != nil {
		return err
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
		detail, err := conoha.GetVMDetail(cfg, token, "mc-premises")
		if err != nil {
			return err
		}
		if detail.Status == "SHUTOFF" {
			break
		}
		time.Sleep(30 * time.Second)
	}

	if err := conoha.CreateImage(cfg, token, detail.ID, "mc-premises"); err != nil {
		return err
	}

	// Wait for image to be saved
	for {
		if _, err := conoha.GetImageID(cfg, token, "mc-premises"); err == nil {
			break
		}
		time.Sleep(30 * time.Second)
	}

	if err := conoha.DeleteVM(cfg, token, detail.ID); err != nil {
		return err
	}

	return nil
}

func LaunchServer(gameConfig *gameconfig.GameConfig, cfg *config.Config) {
	//TODO: temporary
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = filepath.Join(os.Getenv("HOME"), "source/premises-mcmanager")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	server.monitorChan <- &monitor.StatusData{
		Status:   "Waiting for the server to start up...",
		HasError: false,
		Shutdown: false,
	}

	go func() {
		if err := monitor.MonitorServer(cfg, cfg.ServerAddr, server.monitorChan); err != nil {
			log.Println(err)
		}
	}()

	if err := cmd.Run(); err != nil {
		log.Println(err)
	}
}

func StopServer(cfg *config.Config) {
	monitor.StopServer(cfg, cfg.ServerAddr)
}

//go:embed templates/*.html
var templates embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	debugMode := false
	prefix := ""
	if len(os.Getenv("PREMISES_DEBUG")) > 0 {
		debugMode = true
		prefix = "/tmp/premises"
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

	//===================
	// temporary implementation for testing
	cfg.MonitorKey = "hoge"
	cfg.ServerAddr = "localhost"
	//===================

	server.status.Status = "Server is shutdown"
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
			log.Fatal(err)
		}
		template.New(ent.Name()).Parse(string(data))
	}
	r.SetHTMLTemplate(template)

	var sessionStore sessions.Store
	if debugMode {
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

				gameConfig := gameconfig.New()
				go LaunchServer(gameConfig, cfg)

				c.JSON(200, gin.H{"success": true})
			})

			api.POST("/stop", func(c *gin.Context) {
				server.statusMu.Lock()
				defer server.statusMu.Unlock()

				go StopServer(cfg)

				c.JSON(200, gin.H{"success": true})
			})

			api.POST("/getbackups", func(c *gin.Context) {
				server.worldBackupMu.Lock()
				defer server.worldBackupMu.Unlock()

				c.JSON(200, server.worldBackups)
			})

			api.POST("/getgameconfigs", func(c *gin.Context) {
				c.JSON(200, cfg.GameConfigs)
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
