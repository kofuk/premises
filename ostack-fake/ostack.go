package main

import (
	"crypto/subtle"
	_ "embed"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	docker "github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/kofuk/premises/ostack-fake/dockerstack"
	"github.com/kofuk/premises/ostack-fake/entity"
	"github.com/labstack/echo/v4"
)

//go:embed flavors.json
var flavorData []byte

type Ostack struct {
	r             *echo.Echo
	m             sync.Mutex
	docker        *docker.Client
	tenantId      string
	user          string
	password      string
	token         string
	deletedImages map[string]bool
	secGroups     map[string]entity.SecurityGroup
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

func (self *Ostack) ServeGetToken(c echo.Context) error {
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

	if user != self.user || password != self.password || tenantId != self.tenantId {
		return c.JSON(http.StatusUnauthorized, nil)
	}

	resp := entity.GetTokenResp{}
	resp.Token.ExpiresAt = time.Now().Add(30 * time.Minute).Format(time.RFC3339)

	c.Response().Header().Add("x-subject-token", self.token)

	return c.JSON(http.StatusCreated, resp)
}

func (self *Ostack) ServeGetServerDetails(c echo.Context) error {
	servers, err := dockerstack.GetServerDetails(c.Request().Context(), self.docker)
	if err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, servers)
}

func (self *Ostack) ServeGetServerDetail(c echo.Context) error {
	servers, err := dockerstack.GetServerDetail(c.Request().Context(), self.docker, c.Param("id"))
	if err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, servers)
}

func (self *Ostack) ServeLaunchServer(c echo.Context) error {
	self.m.Lock()
	defer self.m.Unlock()

	var req entity.LaunchServerReq
	if err := c.Bind(&req); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusBadRequest, nil)
	}

outer:
	for _, rsg := range req.Server.SecurityGroups {
		for _, sg := range self.secGroups {
			if rsg.Name == sg.Name {
				continue outer
			}

			slog.Error("Unknown security group")
			return c.JSON(http.StatusBadRequest, nil)
		}
	}

	server, err := dockerstack.LaunchServer(c.Request().Context(), self.docker, req)
	if err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusAccepted, server)
}

func (self *Ostack) ServeServerAction(c echo.Context) error {
	serverId := c.Param("server")

	if err := dockerstack.StopServer(c.Request().Context(), self.docker, serverId); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusAccepted, nil)
}

func (self *Ostack) ServeDeleteServer(c echo.Context) error {
	serverId := c.Param("server")

	if err := dockerstack.DeleteServer(c.Request().Context(), self.docker, serverId); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (self *Ostack) ServeGetFlavors(c echo.Context) error {
	c.Response().Header().Add("Content-Type", "application/json")
	c.Response().Writer.Write(flavorData)
	return nil
}

func (self *Ostack) ServeGetImages(c echo.Context) error {
	self.m.Lock()
	defer self.m.Unlock()

	images, err := dockerstack.GetImages(c.Request().Context(), self.docker)
	if err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	visibleImage := make([]entity.Image, 0)
	for _, image := range images.Images {
		if !self.deletedImages[image.ID] {
			visibleImage = append(visibleImage, image)
		}
	}
	images.Images = visibleImage

	return c.JSON(http.StatusOK, images)
}

func (self *Ostack) ServeDeleteImages(c echo.Context) error {
	self.m.Lock()
	defer self.m.Unlock()

	imageId := c.Param("image")
	// No one can delete image of running container.
	// We save removed image ID and emulate Open Stack behavior.
	self.deletedImages[imageId] = true
	return c.JSON(http.StatusNoContent, nil)
}

func (self *Ostack) ServeGetSecurityGroups(c echo.Context) error {
	self.m.Lock()
	defer self.m.Unlock()

	var result entity.SecurityGroupResp
	result.SecurityGroups = make([]entity.SecurityGroup, 0)
	for _, sg := range self.secGroups {
		result.SecurityGroups = append(result.SecurityGroups, sg)
	}
	return c.JSON(http.StatusOK, result)
}

func (self *Ostack) ServeCreateSecurityGroup(c echo.Context) error {
	self.m.Lock()
	defer self.m.Unlock()

	var req entity.SecurityGroupReq
	if err := c.Bind(&req); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusBadRequest, nil)
	}

	id := uuid.NewString()

	sg := req.SecurityGroup
	sg.ID = id
	sg.SecurityGroupRules = append(sg.SecurityGroupRules, entity.SecurityGroupRule{
		SecurityGroupID: id,
		Direction:       "egress",
		EtherType:       "IPv4",
		PortRangeMin:    nil,
		PortRangeMax:    nil,
		Protocol:        nil,
	})
	sg.SecurityGroupRules = append(sg.SecurityGroupRules, entity.SecurityGroupRule{
		SecurityGroupID: id,
		Direction:       "egress",
		EtherType:       "IPv6",
		PortRangeMin:    nil,
		PortRangeMax:    nil,
		Protocol:        nil,
	})

	self.secGroups[id] = sg

	return c.JSON(http.StatusCreated, entity.SecurityGroupReq{
		SecurityGroup: sg,
	})
}

func (self *Ostack) ServeCreateSecurityGroupRule(c echo.Context) error {
	self.m.Lock()
	defer self.m.Unlock()

	var req entity.SecurityGroupRuleReq
	if err := c.Bind(&req); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusBadRequest, nil)
	}

	sg, ok := self.secGroups[req.SecurityGroupRule.SecurityGroupID]
	if !ok {
		slog.Error("Security group not found")
		return c.JSON(http.StatusNotFound, nil)
	}
	sg.SecurityGroupRules = append(sg.SecurityGroupRules, req.SecurityGroupRule)
	self.secGroups[req.SecurityGroupRule.SecurityGroupID] = sg

	return c.JSON(http.StatusCreated, req)
}

func (self *Ostack) ServeVolumeAction(c echo.Context) error {
	volumeId := c.Param("volume")
	if !strings.HasPrefix(volumeId, "volume_") {
		slog.Error("Invalid volume", slog.String("volume_id", volumeId))
		return c.JSON(http.StatusNotFound, nil)
	}

	var req entity.VolumeActionReq
	if err := c.Bind(&req); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusBadRequest, nil)
	}

	if err := dockerstack.CreateImage(c.Request().Context(), self.docker, strings.TrimPrefix(volumeId, "volume_"), req.UploadImage.ImageName); err != nil {
		slog.Error(err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	// OpenStack actually returnes status of the image, but we don't emulate it because Premises don't check the response body.
	return c.JSON(http.StatusAccepted, nil)
}

func (self *Ostack) setupRoutes() {
	self.r.POST("/identity/v3/auth/tokens", self.ServeGetToken)

	needsAuthEndpoint := self.r.Group("", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if subtle.ConstantTimeCompare([]byte(c.Request().Header.Get("X-Auth-Token")), []byte(self.token)) != 1 {
				return c.JSON(http.StatusUnauthorized, nil)
			}
			return next(c)
		}
	})

	needsAuthEndpoint.GET("/image/v2/images", self.ServeGetImages)
	needsAuthEndpoint.DELETE("/image/v2/images/:image", self.ServeDeleteImages)
	needsAuthEndpoint.POST("/compute/v2.1/servers", self.ServeLaunchServer)
	needsAuthEndpoint.GET("/compute/v2.1/servers/detail", self.ServeGetServerDetails)
	needsAuthEndpoint.GET("/compute/v2.1/servers/:id", self.ServeGetServerDetail)
	needsAuthEndpoint.POST("/compute/v2.1/servers/:server/action", self.ServeServerAction)
	needsAuthEndpoint.DELETE("/compute/v2.1/servers/:server", self.ServeDeleteServer)
	needsAuthEndpoint.GET("/compute/v2.1/flavors/detail", self.ServeGetFlavors)
	needsAuthEndpoint.GET("/network/v2.0/security-groups", self.ServeGetSecurityGroups)
	needsAuthEndpoint.POST("/network/v2.0/security-groups", self.ServeCreateSecurityGroup)
	needsAuthEndpoint.POST("/network/v2.0/security-group-rules", self.ServeCreateSecurityGroupRule)
	needsAuthEndpoint.POST("/volume/v3/volumes/:volume/action", self.ServeVolumeAction)
}

func NewOstack(options ...OstackOption) (*Ostack, error) {
	docker, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		return nil, err
	}

	ostack := &Ostack{
		r:             echo.New(),
		docker:        docker,
		deletedImages: make(map[string]bool),
		secGroups:     make(map[string]entity.SecurityGroup),
	}

	ostack.setupRoutes()

	for _, opt := range options {
		opt(ostack)
	}

	return ostack, nil
}

func (self *Ostack) Start() error {
	return self.r.Start("127.0.0.1:8010")
}
