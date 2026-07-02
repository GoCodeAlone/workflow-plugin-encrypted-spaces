package internal

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func TestEncryptedSpaceStateStoreSupportsMemoryAndFileModes(t *testing.T) {
	memory, err := newEncryptedSpaceStateStoreModuleFromConfig("memory-state", &contracts.StateStoreConfig{Backend: "memory"})
	if err != nil {
		t.Fatalf("memory state store: %v", err)
	}
	if memory.backend != stateStoreBackendMemory {
		t.Fatalf("memory backend = %q, want %q", memory.backend, stateStoreBackendMemory)
	}

	path := filepath.Join(t.TempDir(), "spaces.json")
	fileStore, err := newEncryptedSpaceStateStoreModuleFromConfig("file-state", &contracts.StateStoreConfig{
		Backend:             "file",
		StoragePath:         path,
		AllowFileStateStore: true,
		MaxSpaces:           4,
	})
	if err != nil {
		t.Fatalf("file state store: %v", err)
	}
	if fileStore.backend != stateStoreBackendFile {
		t.Fatalf("file backend = %q, want %q", fileStore.backend, stateStoreBackendFile)
	}
	if err := fileStore.Start(t.Context()); err != nil {
		t.Fatalf("start file state store: %v", err)
	}
	t.Cleanup(func() {
		_ = fileStore.Stop(t.Context())
	})

	state := initializedStateForTest(t, "space-file", "member-a")
	if _, err := ExecuteEncryptedSpaceStateSave(t.Context(), sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput]{
		Config: &contracts.StateSaveConfig{StateStore: "file-state"},
		Input:  &contracts.StateSaveInput{State: state},
	}); err != nil {
		t.Fatalf("save file state: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	var snapshot map[string]any
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("decode state file: %v", err)
	}
	if got := snapshot["schema_version"]; got != float64(1) {
		t.Fatalf("schema_version = %v, want 1", got)
	}
	if got := snapshot["backend"]; got != "file" {
		t.Fatalf("backend = %v, want file", got)
	}
	checksum, _ := snapshot["checksum"].(string)
	if !strings.HasPrefix(checksum, "sha256:") {
		t.Fatalf("checksum = %q, want sha256 prefix", checksum)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp state file still exists after save: %v", err)
	}
	for _, forbidden := range [][]byte{[]byte("plaintext"), []byte("proof-secret"), []byte("private-key-material")} {
		if bytes.Contains(raw, forbidden) {
			t.Fatalf("state file leaked forbidden secret %q: %s", forbidden, raw)
		}
	}
}

func TestEncryptedSpaceStateStoreInitRegistersForPipelineLifecycle(t *testing.T) {
	module, err := newEncryptedSpaceStateStoreModuleFromConfig("init-state", &contracts.StateStoreConfig{Backend: "memory"})
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	if err := module.Init(); err != nil {
		t.Fatalf("init state store: %v", err)
	}
	t.Cleanup(func() {
		_ = module.Stop(t.Context())
	})

	if _, err := ExecuteEncryptedSpaceStateSave(t.Context(), sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput]{
		Config: &contracts.StateSaveConfig{StateStore: "init-state"},
		Input:  &contracts.StateSaveInput{State: initializedStateForTest(t, "space-init", "member-a")},
	}); err != nil {
		t.Fatalf("save after init-only registration: %v", err)
	}
}

func TestEncryptedSpaceStateStoreRejectsCorruptFileSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, path string)
	}{
		{
			name: "checksum mismatch",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read state file: %v", err)
				}
				raw = bytes.Replace(raw, []byte("member-a"), []byte("member-x"), 1)
				if err := os.WriteFile(path, raw, 0o600); err != nil {
					t.Fatalf("write corrupted state file: %v", err)
				}
			},
		},
		{
			name: "unknown schema",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read state file: %v", err)
				}
				raw = bytes.Replace(raw, []byte(`"schema_version": 1`), []byte(`"schema_version": 999`), 1)
				if err := os.WriteFile(path, raw, 0o600); err != nil {
					t.Fatalf("write unknown schema state file: %v", err)
				}
			},
		},
		{
			name: "truncated",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(`{"schema_version":`), 0o600); err != nil {
					t.Fatalf("write truncated state file: %v", err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "spaces.json")
			saveStateStoreFixture(t, path, "member-a")
			tc.mutate(t, path)

			module, err := newEncryptedSpaceStateStoreModuleFromConfig("file-state-corrupt", &contracts.StateStoreConfig{
				Backend:             "file",
				StoragePath:         path,
				AllowFileStateStore: true,
			})
			if err != nil {
				t.Fatalf("new file state store: %v", err)
			}
			if err := module.Start(t.Context()); err == nil {
				t.Fatal("start with corrupt state file succeeded, want error")
			}
		})
	}
}

func TestEncryptedSpaceStateStoreRestartMidWriteLoadsOnlyCompleteSnapshots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spaces.json")
	saveStateStoreFixture(t, path, "member-old")

	if err := os.WriteFile(path+".tmp", []byte(`{"schema_version": 1, "backend": "file", "state":`), 0o600); err != nil {
		t.Fatalf("write partial temp state file: %v", err)
	}

	restarted, err := newEncryptedSpaceStateStoreModuleFromConfig("file-state-restart", &contracts.StateStoreConfig{
		Backend:             "file",
		StoragePath:         path,
		AllowFileStateStore: true,
	})
	if err != nil {
		t.Fatalf("new restarted store: %v", err)
	}
	if err := restarted.Start(t.Context()); err != nil {
		t.Fatalf("start restarted store: %v", err)
	}
	loaded, err := ExecuteEncryptedSpaceStateLoad(t.Context(), sdk.TypedStepRequest[*contracts.StateLoadConfig, *contracts.StateLoadInput]{
		Config: &contracts.StateLoadConfig{StateStore: "file-state-restart"},
		Input:  &contracts.StateLoadInput{SpaceId: "space-file"},
	})
	if err != nil {
		t.Fatalf("load restarted state: %v", err)
	}
	if got := loaded.Output.GetState().GetMembers(); len(got) != 1 || got[0] != "member-old" {
		t.Fatalf("members after partial restart = %v, want previous complete snapshot", got)
	}

	updated := initializedStateForTest(t, "space-file", "member-new")
	if _, err := ExecuteEncryptedSpaceStateSave(t.Context(), sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput]{
		Config: &contracts.StateSaveConfig{StateStore: "file-state-restart"},
		Input:  &contracts.StateSaveInput{State: updated},
	}); err != nil {
		t.Fatalf("save updated state: %v", err)
	}
	if err := restarted.Stop(t.Context()); err != nil {
		t.Fatalf("stop restarted store: %v", err)
	}

	reloaded, err := newEncryptedSpaceStateStoreModuleFromConfig("file-state-reloaded", &contracts.StateStoreConfig{
		Backend:             "file",
		StoragePath:         path,
		AllowFileStateStore: true,
	})
	if err != nil {
		t.Fatalf("new reloaded store: %v", err)
	}
	if err := reloaded.Start(t.Context()); err != nil {
		t.Fatalf("start reloaded store: %v", err)
	}
	t.Cleanup(func() {
		_ = reloaded.Stop(t.Context())
	})
	loadedNew, err := ExecuteEncryptedSpaceStateLoad(t.Context(), sdk.TypedStepRequest[*contracts.StateLoadConfig, *contracts.StateLoadInput]{
		Config: &contracts.StateLoadConfig{StateStore: "file-state-reloaded"},
		Input:  &contracts.StateLoadInput{SpaceId: "space-file"},
	})
	if err != nil {
		t.Fatalf("load reloaded state: %v", err)
	}
	if got := loadedNew.Output.GetState().GetMembers(); len(got) != 1 || got[0] != "member-new" {
		t.Fatalf("members after complete restart = %v, want new complete snapshot", got)
	}
}

func saveStateStoreFixture(t *testing.T, path, memberID string) {
	t.Helper()
	module, err := newEncryptedSpaceStateStoreModuleFromConfig("file-state-fixture", &contracts.StateStoreConfig{
		Backend:             "file",
		StoragePath:         path,
		AllowFileStateStore: true,
	})
	if err != nil {
		t.Fatalf("new fixture store: %v", err)
	}
	if err := module.Start(t.Context()); err != nil {
		t.Fatalf("start fixture store: %v", err)
	}
	state := initializedStateForTest(t, "space-file", memberID)
	if _, err := ExecuteEncryptedSpaceStateSave(t.Context(), sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput]{
		Config: &contracts.StateSaveConfig{StateStore: "file-state-fixture"},
		Input:  &contracts.StateSaveInput{State: state},
	}); err != nil {
		t.Fatalf("save fixture state: %v", err)
	}
	if err := module.Stop(t.Context()); err != nil {
		t.Fatalf("stop fixture store: %v", err)
	}
}
