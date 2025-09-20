package tool

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/go/packages"
)

const (
	toolNameGoInfo  = "go_info"
	toolNameStdLibs = "stdlib_symbols"

	descGoInfo  = "reports the Go environment, including GOROOT, GOBIN, and Go version."
	descStdLibs = "reports all std packages of the global go with all exported symbol details."
)

func RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: toolNameGoInfo, Description: descGoInfo}, goEnv)
	log.Printf("Registered tool: %s", toolNameGoInfo)
	mcp.AddTool(server, &mcp.Tool{Name: toolNameStdLibs, Description: descStdLibs}, stdLib)
	log.Printf("Registered tool: %s", toolNameStdLibs)
}

type IEmpty struct{}

type OGo struct {
	Version string `json:"version" jsonschema:"the Go version"`
	GoBin   string `json:"gobin" jsonschema:"the GOBIN environment variable"`
	GOROOT  string `json:"goroot" jsonschema:"the GOROOT environment variable"`
}

var reg = regexp.MustCompile("^go version (go[0-9.]+)")

func goEnv(_ context.Context, request *mcp.CallToolRequest, input IEmpty) (*mcp.CallToolResult, *OGo, error) {
	out, err := gobin()
	if err != nil {
		return nil, nil, err
	}
	goBin := strings.TrimSpace(string(out))
	out, err = goroot(goBin)
	if err != nil {
		return nil, nil, err
	}

	goroot := strings.TrimSpace(string(out))
	if err != nil {
		return nil, nil, err
	}
	goVersion, err := goVersion(goBin)
	if err != nil {
		return nil, nil, err
	}

	return nil, &OGo{
		Version: goVersion,
		GoBin:   goBin,
		GOROOT:  goroot,
	}, nil
}

func stdLib(_ context.Context, req *mcp.CallToolRequest, input IEmpty) (*mcp.CallToolResult, *OGoStdLib, error) {
	cfg := packages.Config{
		Mode: packages.LoadAllSyntax,
	}
	pkgs, err := packages.Load(&cfg, "std")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load Go standard library packages: %w", err)
	}
	stdLibMap := make(map[string]Stdlib, len(pkgs))
	for _, pkg := range pkgs {
		if strings.HasPrefix(pkg.PkgPath, "internal/") {
			continue
		}
		l := analyzePkg(pkg)
		stdLibMap[pkg.PkgPath] = l
	}

	return nil, &OGoStdLib{StdLibs: stdLibMap}, nil
}

func analyzePkg(pkg *packages.Package) Stdlib {
	symbols := []Symbol{}
	var pkgDoc string
	for _, file := range pkg.Syntax {
		if file.Doc != nil {
			pkgDoc += file.Doc.Text() + "\n"
		}
		tok := pkg.Fset.File(file.FileStart)
		for _, _decl := range file.Decls {
			switch decl := _decl.(type) {
			case *ast.FuncDecl:
				if decl.Name.Name == "_" || !ast.IsExported(decl.Name.Name) {
					continue
				}

				sym := funcSymbol(decl)
				sym.Doc = decl.Doc.Text()
				symbols = append(symbols, sym)
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if spec.Name.Name == "_" || !ast.IsExported(spec.Name.Name) {
							continue
						}
						sym := typeSymbol(tok, spec)
						sym.Doc = decl.Doc.Text()
						symbols = append(symbols, sym)
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if name.Name == "_" || !ast.IsExported(name.Name) {
								continue
							}
							vs := varSymbol(tok, spec, name, decl.Tok == token.CONST)
							vs.Doc = decl.Doc.Text()
							symbols = append(symbols, vs)
						}
					}
				}
			}
		}
	}

	return Stdlib{
		Path:    pkg.PkgPath,
		Docs:    strings.TrimSpace(pkgDoc),
		Symbols: symbols,
	}
}
