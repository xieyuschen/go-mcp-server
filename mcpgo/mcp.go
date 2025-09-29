package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xieyuschen/go-mcp-server/internal/tool"
)

const (
	mcpName    = "go-mcp-server"
	mcpVersion = "v0.0.2"
)

var (
	addr    = flag.String("addr", "", "Address to listen on")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
	cwd     = flag.String("cwd", "", "Set the current working directory (default is the process's current directory)")
)

func main() {
	flag.Parse()
	log.SetOutput(io.Discard)
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		// write to stdout rather than stderr otherwise vscode plugin complains.
		log.SetOutput(os.Stdout)
	}

	if *cwd == "" {
		if dir, err := os.Getwd(); err == nil {
			*cwd = dir
		}
	}

	// Create a server with a single tool.
	server := mcp.NewServer(&mcp.Implementation{Name: mcpName, Version: mcpVersion}, nil)
	// todo: integrate cwd into mcp server later.
	tool.RegisterTools(server)

	if *addr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
			return server
		}, &mcp.StreamableHTTPOptions{JSONResponse: true})

		http.Handle("/", handler)
		log.Printf("Starting %s at %s", mcpName, *addr)
		if err := http.ListenAndServe(*addr, nil); err != nil {
			panic(err)
		}
		return
	}

	log.Printf("Starting %s in stdio mode", mcpName)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		panic(err)
	}
}
