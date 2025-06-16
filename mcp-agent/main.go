package main

import (
	"agent-boot/mcp-agent/handlers"
	"log"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	godotenv.Load()

	s := server.NewMCPServer(
		"agent-boot-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// define tool for each agent-boot tenant.
	healthTool := mcp.NewTool(
		"health_search",
		mcp.WithDescription("Accepts a health question and an array of refined queries and returns journal insights"),
		mcp.WithString("input", // free‑form question
			mcp.Description("User's raw health question"),
			mcp.Required(),
		),
		mcp.WithArray("queries", // pre‑tokenised queries
			mcp.Items(map[string]any{"type": "string"}), // each element must be a string
			mcp.Description("Array of refined search queries"),
			mcp.Required(),
		),
	)

	healthSearchHandler := handlers.ProvideHealthSearchHandler()
	s.AddTool(healthTool, healthSearchHandler.Handle)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Failed to serve MCP: %v", err)
	}
}
