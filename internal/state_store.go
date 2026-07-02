package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GoCodeAlone/encrypted-spaces-go/operationlog"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

const (
	stateStoreBackendMemory = "memory"
	stateStoreBackendFile   = "file"
	stateStoreSchemaVersion = 1
)

var encryptedSpaceStateStores = struct {
	sync.RWMutex
	stores map[string]*encryptedSpaceStateStore
}{
	stores: map[string]*encryptedSpaceStateStore{},
}

type encryptedSpaceStateStoreModule struct {
	name      string
	backend   string
	maxSpaces uint64
	store     *encryptedSpaceStateStore
}

type encryptedSpaceStateStore struct {
	mu          sync.RWMutex
	backend     string
	storagePath string
	maxSpaces   uint64
	states      map[string]*contracts.SpaceState
}

type stateStoreFileSnapshot struct {
	SchemaVersion int                   `json:"schema_version"`
	Backend       string                `json:"backend"`
	Checksum      string                `json:"checksum"`
	State         stateStoreFilePayload `json:"state"`
}

type stateStoreFilePayload struct {
	Spaces map[string]stateStoreFileSpace `json:"spaces"`
}

type stateStoreFileSpace struct {
	SpaceID         string   `json:"space_id"`
	KeyEpoch        uint64   `json:"key_epoch"`
	MembershipEpoch uint64   `json:"membership_epoch"`
	Members         []string `json:"members,omitempty"`
	RemovedMembers  []string `json:"removed_members,omitempty"`
}

func newEncryptedSpaceStateStoreModule(name string) *encryptedSpaceStateStoreModule {
	return &encryptedSpaceStateStoreModule{
		name:    name,
		backend: stateStoreBackendMemory,
		store: &encryptedSpaceStateStore{
			backend: stateStoreBackendMemory,
			states:  map[string]*contracts.SpaceState{},
		},
	}
}

func newEncryptedSpaceStateStoreModuleFromConfig(name string, cfg *contracts.StateStoreConfig) (*encryptedSpaceStateStoreModule, error) {
	module := newEncryptedSpaceStateStoreModule(name)
	if cfg == nil {
		return module, nil
	}
	backend := cfg.GetBackend()
	if backend == "" {
		backend = stateStoreBackendMemory
	}
	switch backend {
	case stateStoreBackendMemory:
	case stateStoreBackendFile:
		if !cfg.GetAllowFileStateStore() {
			return nil, fmt.Errorf("encrypted space state store %q: file backend requires allow_file_state_store", name)
		}
		if cfg.GetStoragePath() == "" {
			return nil, fmt.Errorf("encrypted space state store %q: storage_path is required", name)
		}
		if productionPolicyMode(cfg.GetPolicyMode()) {
			return nil, fmt.Errorf("encrypted space state store %q: production policy rejects file backend", name)
		}
	default:
		return nil, fmt.Errorf("encrypted space state store %q: unsupported backend %q", name, cfg.GetBackend())
	}
	module.backend = backend
	module.maxSpaces = cfg.GetMaxSpaces()
	module.store.backend = backend
	module.store.storagePath = cfg.GetStoragePath()
	module.store.maxSpaces = cfg.GetMaxSpaces()
	return module, nil
}

func (m *encryptedSpaceStateStoreModule) Init() error {
	return m.register()
}

func (m *encryptedSpaceStateStoreModule) Start(ctx context.Context) error {
	_ = ctx
	return m.register()
}

func (m *encryptedSpaceStateStoreModule) register() error {
	encryptedSpaceStateStores.RLock()
	if encryptedSpaceStateStores.stores[m.name] == m.store {
		encryptedSpaceStateStores.RUnlock()
		return nil
	}
	encryptedSpaceStateStores.RUnlock()

	if m.backend == stateStoreBackendFile {
		if err := m.store.loadFromFile(); err != nil {
			return fmt.Errorf("encrypted space state store %q: %w", m.name, err)
		}
	}
	encryptedSpaceStateStores.Lock()
	defer encryptedSpaceStateStores.Unlock()
	if encryptedSpaceStateStores.stores[m.name] == m.store {
		return nil
	}
	if _, exists := encryptedSpaceStateStores.stores[m.name]; exists {
		return fmt.Errorf("encrypted space state store %q already registered", m.name)
	}
	encryptedSpaceStateStores.stores[m.name] = m.store
	return nil
}

func (m *encryptedSpaceStateStoreModule) Stop(ctx context.Context) error {
	_ = ctx
	encryptedSpaceStateStores.Lock()
	defer encryptedSpaceStateStores.Unlock()
	if encryptedSpaceStateStores.stores[m.name] == m.store {
		delete(encryptedSpaceStateStores.stores, m.name)
	}
	return nil
}

func ExecuteEncryptedSpaceStateLoad(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.StateLoadConfig, *contracts.StateLoadInput],
) (*sdk.TypedStepResult[*contracts.StateLoadOutput], error) {
	store, err := lookupEncryptedSpaceStateStore(req.Config.GetStateStore())
	if err != nil {
		return nil, fmt.Errorf("encrypted space state load: %w", err)
	}
	if req.Input == nil || req.Input.GetSpaceId() == "" {
		return nil, fmt.Errorf("encrypted space state load: space id is required")
	}
	state := store.load(operationlog.SpaceID(req.Input.GetSpaceId()))
	if state == nil {
		return &sdk.TypedStepResult[*contracts.StateLoadOutput]{
			Output: &contracts.StateLoadOutput{Found: false},
		}, nil
	}
	return &sdk.TypedStepResult[*contracts.StateLoadOutput]{
		Output: &contracts.StateLoadOutput{State: state, Found: true},
	}, nil
}

func ExecuteEncryptedSpaceStateSave(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput],
) (*sdk.TypedStepResult[*contracts.StateSaveOutput], error) {
	store, err := lookupEncryptedSpaceStateStore(req.Config.GetStateStore())
	if err != nil {
		return nil, fmt.Errorf("encrypted space state save: %w", err)
	}
	if req.Input == nil || req.Input.GetState() == nil {
		return nil, fmt.Errorf("encrypted space state save: state is required")
	}
	state, err := stateFromContract(req.Input.GetState())
	if err != nil {
		return nil, fmt.Errorf("encrypted space state save: %w", err)
	}
	snapshot := spaceStateToContract(state.Snapshot())
	if err := store.save(snapshot); err != nil {
		return nil, fmt.Errorf("encrypted space state save: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.StateSaveOutput]{
		Output: &contracts.StateSaveOutput{State: cloneSpaceState(snapshot), Stored: true},
	}, nil
}

func lookupEncryptedSpaceStateStore(name string) (*encryptedSpaceStateStore, error) {
	if name == "" {
		return nil, fmt.Errorf("state_store is required")
	}
	encryptedSpaceStateStores.RLock()
	store := encryptedSpaceStateStores.stores[name]
	encryptedSpaceStateStores.RUnlock()
	if store == nil {
		return nil, fmt.Errorf("state store %q is not registered", name)
	}
	return store, nil
}

func (s *encryptedSpaceStateStore) load(spaceID operationlog.SpaceID) *contracts.SpaceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSpaceState(s.states[string(spaceID)])
}

func (s *encryptedSpaceStateStore) save(state *contracts.SpaceState) error {
	if state.GetSpaceId() == "" {
		return fmt.Errorf("space id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.maxSpaces > 0 {
		if _, exists := s.states[state.GetSpaceId()]; !exists && uint64(len(s.states)) >= s.maxSpaces {
			return fmt.Errorf("state store has reached max_spaces=%d", s.maxSpaces)
		}
	}
	previous, hadPrevious := s.states[state.GetSpaceId()]
	s.states[state.GetSpaceId()] = cloneSpaceState(state)
	if err := s.persistLocked(); err != nil {
		if hadPrevious {
			s.states[state.GetSpaceId()] = previous
		} else {
			delete(s.states, state.GetSpaceId())
		}
		return err
	}
	return nil
}

func (s *encryptedSpaceStateStore) loadFromFile() error {
	raw, err := os.ReadFile(s.storagePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read state file: %w", err)
	}
	var snapshot stateStoreFileSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return fmt.Errorf("decode state file: %w", err)
	}
	if snapshot.SchemaVersion != stateStoreSchemaVersion {
		return fmt.Errorf("unsupported state file schema_version=%d", snapshot.SchemaVersion)
	}
	if snapshot.Backend != stateStoreBackendFile {
		return fmt.Errorf("state file backend=%q, want %q", snapshot.Backend, stateStoreBackendFile)
	}
	if snapshot.Checksum == "" {
		return fmt.Errorf("state file checksum is required")
	}
	expected, err := stateStoreChecksum(snapshot.SchemaVersion, snapshot.Backend, snapshot.State)
	if err != nil {
		return err
	}
	if snapshot.Checksum != expected {
		return fmt.Errorf("state file checksum mismatch")
	}
	loaded := make(map[string]*contracts.SpaceState, len(snapshot.State.Spaces))
	for id, state := range snapshot.State.Spaces {
		if id == "" || state.SpaceID == "" || id != state.SpaceID {
			return fmt.Errorf("state file contains invalid space id %q/%q", id, state.SpaceID)
		}
		contractState := &contracts.SpaceState{
			SpaceId:         state.SpaceID,
			KeyEpoch:        state.KeyEpoch,
			MembershipEpoch: state.MembershipEpoch,
			Members:         append([]string(nil), state.Members...),
			RemovedMembers:  append([]string(nil), state.RemovedMembers...),
		}
		if _, err := stateFromContract(contractState); err != nil {
			return fmt.Errorf("state file contains invalid state for %q: %w", id, err)
		}
		loaded[id] = contractState
	}
	if s.maxSpaces > 0 && uint64(len(loaded)) > s.maxSpaces {
		return fmt.Errorf("state file contains %d spaces, max_spaces=%d", len(loaded), s.maxSpaces)
	}
	s.mu.Lock()
	s.states = loaded
	s.mu.Unlock()
	return nil
}

func (s *encryptedSpaceStateStore) persistLocked() error {
	if s.backend != stateStoreBackendFile {
		return nil
	}
	snapshot := stateStoreFileSnapshot{
		SchemaVersion: stateStoreSchemaVersion,
		Backend:       stateStoreBackendFile,
		State:         s.filePayloadLocked(),
	}
	checksum, err := stateStoreChecksum(snapshot.SchemaVersion, snapshot.Backend, snapshot.State)
	if err != nil {
		return err
	}
	snapshot.Checksum = checksum
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o700); err != nil {
		return fmt.Errorf("create state file dir: %w", err)
	}
	tmp := s.storagePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write state temp file: %w", err)
	}
	if err := os.Rename(tmp, s.storagePath); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}
	return nil
}

func (s *encryptedSpaceStateStore) filePayloadLocked() stateStoreFilePayload {
	payload := stateStoreFilePayload{Spaces: make(map[string]stateStoreFileSpace, len(s.states))}
	for id, state := range s.states {
		payload.Spaces[id] = stateStoreFileSpace{
			SpaceID:         state.GetSpaceId(),
			KeyEpoch:        state.GetKeyEpoch(),
			MembershipEpoch: state.GetMembershipEpoch(),
			Members:         append([]string(nil), state.GetMembers()...),
			RemovedMembers:  append([]string(nil), state.GetRemovedMembers()...),
		}
	}
	return payload
}

func stateStoreChecksum(schemaVersion int, backend string, payload stateStoreFilePayload) (string, error) {
	canonical := struct {
		SchemaVersion int                   `json:"schema_version"`
		Backend       string                `json:"backend"`
		State         stateStoreFilePayload `json:"state"`
	}{
		SchemaVersion: schemaVersion,
		Backend:       backend,
		State:         payload,
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("checksum state file: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func productionPolicyMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

func cloneSpaceState(state *contracts.SpaceState) *contracts.SpaceState {
	if state == nil {
		return nil
	}
	return &contracts.SpaceState{
		SpaceId:         state.GetSpaceId(),
		KeyEpoch:        state.GetKeyEpoch(),
		MembershipEpoch: state.GetMembershipEpoch(),
		Members:         append([]string(nil), state.GetMembers()...),
		RemovedMembers:  append([]string(nil), state.GetRemovedMembers()...),
	}
}
