package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xieyuschen/go-mcp-server/internal/tool"
)

const (
	mcpName    = "go-mcp-server"
	mcpVersion = "v0.0.1"
)

var port int

func init() {
	flag.IntVar(&port, "port", 8555, "Port to listen on")
	flag.Parse()
}

func main() {
	// Create a server with a single tool.
	server := mcp.NewServer(&mcp.Implementation{Name: mcpName, Version: mcpVersion}, nil)

	tool.RegisterTools(server)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})

	http.Handle("/", handler)
	log.Printf("Starting %s at %d", mcpName, port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		panic(err)
	}
}
