package tool

import (
	"fmt"
	"go/ast"
	"go/token"
	"os/exec"
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
