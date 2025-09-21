package tool

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDocs(t *testing.T) {
	bs, err := os.ReadFile("../../README.md")
	if err != nil {
		panic(err)
	}

	newBytes := UpdateReadme(string(bs))

	if diff := cmp.Diff(string(bs), newBytes); diff != "" {
		t.Fatalf("generated readme and current readme is different, consider to run 'go generate ./...'. Diff: %s", diff)
	}
}
