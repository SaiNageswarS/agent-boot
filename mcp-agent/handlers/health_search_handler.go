package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	pb "agent-boot/proto/generated"
	"agent-boot/proto/generated/generatedconnect"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"
)

type HealthSearchHandler struct {
	client generatedconnect.SearchClient
}

func ProvideHealthSearchHandler() *HealthSearchHandler {
	searchCoreUrl := os.Getenv("SEARCH_CORE_URL")
	if searchCoreUrl == "" {
		panic("SEARCH_CORE_URL environment variable is not set")
	}

	client := generatedconnect.NewSearchClient(
		http.DefaultClient,
		searchCoreUrl,
		connect.WithGRPCWeb())

	return &HealthSearchHandler{client: client}
}

func (s *HealthSearchHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	queries := req.GetStringSlice("queries", nil)

	if len(queries) == 0 {
		return mcp.NewToolResultError("No queries provided"), nil
	}

	log.Printf("Received queries: %v", queries)
	authToken := os.Getenv("SEARCH_CORE_AUTH_TOKEN")
	if authToken == "" {
		log.Println("SEARCH_CORE_AUTH_TOKEN environment variable is not set")
		return mcp.NewToolResultError("SEARCH_CORE_AUTH_TOKEN environment variable is not set"), nil
	}

	searchReq := connect.NewRequest(&pb.SearchRequest{
		Queries: queries,
	})
	searchReq.Header().Set("Authorization", "Bearer "+authToken)

	resp, err := s.client.Search(ctx, searchReq)

	if err != nil {
		log.Printf("Search request failed: %v", err)
		return mcp.NewToolResultError("Search request failed: " + err.Error()), nil
	}

	jsonResponse, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal search response: %v", err)
		return mcp.NewToolResultError("Failed to marshal search response: " + err.Error()), nil
	}

	log.Printf("Search response: %s", jsonResponse)
	return mcp.NewToolResultText(string(jsonResponse)), nil
}
