package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/strider2038/multica-mcp/internal/app"
	"github.com/strider2038/multica-mcp/internal/config"
	"github.com/strider2038/multica-mcp/internal/logging"
	mcpserver "github.com/strider2038/multica-mcp/internal/mcp"
	"github.com/strider2038/multica-mcp/internal/middleware"
	"github.com/strider2038/multica-mcp/internal/multica"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	logging.Setup(cfg.LogLevel)
	slog.Info("starting multica-mcp-server",
		"transport", cfg.MCPTransport,
		"read_only", cfg.ReadOnly,
	)

	client := multica.NewClient(cfg.MulticaBaseURL, cfg.MulticaToken)

	wsID := os.Getenv("MULTICA_WORKSPACE_ID")
	if wsID == "" {
		wsID = resolveWorkspace(client)
	}
	if wsID == "" {
		fmt.Fprintf(os.Stderr, "MULTICA_WORKSPACE_ID is required or a single workspace must exist\n")
		os.Exit(1)
	}

	useCase := app.NewUseCase(client, wsID, cfg.ReadOnly)
	mcpSrv := mcpserver.NewServer(useCase, cfg.ReadOnly)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	switch cfg.MCPTransport {
	case "http":
		runHTTP(mcpSrv.GetMCPServer(), cfg.HTTPPort, cfg.MCPAPIKey, ctx)
	default:
		runStdio(mcpSrv.GetMCPServer())
	}
}

func resolveWorkspace(client *multica.Client) string {
	workspaces, err := client.ListWorkspaces(context.Background())
	if err != nil {
		slog.Warn("failed to list workspaces for auto-detection", "error", err)
		return ""
	}
	if len(workspaces) == 1 {
		slog.Info("auto-detected single workspace", "workspace_id", workspaces[0].ID, "name", workspaces[0].Name)
		return workspaces[0].ID
	}
	if len(workspaces) > 1 {
		fmt.Fprintf(os.Stderr, "multiple workspaces found; set MULTICA_WORKSPACE_ID to one of:\n")
		for _, ws := range workspaces {
			fmt.Fprintf(os.Stderr, "  %s (%s)\n", ws.ID, ws.Name)
		}
	}
	return ""
}

func runStdio(mcpServer *server.MCPServer) {
	stdioServer := server.NewStdioServer(mcpServer)
	if err := stdioServer.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		slog.Error("stdio server error", "error", err)
		os.Exit(1)
	}
}

func runHTTP(mcpServer *server.MCPServer, port int, apiKey string, ctx context.Context) {
	streamable := server.NewStreamableHTTPServer(mcpServer)

	handler := http.Handler(streamable)
	if apiKey != "" {
		throttle := middleware.NewLoginThrottle(5, 1*time.Minute, 15*time.Minute)
		handler = throttle.Wrap(apiKey, handler)
		slog.Info("API key authentication enabled", "max_failures", 5, "ban_duration", "15m")
	}

	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	slog.Info("starting HTTP MCP server", "addr", addr)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5)
	defer cancel()
	httpServer.Shutdown(shutdownCtx)
}
