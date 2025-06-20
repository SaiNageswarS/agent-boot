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
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithRecovery(),
	)

	// define tool for each agent-boot tenant.
	healthTool := mcp.NewTool(
		"health_search",
		mcp.WithDescription("Searches medical literature and returns structured results with citation indices for proper source attribution. Returns JSON with document indices and sentences for precise citation."),
		mcp.WithString("input", // free‑form question
			mcp.Description("User's raw health question"),
			mcp.Required(),
		),
		mcp.WithArray("queries", // pre‑tokenised queries
			mcp.Items(map[string]any{"type": "string"}), // each element must be a string
			mcp.Description("Array of refined search queries derived from the input question"),
			mcp.Required(),
		),
	)

	// Medical reasoning prompt for analyzing health search results
	medicalReasoningPrompt := mcp.NewPrompt(
		"medical_reasoning",
		mcp.WithPromptDescription("Analyze health search results and provide reasoned medical responses with evidence assessment"),
		mcp.WithArgument("user_question",
			mcp.ArgumentDescription("The original health question asked by the user"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("search_results",
			mcp.ArgumentDescription("JSON results from the health search tool containing citations and evidence"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("search_status",
			mcp.ArgumentDescription("Status of the search: 'success', 'no_results', or 'partial_results'"),
			mcp.RequiredArgument(),
		),
	)

	healthSearchHandler := handlers.ProvideHealthSearchHandler()
	medicalReasoningPromptHandler := handlers.HandleMedicalReasoningPrompt

	s.AddTool(healthTool, healthSearchHandler.Handle)
	s.AddPrompt(medicalReasoningPrompt, medicalReasoningPromptHandler)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Failed to serve MCP: %v", err)
	}
}
