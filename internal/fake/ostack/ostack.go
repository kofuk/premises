package ostack

import (
	"context"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	docker "github.com/docker/docker/client"
	"github.com/kofuk/premises/internal/fake/ostack/dockerstack"
	"github.com/kofuk/premises/internal/fake/ostack/entity"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

//go:embed flavors.json
var flavorData []byte
var listFlavorsResp entity.ListFlavorsResp

func init() {
	if err := json.Unmarshal(flavorData, &listFlavorsResp); err != nil {
		log.Fatal(err)
	}
}

type Ostack struct {
	r           *echo.Echo
	m           sync.Mutex
	docker      *docker.Client
	tenantId    string
	user        string
	password    string
	token       string
	volumeNames map[string]string
}

type OstackOption func(ostack *Ostack)

func TenantCredentials(tenantId, user, password string) OstackOption {
	return func(ostack *Ostack) {
		ostack.tenantId = tenantId
		ostack.user = user
		ostack.password = password
	}
}

func Token(token string) OstackOption {
	return func(ostack *Ostack) {
		ostack.token = token
	}
}

func (o *Ostack) ServeGetHealth(c *echo.Context) error {
	ver, err := o.docker.ServerVersion(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ver)
}

func (o *Ostack) ServeGetToken(c *echo.Context) error {
	var req entity.GetTokenReq
	if err := c.Bind(&req); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusBadRequest, nil)
	}

	if len(req.Auth.Identity.Methods) != 1 || req.Auth.Identity.Methods[0] != "password" {
		slog.Error("Unsupported identity method", slog.Any("specified_methods", req.Auth.Identity.Methods))
		return c.JSON(http.StatusBadRequest, nil)
	}

	user := req.Auth.Identity.Password.User.Name
	password := req.Auth.Identity.Password.User.Password
	tenantId := req.Auth.Scope.Project.ID

	if user != o.user || password != o.password || tenantId != o.tenantId {
		return c.JSON(http.StatusUnauthorized, nil)
	}

	resp := entity.GetTokenResp{}
	resp.Token.ExpiresAt = time.Now().Add(30 * time.Minute).Format(time.RFC3339)

	c.Response().Header().Add("x-subject-token", o.token)

	return c.JSON(http.StatusCreated, resp)
}

func (o *Ostack) ServeListServerDetails(c *echo.Context) error {
	servers, err := dockerstack.ListServerDetails(c.Request().Context(), o.docker)
	if err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, servers)
}

func (o *Ostack) ServeGetServerDetail(c *echo.Context) error {
	servers, err := dockerstack.GetServerDetail(c.Request().Context(), o.docker, c.Param("id"))
	if err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, servers)
}

func (o *Ostack) ServeCreateServer(c *echo.Context) error {
	o.m.Lock()
	defer o.m.Unlock()

	var req entity.LaunchServerReq
	if err := c.Bind(&req); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusBadRequest, nil)
	}

	// Validate flavorRef
	flavorFound := false
	for _, flavor := range listFlavorsResp.Flavors {
		if flavor.ID == req.Server.FlavorRef {
			if flavor.Disabled || !flavor.Public || !strings.HasPrefix(flavor.Name, "g2l-t-") {
				// Unavailable flavor
				return c.JSON(http.StatusBadRequest, nil)
			}

			flavorFound = true
			break
		}
	}
	if !flavorFound {
		// Unknown flavor
		return c.JSON(http.StatusBadRequest, nil)
	}

	server, err := dockerstack.LaunchServer(c.Request().Context(), o.docker, req)
	if err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusAccepted, server)
}

func (o *Ostack) ServeServerAction(c *echo.Context) error {
	serverId := c.Param("server")

	if err := dockerstack.StopServer(c.Request().Context(), o.docker, serverId); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	imageName, ok := o.volumeNames[serverId]
	if !ok {
		servers, err := dockerstack.ListServerDetails(c.Request().Context(), o.docker)
		if err != nil {
			slog.Error(err.Error())
			return c.JSON(http.StatusInternalServerError, nil)
		}

		for _, s := range servers.Servers {
			if s.ID == serverId {
				imageName = s.Metadata.InstanceNameTag
			}
		}
	}

	slog.Debug("Creating image",
		slog.String("image_name", imageName),
		slog.String("volume_id", serverId),
	)

	if err := dockerstack.CreateImage(c.Request().Context(), o.docker, serverId, imageName); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusAccepted, nil)
}

func (o *Ostack) ServeDeleteServer(c *echo.Context) error {
	serverId := c.Param("server")

	if err := dockerstack.DeleteServer(c.Request().Context(), o.docker, serverId); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}
	return c.String(http.StatusNoContent, "")
}

func (o *Ostack) ServeListFlavors(c *echo.Context) error {
	c.Response().Header().Add("Content-Type", "application/json")
	c.Response().Write(flavorData)
	return nil
}

func (o *Ostack) ServeListVolumes(c *echo.Context) error {
	o.m.Lock()
	defer o.m.Unlock()

	volumes, err := dockerstack.ListVolumes(c.Request().Context(), o.docker)
	if err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	for i := 0; i < len(volumes.Volumes); i++ {
		if name, ok := o.volumeNames[volumes.Volumes[i].ID]; ok {
			volumes.Volumes[i].Name = name
		}
	}

	slog.Debug("list volumes", slog.Any("volumes", volumes.Volumes))

	return c.JSON(http.StatusOK, volumes)
}

func (o *Ostack) ServeUpdateVolume(c *echo.Context) error {
	o.m.Lock()
	defer o.m.Unlock()

	var req entity.UpdateVolumeReq
	if err := c.Bind(&req); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusBadRequest, nil)
	}

	volumeId := c.Param("volume")
	volumeId = strings.TrimPrefix(volumeId, "volume_")

	slog.Debug("Saving image names", slog.String("volume_id", volumeId), slog.String("name", req.Volume.Name))

	o.volumeNames[volumeId] = req.Volume.Name

	return c.JSON(http.StatusOK, nil)
}

func (o *Ostack) setupRoutes() {
	o.r.POST("/identity/v3/auth/tokens", o.ServeGetToken)
	o.r.GET("/health", o.ServeGetHealth)

	needsAuthEndpoint := o.r.Group("", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if subtle.ConstantTimeCompare([]byte(c.Request().Header.Get("X-Auth-Token")), []byte(o.token)) != 1 {
				return c.JSON(http.StatusUnauthorized, nil)
			}
			return next(c)
		}
	})

	needsAuthEndpoint.POST("/compute/v2.1/servers", o.ServeCreateServer)
	needsAuthEndpoint.GET("/compute/v2.1/servers/detail", o.ServeListServerDetails)
	needsAuthEndpoint.GET("/compute/v2.1/servers/:id", o.ServeGetServerDetail)
	needsAuthEndpoint.POST("/compute/v2.1/servers/:server/action", o.ServeServerAction)
	needsAuthEndpoint.DELETE("/compute/v2.1/servers/:server", o.ServeDeleteServer)
	needsAuthEndpoint.GET("/compute/v2.1/flavors/detail", o.ServeListFlavors)
	needsAuthEndpoint.GET("/volume/v3/volumes", o.ServeListVolumes)
	needsAuthEndpoint.PUT("/volume/v3/volumes/:volume", o.ServeUpdateVolume)
}

func NewOstack(options ...OstackOption) (*Ostack, error) {
	docker, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		return nil, err
	}

	engine := echo.New()
	engine.Use(middleware.RequestLogger())

	origErrHandler := echo.DefaultHTTPErrorHandler(true)
	engine.HTTPErrorHandler = func(c *echo.Context, err error) {
		if err == echo.ErrNotFound {
			c.JSON(http.StatusNotFound, map[string]any{
				"code":  http.StatusNotFound,
				"error": "Specified resource not found or not implemented",
			})
		} else {
			origErrHandler(c, err)
		}
	}

	ostack := &Ostack{
		r:           engine,
		docker:      docker,
		volumeNames: make(map[string]string),
	}

	ostack.setupRoutes()

	for _, opt := range options {
		opt(ostack)
	}

	return ostack, nil
}

func (o *Ostack) Start(ctx context.Context) error {
	sc := echo.StartConfig{
		Address:    "0.0.0.0:8010",
		HideBanner: true,
	}
	return sc.Start(ctx, o.r)
}
