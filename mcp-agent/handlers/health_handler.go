package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	pb "agent-boot/proto/generated"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type HealthSearchHandler struct {
	client pb.SearchClient
}

func ProvideHealthSearchHandler() *HealthSearchHandler {
	searchCoreUrl := os.Getenv("SEARCH_CORE_URL")
	if searchCoreUrl == "" {
		panic("SEARCH_CORE_URL environment variable is not set")
	}

	conn, err := grpc.NewClient(searchCoreUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic("Failed to connect to search core: " + err.Error())
	}

	client := pb.NewSearchClient(conn)
	return &HealthSearchHandler{client: client}
}

func (s *HealthSearchHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	queries := req.GetStringSlice("queries", nil)

	if len(queries) == 0 {
		return mcp.NewToolResultError("No queries provided"), nil
	}

	authToken := os.Getenv("SEARCH_CORE_AUTH_TOKEN")
	if authToken == "" {
		log.Println("SEARCH_CORE_AUTH_TOKEN environment variable is not set")
		return mcp.NewToolResultError("SEARCH_CORE_AUTH_TOKEN environment variable is not set"), nil
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+authToken)
	resp, err := s.client.Search(ctx, &pb.SearchRequest{
		Queries: queries,
	})

	if err != nil {
		log.Printf("Search request failed: %v", err)
		return mcp.NewToolResultError("Search request failed: " + err.Error()), nil
	}

	groundingData := formatResults(resp.Chunks)
	return mcp.NewToolResultText(groundingData), nil
}

func formatResults(rs []*pb.Chunk) string {
	if len(rs) == 0 {
		return "No relevant journal articles found."
	}

	var b strings.Builder
	// Main passages with footnote markers
	for i, r := range rs {
		footnote := fmt.Sprintf("[^%d]", i+1)
		fmt.Fprintf(&b, "%s%s\n\n", r.Body, footnote)
	}

	// Sources section
	b.WriteString("### Sources\n")
	for i, r := range rs {
		fmt.Fprintf(&b, "[^%d]: %s\n", i+1, r.Citation)
	}

	return b.String()
}
