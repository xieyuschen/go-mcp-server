package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func gobin() ([]byte, error) {
	return exec.Command("which", "go").Output()
}

func goroot(goBin string) ([]byte, error) {
	return exec.Command(goBin, "env", "GOROOT").Output()
}

func goVersion(goBin string) (string, error) {
	out, err := exec.Command(goBin, "version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Go version: %w", err)
	}
	matches := reg.FindStringSubmatch(string(out))
	if len(matches) < 2 {
		return "", fmt.Errorf("failed to parse Go version: %w", err)
	}
	goVersion := matches[1]
	return goVersion, nil
}

func getPackageInfo(pkg *packages.Package, ignoreSymbols bool) Package {
	symbols := []Symbol{}
	var pkgDoc string
	for _, file := range pkg.Syntax {
		if file.Doc != nil {
			pkgDoc += file.Doc.Text() + "\n"
		}
		if ignoreSymbols {
			continue
		}
		tok := pkg.Fset.File(file.FileStart)
		for _, _decl := range file.Decls {
			switch decl := _decl.(type) {
			case *ast.FuncDecl:
				if decl.Name.Name == "_" || !ast.IsExported(decl.Name.Name) {
					continue
				}

				shortSym := funcSymbol(decl)
				start, end := Position(tok, decl.Pos()), Position(tok, decl.End())
				sym := Symbol{
					ShortSymbol: shortSym,
					Doc:         decl.Doc.Text(),
					Start:       Pos(start),
					End:         Pos(end),
					FilePath:    file.Name.Name,
				}
				symbols = append(symbols, sym)
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if spec.Name.Name == "_" || !ast.IsExported(spec.Name.Name) {
							continue
						}
						shortSym := typeSymbol(tok, spec)
						start, end := Position(tok, decl.Pos()), Position(tok, decl.End())
						sym := Symbol{
							ShortSymbol: shortSym,
							Doc:         decl.Doc.Text(),
							Start:       Pos(start),
							End:         Pos(end),
							FilePath:    file.Name.Name,
						}
						symbols = append(symbols, sym)
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if name.Name == "_" || !ast.IsExported(name.Name) {
								continue
							}
							shortSym := varSymbol(tok, spec, name, decl.Tok == token.CONST)
							start, end := Position(tok, decl.Pos()), Position(tok, decl.End())
							sym := Symbol{
								ShortSymbol: shortSym,
								Doc:         decl.Doc.Text(),
								Start:       Pos(start),
								End:         Pos(end),
								FilePath:    file.Name.Name,
							}
							symbols = append(symbols, sym)
						}
					}
				}
			}
		}
	}

	p := Package{
		Name:    pkg.Name,
		Path:    pkg.PkgPath,
		Docs:    strings.TrimSpace(pkgDoc),
		Symbols: symbols,
	}
	if pkg.Module != nil {
		p.ModuleName = pkg.Module.Path
		p.ModuleVersion = pkg.Module.Version
	}
	return p
}

type GoListModule struct {
	Path      string
	Version   string
	Main      bool
	Indirect  bool
	Dir       string
	GoMod     string
	GoVersion string
	Error     *struct {
		Err string
	}
}

type analzer struct {
	cwd string
}

func New(cwd string) (*analzer, error) {
	cwd = filepath.Clean(strings.TrimSpace(cwd))
	if cwd == "" {
		return nil, fmt.Errorf("input 'Cwd' (current working directory) cannot be empty")
	}
	return &analzer{cwd: cwd}, nil
}

func (a *analzer) usedModules() ([]Module, error) {
	cmd := exec.Command("go", "mod", "download", "-json")
	cmd.Dir = a.cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute 'go list -m -json all': %w\nOutput:\n%s", err, string(output))
	}

	modules := make([]Module, 0)
	decoder := json.NewDecoder(bytes.NewReader(output))
	for decoder.More() {
		var goListMod GoListModule
		if err := decoder.Decode(&goListMod); err != nil {
			return nil, fmt.Errorf("failed to parse JSON output from 'go list': %w", err)
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
	return modules, nil
}

func (a *analzer) projectDefinedPackages(skipSymbols bool) (map[string]Package, error) {
	cwd := filepath.Clean(strings.TrimSpace(a.cwd))
	if cwd == "" {
		return nil, fmt.Errorf("input 'Cwd' (current working directory) cannot be empty")
	}
	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedDeps | packages.NeedImports | packages.NeedModule,
		Dir:  cwd,
	}
	pkgs, err := packages.Load(&cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("failed to load packages for project %s: %w", cwd, err)
	}

	pkgMaps := make(map[string]Package, len(pkgs))
	for _, pkg := range pkgs {
		pkgMaps[pkg.PkgPath] = getPackageInfo(pkg, skipSymbols)
	}
	return pkgMaps, nil
}

func (a *analzer) projectSinglePackage(pkgPath string, skipSymbols bool) (*Package, error) {
	pkgPath = strings.TrimSpace(pkgPath)
	cfg := packages.Config{
		Mode: packages.LoadAllSyntax | packages.NeedModule,
		Dir:  a.cwd,
	}
	pkgs, err := packages.Load(&cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages for project %s: %w", a.cwd, err)
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("load %d packages for path %s", len(pkgs), pkgPath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		errMsgs := []string{}
		for _, e := range pkg.Errors {
			errMsgs = append(errMsgs, e.Error())
		}
		return nil, fmt.Errorf("failed to load package %s: %s", pkgPath, strings.Join(errMsgs, "; "))
	}

	info := getPackageInfo(pkg, skipSymbols)

	return &info, nil
}

type stdlib struct{}

func (s stdlib) listPackages(skipSymbols bool) (map[string]Package, error) {
	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedFiles,
	}
	pkgs, err := packages.Load(&cfg, "std")
	if err != nil {
		return nil, fmt.Errorf("failed to load Go standard library packages: %w", err)
	}
	stdLibMap := make(map[string]Package, len(pkgs))
	for _, pkg := range pkgs {
		if strings.HasPrefix(pkg.PkgPath, "internal/") {
			continue
		}

		stdLibMap[pkg.PkgPath] = getPackageInfo(pkg, skipSymbols)
	}
	return stdLibMap, nil
}
