package internal

import (
	"testing"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func TestEncryptedSpaceStateStoreSaveLoadAndMemberCheck(t *testing.T) {
	module := newEncryptedSpaceStateStoreModule("scenario-state")
	if err := module.Start(t.Context()); err != nil {
		t.Fatalf("start state store: %v", err)
	}
	t.Cleanup(func() {
		if err := module.Stop(t.Context()); err != nil {
			t.Fatalf("stop state store: %v", err)
		}
	})

	initResult, err := ExecuteEncryptedSpaceStateInit(t.Context(), sdk.TypedStepRequest[*contracts.StateInitConfig, *contracts.StateInitInput]{
		Input: &contracts.StateInitInput{
			SpaceId: "space-1",
			Members: []string{
				"member-2",
				"member-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	saveResult, err := ExecuteEncryptedSpaceStateSave(t.Context(), sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput]{
		Config: &contracts.StateSaveConfig{StateStore: "scenario-state"},
		Input:  &contracts.StateSaveInput{State: initResult.Output.State},
	})
	if err != nil {
		t.Fatalf("save state: %v", err)
	}
	if !saveResult.Output.Stored {
		t.Fatal("save state stored=false, want true")
	}
	saveResult.Output.State.Members[0] = "mutated"

	loadResult, err := ExecuteEncryptedSpaceStateLoad(t.Context(), sdk.TypedStepRequest[*contracts.StateLoadConfig, *contracts.StateLoadInput]{
		Config: &contracts.StateLoadConfig{StateStore: "scenario-state"},
		Input:  &contracts.StateLoadInput{SpaceId: "space-1"},
	})
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if !loadResult.Output.Found {
		t.Fatal("load state found=false, want true")
	}
	if got := loadResult.Output.State.Members; len(got) != 2 || got[0] != "member-1" || got[1] != "member-2" {
		t.Fatalf("loaded members = %v, want sorted persisted members", got)
	}

	checkResult, err := ExecuteEncryptedSpaceMemberCheck(t.Context(), sdk.TypedStepRequest[*contracts.MemberCheckConfig, *contracts.MemberCheckInput]{
		Input: &contracts.MemberCheckInput{
			State:    loadResult.Output.State,
			MemberId: "member-2",
		},
	})
	if err != nil {
		t.Fatalf("check member: %v", err)
	}
	if !checkResult.Output.MemberAllowed {
		t.Fatal("member-2 allowed=false, want true")
	}
}

func TestEncryptedSpaceStateStorePersistsUpdatedSnapshot(t *testing.T) {
	module := newEncryptedSpaceStateStoreModule("scenario-state-update")
	if err := module.Start(t.Context()); err != nil {
		t.Fatalf("start state store: %v", err)
	}
	t.Cleanup(func() {
		if err := module.Stop(t.Context()); err != nil {
			t.Fatalf("stop state store: %v", err)
		}
	})

	initResult, err := ExecuteEncryptedSpaceStateInit(t.Context(), sdk.TypedStepRequest[*contracts.StateInitConfig, *contracts.StateInitInput]{
		Input: &contracts.StateInitInput{SpaceId: "space-1", Members: []string{"member-1", "member-2"}},
	})
	if err != nil {
		t.Fatalf("init state: %v", err)
	}
	updateResult, err := ExecuteEncryptedSpaceStateUpdate(t.Context(), sdk.TypedStepRequest[*contracts.StateUpdateConfig, *contracts.StateUpdateInput]{
		Input: &contracts.StateUpdateInput{
			State:    initResult.Output.State,
			MemberId: "member-2",
			Action:   "remove",
			Reason:   "scenario removal",
		},
	})
	if err != nil {
		t.Fatalf("update state: %v", err)
	}
	if _, err := ExecuteEncryptedSpaceStateSave(t.Context(), sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput]{
		Config: &contracts.StateSaveConfig{StateStore: "scenario-state-update"},
		Input:  &contracts.StateSaveInput{State: updateResult.Output.State},
	}); err != nil {
		t.Fatalf("save updated state: %v", err)
	}

	loadResult, err := ExecuteEncryptedSpaceStateLoad(t.Context(), sdk.TypedStepRequest[*contracts.StateLoadConfig, *contracts.StateLoadInput]{
		Config: &contracts.StateLoadConfig{StateStore: "scenario-state-update"},
		Input:  &contracts.StateLoadInput{SpaceId: "space-1"},
	})
	if err != nil {
		t.Fatalf("load updated state: %v", err)
	}
	checkRemoved, err := ExecuteEncryptedSpaceMemberCheck(t.Context(), sdk.TypedStepRequest[*contracts.MemberCheckConfig, *contracts.MemberCheckInput]{
		Input: &contracts.MemberCheckInput{State: loadResult.Output.State, MemberId: "member-2"},
	})
	if err != nil {
		t.Fatalf("check removed member: %v", err)
	}
	if checkRemoved.Output.MemberAllowed || !checkRemoved.Output.MemberRemoved {
		t.Fatalf("removed member allowed=%v removed=%v, want allowed=false removed=true", checkRemoved.Output.MemberAllowed, checkRemoved.Output.MemberRemoved)
	}
}

func TestEncryptedSpaceStateStoreLoadMissing(t *testing.T) {
	module := newEncryptedSpaceStateStoreModule("scenario-state-missing")
	if err := module.Start(t.Context()); err != nil {
		t.Fatalf("start state store: %v", err)
	}
	t.Cleanup(func() {
		if err := module.Stop(t.Context()); err != nil {
			t.Fatalf("stop state store: %v", err)
		}
	})

	loadResult, err := ExecuteEncryptedSpaceStateLoad(t.Context(), sdk.TypedStepRequest[*contracts.StateLoadConfig, *contracts.StateLoadInput]{
		Config: &contracts.StateLoadConfig{StateStore: "scenario-state-missing"},
		Input:  &contracts.StateLoadInput{SpaceId: "space-absent"},
	})
	if err != nil {
		t.Fatalf("load missing state: %v", err)
	}
	if loadResult.Output.Found {
		t.Fatal("load missing found=true, want false")
	}
	if loadResult.Output.State != nil {
		t.Fatalf("load missing state = %#v, want nil", loadResult.Output.State)
	}
}

func TestEncryptedSpaceStateStoreRequiresRegisteredStore(t *testing.T) {
	initResult, err := ExecuteEncryptedSpaceStateInit(t.Context(), sdk.TypedStepRequest[*contracts.StateInitConfig, *contracts.StateInitInput]{
		Input: &contracts.StateInitInput{SpaceId: "space-1", Members: []string{"member-1"}},
	})
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	if _, err := ExecuteEncryptedSpaceStateSave(t.Context(), sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput]{
		Config: &contracts.StateSaveConfig{StateStore: "missing-state-store"},
		Input:  &contracts.StateSaveInput{State: initResult.Output.State},
	}); err == nil {
		t.Fatal("save with missing state store succeeded, want error")
	}
}
