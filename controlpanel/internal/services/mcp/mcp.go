package mcp

import (
	"net/http"

	"github.com/kofuk/premises/controlpanel/internal/auth"
	"github.com/kofuk/premises/controlpanel/internal/world"
	"github.com/kofuk/premises/internal"
	"github.com/mark3labs/mcp-go/mcp"
	mcpServer "github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

type MCPServer struct {
	redis *redis.Client
	db    *bun.DB
	world *world.WorldService
	auth  *auth.AuthService
}

func NewMCPServer(redis *redis.Client, db *bun.DB, world *world.WorldService, auth *auth.AuthService) *MCPServer {
	return &MCPServer{
		redis: redis,
		db:    db,
		world: world,
		auth:  auth,
	}
}

func (s *MCPServer) registerTools(server *mcpServer.MCPServer) {
	toolHandler := &ToolHandler{
		server: s,
	}
	server.AddTool(
		mcp.NewTool("list_hardware_configurations",
			mcp.WithDescription("List hardware configurations"),
		),
		toolHandler.ListHardwareConfigurations,
	)
	server.AddTool(
		mcp.NewTool("list_existing_worlds",
			mcp.WithDescription("List existing worlds"),
		),
		toolHandler.ListExistingWorlds,
	)
}

func (s *MCPServer) Start() error {
	server := mcpServer.NewMCPServer("Premises - a web-based Minecraft server launcher", internal.Version)

	s.registerTools(server)

	sseServer := mcpServer.NewSSEServer(server,
		mcpServer.WithBasePath("/mcp"),
	)
	http.HandleFunc("/mcp/", func(w http.ResponseWriter, r *http.Request) {
		token, err := s.auth.GetFromRequest(r.Context(), r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !token.HasScope(auth.ScopeAdmin) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		sseServer.ServeHTTP(w, r)
	})
	return http.ListenAndServe(":10001", nil)
}
