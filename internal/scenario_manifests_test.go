package internal

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScenarioProjectManifestsDeclareVersion(t *testing.T) {
	scenariosDir := filepath.Join("..", "scenarios")
	err := filepath.WalkDir(scenariosDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "wfctl.yaml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if firstManifestLine(data) != "version: 1" {
			t.Fatalf("%s must declare wfctl project manifest version: 1", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scenario manifests: %v", err)
	}
}

func firstManifestLine(data []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
	}
	return ""
}
