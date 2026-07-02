package internal

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
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
		if !bytes.Contains(data, []byte("version: 1\n")) {
			t.Fatalf("%s must declare wfctl project manifest version: 1", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scenario manifests: %v", err)
	}
}
