package tool

// ===== Input ====
type IEmpty struct{}

type IUsedModules struct {
	Cwd string `json:"cwd" jsonschema:"the current working directory to find the go.mod file"`
}

type IGoProjectPackage struct {
	Cwd         string `json:"cwd" jsonschema:"the current working directory to find the go.mod file"`
	PackagePath string `json:"package_path" jsonschema:"imported package path in current project"`
}

type ICheck struct {
	Cwd    string `json:"cwd" jsonschema:"the current working directory to find the go.mod file"`
	Path   string `json:"path" jsonschema:"the import path of the package to check"`
	Symbol string `json:"symbol" jsonschema:"the symbol to check, if not set, only check the package existence"`
}
type IPackageInfo struct {
	PackagePath string `json:"package_path" jsonschema:"the import path of the package"`
	Cwd         string `json:"cwd" jsonschema:"the current working directory to find the go.mod file"`
}

// ===== Output ====
type OGoInfo struct {
	Version string `json:"version" jsonschema:"the Go version"`
	GoBin   string `json:"gobin" jsonschema:"the GOBIN environment variable"`
	GOROOT  string `json:"goroot" jsonschema:"the GOROOT environment variable"`
}

type OStdlibSymbols struct {
	StdLibs map[string]Package `json:"stdlibs" jsonschema:"the Go standard libraries"`
}

type OProjectUsedModules struct {
	Modules []Module `json:"modules" jsonschema:"the used modules"`
}

type OProjectDefinedPackages struct {
	Packages map[string]Package `json:"packages" jsonschema:"packages under current project"`
}

type OGoPackage struct {
	Package Package `json:"package" jsonschema:"all exported symbols of a given package based on current project go.mod."`
}

type OCheck struct {
	Validated   bool   `json:"validated" jsonschema:"whether the check is successful"`
	Explanation string `json:"explanation,omitempty" jsonschema:"if not validated, the explanation of the failure"`
}

// ===== Other Structures ====

type Module struct {
	Path    string `json:"path" jsonschema:"the module path"`
	Version string `json:"version" jsonschema:"the module version"`
	// todo: consider whether and how to support Replace field.
	// Replace   *Module    `json:"replace" jsonschema:"replaced by this module"`
	Main      bool   `json:"main" jsonschema:"is this the main module?"`
	Indirect  bool   `json:"indirect" jsonschema:"is this module only an indirect dependency of main module?"`
	Dir       string `json:"dir" jsonschema:"directory holding files for this module, if any"`
	GoMod     string `json:"go_mod" jsonschema:"path to go.mod file used when loading this module, if any"`
	GoVersion string `json:"go_version" jsonschema:"go version used in module"`
}

type Pos struct {
	Filename string `json:"filename" jsonschema:"the filename of current position"`
	Offset   int    `json:"offset" jsonschema:"the line of current position, offset, starting at 0"`
	Line     int    `json:"line" jsonschema:"line number, starting at 1"`
	Column   int    `json:"column" jsonschema:"column number, starting at 1 (byte count)"`
}

type Symbol struct {
	ShortSymbol
	FilePath string `json:"file_path" jsonschema:"the symbol's file path"`
	Doc      string `json:"doc,omitempty" jsonschema:"the documentation of the symbol"`
	Start    Pos    `json:"start" jsonschema:"the place this symbol starts"`
	End      Pos    `json:"end" jsonschema:"the place this symbol ends"`
}

type Package struct {
	Name          string   `json:"name" jsonschema:"the name of the symbol"`
	Path          string   `json:"path" jsonschema:"the import path of a package"`
	ModuleName    string   `json:"module,omitempty" jsonschema:"the associated module of a package"`
	ModuleVersion string   `json:"module_version,omitempty" jsonschema:"the associated module version of a package"`
	Docs          string   `json:"docs,omitempty" jsonschema:"the documentation of a package"`
	Symbols       []Symbol `json:"symbols,omitempty" jsonschema:"the symbols in a package"`
}
