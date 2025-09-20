package tool

import (
	"fmt"
	"os/exec"
)

type OGoStdLib struct {
	StdLibs map[string]Stdlib `json:"stdlibs" jsonschema:"the Go standard libraries"`
}

type Symbol struct {
	Name     string           `json:"name" jsonschema:"the name of the symbol"`
	Detail   string           `json:"detail" jsonschema:"the detail of the symbol, e.g., the signature of a function"`
	Kind     filedKind        `json:"kind" jsonschema:"the kind of the symbol, e.g., function, type, variable, constant"`
	Doc      string           `json:"doc" jsonschema:"the documentation of the symbol"`
	Children []DocumentSymbol `json:"children,omitempty" jsonschema:"the child symbols of the symbol"`
}

type Stdlib struct {
	Path    string   `json:"path" jsonschema:"the import path of the standard library"`
	Docs    string   `json:"docs" jsonschema:"the documentation of the standard library"`
	Symbols []Symbol `json:"symbols" jsonschema:"the symbols in the standard library"`
}

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
