package tool

//go:generate go run generate.go

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GenericTool[In, Out any] struct {
	Name        string
	Description string
	Handler     mcp.ToolHandlerFor[In, Out]
}

func (t GenericTool[In, Out]) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: t.Name, Description: t.Description}, t.Handler)
	log.Printf("Registered tool %s: %s", t.Name, t.Description)
}

func (t GenericTool[In, Out]) Details() (name, description string) {
	return t.Name, t.Description
}

var tools = []any{
	GenericTool[IEmpty, *OGoInfo]{
		Name:        "get_go_env",
		Description: "get go environment, including GOROOT, GOBIN, and Go version.",
		Handler:     getGoEnv,
	},
	// this mcp tool aims to provide all information for a go package under current project,
	// so it will affect by current project module setting up.
	// don't be confued with project_defined_packages. which report all packages in current project only(no 3rd packages).
	GenericTool[IGoProjectPackage, *OGoPackage]{
		Name:        "list_package_details",
		Description: "list a package details and all exported symbols of a given package used in current project",
		Handler:     listPackageDetails,
	},
	GenericTool[IEmpty, *OStdlibSymbols]{
		Name:        "list_stdlib_packages_symbols",
		Description: "reports all std packages of the global go with all exported symbol details.",
		Handler:     listStdlibSymbols,
	},
	GenericTool[IEmpty, *OStdlibSymbols]{
		Name:        "list_stdlib_packages",
		Description: "report all available std packages with their docs, if see all symbols inside it, use list_stdlib_packages_symbols",
		Handler:     listStdlibPackages,
	},
	GenericTool[IUsedModules, *OProjectUsedModules]{
		Name:        "fetch_project_build_required_modules",
		Description: "reports all modules used by the build in the current project with their versions, it respects all possible go mod directives like replace.",
		Handler:     listProjectUsedModules,
	},
	GenericTool[IUsedModules, *OProjectDefinedPackages]{
		Name:        "list_project_defined_packages",
		Description: "reports all packages defined by current project with their docs",
		Handler:     listProjectDefinedPackages,
	},
	GenericTool[IPackageInfo, *OCheck]{
		Name:        "check_package_exists",
		Description: "check if a package exists in current project module",
		Handler:     checkPackageExists,
	},
	GenericTool[ICheck, *OCheck]{
		Name:        "check_package_symbol_exists",
		Description: "check if a symbol exists in a given package in current project module",
		Handler:     checkSymbolExists,
	},
}

func UpdateReadme(content string) string {
	topics := strings.Split(content, "##")

	var builder strings.Builder
	builder.WriteString(" MCP Server Tools\n\n")
	for _, tool := range tools {
		type r interface {
			Details() (string, string)
		}
		if d, ok := tool.(r); ok {
			name, des := d.Details()
			builder.Write(fmt.Appendf(nil, "- `%s`: %s\n", name, des))
		}
	}
	builder.WriteString("\n")
	// by convention, the docs section is the 3rd one
	topics[2] = builder.String()
	return strings.Join(topics, "##")
}

func RegisterTools(server *mcp.Server) {
	for _, tool := range tools {
		type r interface {
			Register(server *mcp.Server)
		}
		if reg, ok := tool.(r); ok {
			reg.Register(server)
		}
	}
}

var reg = regexp.MustCompile("^go version (go[0-9.]+)")

func getGoEnv(_ context.Context, _ *mcp.CallToolRequest, input IEmpty) (*mcp.CallToolResult, *OGoInfo, error) {
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

	return nil, &OGoInfo{
		Version: goVersion,
		GoBin:   goBin,
		GOROOT:  goroot,
	}, nil
}

func listStdlibSymbols(_ context.Context, _ *mcp.CallToolRequest, input IEmpty) (*mcp.CallToolResult, *OStdlibSymbols, error) {
	stdLibMap, err := stdlib{}.listPackages(false)
	if err != nil {
		return nil, nil, err
	}
	return nil, &OStdlibSymbols{StdLibs: stdLibMap}, nil
}

func listStdlibPackages(_ context.Context, _ *mcp.CallToolRequest, input IEmpty) (*mcp.CallToolResult, *OStdlibSymbols, error) {
	stdLibMap, err := stdlib{}.listPackages(true)
	if err != nil {
		return nil, nil, err
	}
	return nil, &OStdlibSymbols{StdLibs: stdLibMap}, nil
}

func listProjectUsedModules(_ context.Context, _ *mcp.CallToolRequest, input IUsedModules) (*mcp.CallToolResult, *OProjectUsedModules, error) {
	analyzer, err := New(input.Cwd)
	if err != nil {
		return nil, nil, err
	}
	modules, err := analyzer.usedModules()
	if err != nil {
		return nil, nil, err
	}
	return nil, &OProjectUsedModules{Modules: modules}, nil
}

func listProjectDefinedPackages(_ context.Context, _ *mcp.CallToolRequest, input IUsedModules) (*mcp.CallToolResult, *OProjectDefinedPackages, error) {
	analyzer, err := New(input.Cwd)
	if err != nil {
		return nil, nil, err
	}
	packages, err := analyzer.projectDefinedPackages(true)
	if err != nil {
		return nil, nil, err
	}
	return nil, &OProjectDefinedPackages{Packages: packages}, nil
}

func listPackageDetails(_ context.Context, _ *mcp.CallToolRequest, input IGoProjectPackage) (*mcp.CallToolResult, *OGoPackage, error) {
	analyzer, err := New(input.Cwd)
	if err != nil {
		return nil, nil, err
	}
	info, err := analyzer.projectSinglePackage(input.PackagePath, false)
	if err != nil {
		return nil, nil, err
	}

	return nil, &OGoPackage{Package: *info}, nil
}

func checkPackageExists(_ context.Context, _ *mcp.CallToolRequest, input IPackageInfo) (*mcp.CallToolResult, *OCheck, error) {
	analyser, err := New(input.Cwd)
	if err != nil {
		return nil, nil, err
	}
	pkg, err := analyser.projectSinglePackage(input.PackagePath, false)
	if err != nil || pkg == nil {
		o := &OCheck{
			Validated: false,
			Explanation: fmt.Sprintf(`failed to find the existence of package '%s': %s.
It may be caused by the package doensn't exist in current module, 
or you want to load a package from another module but haven't added in go.mod file yet.`, input.PackagePath, err.Error()),
		}
		return nil, o, nil
	}

	o := &OCheck{
		Validated:   true,
		Explanation: fmt.Sprintf("package '%s' exists", input.PackagePath),
	}
	return nil, o, nil
}

func checkSymbolExists(_ context.Context, _ *mcp.CallToolRequest, input ICheck) (*mcp.CallToolResult, *OCheck, error) {
	analyser, err := New(input.Cwd)
	if err != nil {
		return nil, nil, err
	}
	pkg, err := analyser.projectSinglePackage(input.Path, true)
	if err != nil || pkg == nil {
		o := &OCheck{
			Validated: false,
			Explanation: fmt.Sprintf(`failed to find the existence of package '%s': %s.
It may be caused by the package doensn't exist in current module, 
or you want to load a package from another module but haven't added in go.mod file yet.`, input.Path, err.Error()),
		}
		return nil, o, nil
	}

	var found bool
	for _, sym := range pkg.Symbols {
		if sym.Name == input.Symbol {
			found = true
			break
		}
	}

	if !found {
		o := &OCheck{
			Validated:   false,
			Explanation: fmt.Sprintf("failed to find symbol '%s' in package '%s'", input.Symbol, input.Path),
		}
		return nil, o, nil
	}
	o := &OCheck{
		Validated: true,
	}
	return nil, o, nil
}
