package main

import (
	"crypto/subtle"
	"log"
	"net/http"
	"time"

	docker "github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/kofuk/premises/ostack-fake/dockerstack"
	"github.com/kofuk/premises/ostack-fake/entity"
)

type Ostack struct {
	r             *gin.Engine
	docker        *docker.Client
	tenantId      string
	user          string
	password      string
	token         string
	deletedImages map[string]bool
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

func (self *Ostack) ServeGetToken(c *gin.Context) {
	var req entity.GetTokenReq
	if err := c.ShouldBind(&req); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	user := req.Auth.PasswordCredentials.UserName
	password := req.Auth.PasswordCredentials.Password
	tenantId := req.Auth.TenantID

	if user != self.user || password != self.password || tenantId != self.tenantId {
		c.JSON(http.StatusUnauthorized, nil)
		return
	}

	resp := entity.GetTokenResp{}
	resp.Access.Token.Expires = time.Now().Add(30 * time.Minute).Format(time.RFC3339)
	resp.Access.Token.Id = self.token

	c.JSON(http.StatusOK, resp)
}

func (self *Ostack) ServeGetServerDetail(c *gin.Context) {
	servers, err := dockerstack.GetServerDetail(c.Request.Context(), self.docker)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	c.JSON(http.StatusOK, servers)
}

func (self *Ostack) ServeLaunchServer(c *gin.Context) {
	var req entity.LaunchServerReq
	if err := c.ShouldBind(&req); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	server, err := dockerstack.LaunchServer(c.Request.Context(), self.docker, req)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	c.JSON(http.StatusAccepted, server)
}

func (self *Ostack) ServeServerAction(c *gin.Context) {
	serverId := c.Param("server")

	var req entity.ServerActionReq
	if err := c.ShouldBind(&req); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	if req.CreateImage != nil {
		if err := dockerstack.CreateImage(c.Request.Context(), self.docker, serverId, req.CreateImage.Name); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
	} else {
		if err := dockerstack.StopServer(c.Request.Context(), self.docker, serverId); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
	}

	c.JSON(http.StatusAccepted, nil)
}

func (self *Ostack) ServeDeleteServer(c *gin.Context) {
	serverId := c.Param("server")

	if err := dockerstack.DeleteServer(c.Request.Context(), self.docker, serverId); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (self *Ostack) ServeGetFlavors(c *gin.Context) {
	resp := entity.FlavorsResp{
		Flavors: []entity.Flavor{
			{
				ID:   "10921063-8e6a-4c96-b72d-bf6f7bfe4a2b",
				Name: "g-c3m2d100",
			},
			{
				ID:   "791bda46-b944-499c-affe-c04ba73cb341",
				Name: "g-c4m4d100",
			},
			{
				ID:   "fce5765d-f2bd-447d-9851-0fe695902984",
				Name: "g-c6m8d100",
			},
			{
				ID:   "680f6515-b903-4d8c-895f-006ef040600e",
				Name: "g-c8m16d100",
			},
			{
				ID:   "8b376d12-eb83-4922-9423-6aba0f326aba",
				Name: "g-c12m32d100",
			},
			{
				ID:   "0f5756a5-6e0e-47f3-859d-fd46aacb8694",
				Name: "g-c24m64d100",
			},
		},
	}
	c.JSON(http.StatusOK, resp)
}

func (self *Ostack) ServeGetImages(c *gin.Context) {
	images, err := dockerstack.GetImages(c.Request.Context(), self.docker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	visibleImage := make([]entity.Image, 0)
	for _, image := range images.Images {
		if !self.deletedImages[image.ID] {
			visibleImage = append(visibleImage, image)
		}
	}
	images.Images = visibleImage

	c.JSON(http.StatusOK, images)
}

func (self *Ostack) ServeDeleteImages(c *gin.Context) {
	imageId := c.Param("image")
	// No one can delete image of running container.
	// We save removed image ID and emulate Open Stack behavior.
	self.deletedImages[imageId] = true
	c.JSON(http.StatusNoContent, nil)
}

func (self *Ostack) setupRoutes() {
	self.r.POST("/identity/v2.0/tokens", self.ServeGetToken)

	needsAuthEndpoint := self.r.Group("", func(c *gin.Context) {
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Auth-Token")), []byte(self.token)) != 1 {
			c.JSON(http.StatusUnauthorized, nil)
			c.Abort()
		}
	})

	needsAuthEndpoint.GET("/image/v2/images", self.ServeGetImages)
	needsAuthEndpoint.DELETE("/image/v2/images/:image", self.ServeDeleteImages)
	needsAuthEndpoint.POST("/compute/v2/servers", self.ServeLaunchServer)
	needsAuthEndpoint.GET("/compute/v2/servers/detail", self.ServeGetServerDetail)
	needsAuthEndpoint.POST("/compute/v2/servers/:server/action", self.ServeServerAction)
	needsAuthEndpoint.DELETE("/compute/v2/servers/:server", self.ServeDeleteServer)
	needsAuthEndpoint.GET("/compute/v2/flavors", self.ServeGetFlavors)
}

func NewOstack(options ...OstackOption) (*Ostack, error) {
	docker, err := docker.NewClientWithOpts()
	if err != nil {
		return nil, err
	}

	ostack := &Ostack{
		r:             gin.Default(),
		docker:        docker,
		deletedImages: make(map[string]bool),
	}

	ostack.setupRoutes()

	for _, opt := range options {
		opt(ostack)
	}

	return ostack, nil
}

func (self *Ostack) Start() error {
	return self.r.Run("127.0.0.1:8010")
}
