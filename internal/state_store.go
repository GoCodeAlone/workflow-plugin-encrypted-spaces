package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoCodeAlone/encrypted-spaces-go/operationlog"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

const stateStoreBackendMemory = "memory"

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
	mu        sync.RWMutex
	maxSpaces uint64
	states    map[string]*contracts.SpaceState
}

func newEncryptedSpaceStateStoreModule(name string) *encryptedSpaceStateStoreModule {
	return &encryptedSpaceStateStoreModule{
		name:    name,
		backend: stateStoreBackendMemory,
		store: &encryptedSpaceStateStore{
			states: map[string]*contracts.SpaceState{},
		},
	}
}

func newEncryptedSpaceStateStoreModuleFromConfig(name string, cfg *contracts.StateStoreConfig) (*encryptedSpaceStateStoreModule, error) {
	module := newEncryptedSpaceStateStoreModule(name)
	if cfg == nil {
		return module, nil
	}
	if cfg.GetBackend() != "" && cfg.GetBackend() != stateStoreBackendMemory {
		return nil, fmt.Errorf("encrypted space state store %q: unsupported backend %q", name, cfg.GetBackend())
	}
	module.maxSpaces = cfg.GetMaxSpaces()
	module.store.maxSpaces = cfg.GetMaxSpaces()
	return module, nil
}

func (m *encryptedSpaceStateStoreModule) Init() error {
	return nil
}

func (m *encryptedSpaceStateStoreModule) Start(ctx context.Context) error {
	_ = ctx
	encryptedSpaceStateStores.Lock()
	defer encryptedSpaceStateStores.Unlock()
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
	s.states[state.GetSpaceId()] = cloneSpaceState(state)
	return nil
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
