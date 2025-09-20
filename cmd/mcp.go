package main

import (
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xieyuschen/go-mcp-server/internal/tool"
)

const (
	mcpName    = "go-mcp-server"
	mcpVersion = "v0.0.1"
)

func main() {
	// Create a server with a single tool.
	server := mcp.NewServer(&mcp.Implementation{Name: mcpName, Version: mcpVersion}, nil)

	tool.RegisterTools(server)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})

	http.Handle("/", handler)
	port := ":8080"
	log.Printf("Starting %s at %s", mcpName, port)
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}
