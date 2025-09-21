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

// ===== Output Structures ====

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

type Symbol struct {
	Name     string           `json:"name" jsonschema:"the name of the symbol"`
	Detail   string           `json:"detail" jsonschema:"the detail of the symbol, e.g., the signature of a function"`
	Kind     filedKind        `json:"kind" jsonschema:"the kind of the symbol, e.g., function, type, variable, constant"`
	Doc      string           `json:"doc" jsonschema:"the documentation of the symbol"`
	Children []DocumentSymbol `json:"children,omitempty" jsonschema:"the child symbols of the symbol"`
}

type Package struct {
	Name          string   `json:"name" jsonschema:"the name of the symbol"`
	ModuleName    string   `json:"module" jsonschema:"the associated module of a package"`
	ModuleVersion string   `json:"module_version" jsonschema:"the associated module version of a package"`
	Path          string   `json:"path" jsonschema:"the import path of a package"`
	Docs          string   `json:"docs" jsonschema:"the documentation of a package"`
	Symbols       []Symbol `json:"symbols" jsonschema:"the symbols in a package"`
}
