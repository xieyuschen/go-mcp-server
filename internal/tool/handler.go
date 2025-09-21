package tool

//go:generate go run generate.go

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/go/packages"
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
		Name:        "go_info",
		Description: "reports the Go environment, including GOROOT, GOBIN, and Go version.",
		Handler:     goEnv,
	},
	// this mcp tool aims to provide all information for a go package under current project,
	// so it will affect by current project module setting up.
	// don't be confued with project_defined_packages. which report all packages in current project only(no 3rd packages).
	GenericTool[IGoProjectPackage, *OGoPackage]{
		Name:        "go_package_symbols",
		Description: "report all exported symbols of a given content for current project",
		Handler:     goProjectPackages,
	},
	GenericTool[IEmpty, *OStdlibSymbols]{
		Name:        "go_std_packages_symbols",
		Description: "reports all std packages of the global go with all exported symbol details.",
		Handler:     stdlibs,
	},
	GenericTool[IEmpty, *OStdlibSymbols]{
		Name:        "go_std_packages",
		Description: "report all available std packages with their docs, if see all symbols inside it, use go_std_packages_symbols",
		Handler:     stdlibList,
	},
	GenericTool[IUsedModules, *OProjectUsedModules]{
		Name:        "project_used_modules",
		Description: "reports all modules used in the current project with their versions, it respects all possible go mod directives like replace.",
		Handler:     usedModules,
	},
	GenericTool[IUsedModules, *OProjectDefinedPackages]{
		Name:        "project_defined_packages",
		Description: "reports all pacakges defined by current project with their docs",
		Handler:     projectPackages,
	},
}

func UpdateReadme(content string) string {
	topics := strings.Split(content, "##")

	var builder strings.Builder
	builder.WriteString(" Feature\n\n")
	for _, tool := range tools {
		type r interface {
			Details() (string, string)
		}
		if d, ok := tool.(r); ok {
			name, des := d.Details()
			builder.Write(fmt.Appendf(nil, "- [x] %s: %s\n", name, des))
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

func goEnv(_ context.Context, _ *mcp.CallToolRequest, input IEmpty) (*mcp.CallToolResult, *OGoInfo, error) {
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

func stdlibs(_ context.Context, _ *mcp.CallToolRequest, input IEmpty) (*mcp.CallToolResult, *OStdlibSymbols, error) {
	cfg := packages.Config{
		Mode: packages.LoadAllSyntax,
	}
	pkgs, err := packages.Load(&cfg, "std")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load Go standard library packages: %w", err)
	}
	stdLibMap := make(map[string]Package, len(pkgs))
	for _, pkg := range pkgs {
		if strings.HasPrefix(pkg.PkgPath, "internal/") {
			continue
		}
		stdLibMap[pkg.PkgPath] = getPackageInfo(pkg, false)
	}

	return nil, &OStdlibSymbols{StdLibs: stdLibMap}, nil
}

func stdlibList(_ context.Context, _ *mcp.CallToolRequest, input IEmpty) (*mcp.CallToolResult, *OStdlibSymbols, error) {
	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedFiles,
	}
	pkgs, err := packages.Load(&cfg, "std")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load Go standard library packages: %w", err)
	}
	stdLibMap := make(map[string]Package, len(pkgs))
	for _, pkg := range pkgs {
		if strings.HasPrefix(pkg.PkgPath, "internal/") {
			continue
		}

		stdLibMap[pkg.PkgPath] = getPackageInfo(pkg, true)
	}

	return nil, &OStdlibSymbols{StdLibs: stdLibMap}, nil
}

func usedModules(_ context.Context, _ *mcp.CallToolRequest, input IUsedModules) (*mcp.CallToolResult, *OProjectUsedModules, error) {
	cwd := filepath.Clean(strings.TrimSpace(input.Cwd))
	if cwd == "" {
		return nil, nil, fmt.Errorf("input 'Cwd' (current working directory) cannot be empty")
	}

	cmd := exec.Command("go", "mod", "download", "-json")
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute 'go list -m -json all': %w\nOutput:\n%s", err, string(output))
	}

	modules := make([]Module, 0)
	decoder := json.NewDecoder(bytes.NewReader(output))
	for decoder.More() {
		var goListMod GoListModule
		if err := decoder.Decode(&goListMod); err != nil {
			return nil, nil, fmt.Errorf("failed to parse JSON output from 'go list': %w", err)
		}
		if goListMod.Error != nil {
			continue
		}
		modules = append(modules, Module{
			Path:      goListMod.Path,
			Version:   goListMod.Version,
			Main:      goListMod.Main,
			Indirect:  goListMod.Indirect,
			Dir:       goListMod.Dir,
			GoMod:     goListMod.GoMod,
			GoVersion: goListMod.GoVersion,
		})
	}
	return nil, &OProjectUsedModules{Modules: modules}, nil
}

func projectPackages(_ context.Context, _ *mcp.CallToolRequest, input IUsedModules) (*mcp.CallToolResult, *OProjectDefinedPackages, error) {
	cwd := filepath.Clean(strings.TrimSpace(input.Cwd))
	if cwd == "" {
		return nil, nil, fmt.Errorf("input 'Cwd' (current working directory) cannot be empty")
	}
	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedDeps | packages.NeedImports | packages.NeedModule,
		Dir:  cwd,
	}
	pkgs, err := packages.Load(&cfg, "./...")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load packages for project %s: %w", cwd, err)
	}

	pkgMaps := make(map[string]Package, len(pkgs))
	for _, pkg := range pkgs {
		pkgMaps[pkg.PkgPath] = getPackageInfo(pkg, true)
	}

	return nil, &OProjectDefinedPackages{Packages: pkgMaps}, nil
}

func goProjectPackages(_ context.Context, _ *mcp.CallToolRequest, input IGoProjectPackage) (*mcp.CallToolResult, *OGoPackage, error) {
	cwd := filepath.Clean(strings.TrimSpace(input.Cwd))
	if cwd == "" {
		return nil, nil, fmt.Errorf("input 'Cwd' (current working directory) cannot be empty")
	}
	pkgPath := strings.TrimSpace(input.PackagePath)
	cfg := packages.Config{
		Mode: packages.LoadAllSyntax | packages.NeedModule,
		Dir:  cwd,
	}
	pkgs, err := packages.Load(&cfg, pkgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load packages for project %s: %w", cwd, err)
	}

	if len(pkgs) != 1 {
		return nil, nil, fmt.Errorf("load %d packages for path %s", len(pkgs), pkgPath)
	}
	pkg := pkgs[0]
	info := getPackageInfo(pkg, false)

	return nil, &OGoPackage{Package: info}, nil
}
