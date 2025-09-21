//go:build ignore

package main

import (
	"os"

	"github.com/xieyuschen/go-mcp-server/internal/tool"
)

func main() {
	readme := "../../README.md"
	bs, err := os.ReadFile(readme)
	if err != nil {
		panic(err)
	}

	newBytes := tool.UpdateReadme(string(bs))
	err = os.WriteFile(readme, []byte(newBytes), 0666)
	if err != nil {
		panic(err)
	}
}
