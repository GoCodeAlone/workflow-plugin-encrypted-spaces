package internal

import (
	"errors"
	"testing"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func TestStateLifecycleSteps(t *testing.T) {
	initResult, err := ExecuteEncryptedSpaceStateInit(t.Context(), sdk.TypedStepRequest[*contracts.StateInitConfig, *contracts.StateInitInput]{
		Config: &contracts.StateInitConfig{},
		Input: &contracts.StateInitInput{
			SpaceId: "space-1",
			Members: []string{
				"member-2",
				"member-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("state init: %v", err)
	}
	if got := initResult.Output.GetState().GetSpaceId(); got != "space-1" {
		t.Fatalf("space id = %q, want space-1", got)
	}
	if got := initResult.Output.GetState().GetKeyEpoch(); got != 1 {
		t.Fatalf("key epoch = %d, want 1", got)
	}
	if got := initResult.Output.GetState().GetMembershipEpoch(); got != 1 {
		t.Fatalf("membership epoch = %d, want 1", got)
	}
	if got := initResult.Output.GetState().GetMembers(); !stringSlicesEqual(got, []string{"member-1", "member-2"}) {
		t.Fatalf("members = %v, want sorted members", got)
	}

	updateResult, err := ExecuteEncryptedSpaceStateUpdate(t.Context(), sdk.TypedStepRequest[*contracts.StateUpdateConfig, *contracts.StateUpdateInput]{
		Config: &contracts.StateUpdateConfig{},
		Input: &contracts.StateUpdateInput{
			State:    initResult.Output.GetState(),
			MemberId: "member-2",
			Action:   "remove",
			Reason:   "access revoked",
		},
	})
	if err != nil {
		t.Fatalf("state update: %v", err)
	}
	if updateResult.Output.GetMemberAllowed() {
		t.Fatal("removed member is still allowed")
	}
	if got := updateResult.Output.GetMembershipEpoch(); got != 2 {
		t.Fatalf("membership epoch = %d, want 2", got)
	}
	if got := updateResult.Output.GetState().GetRemovedMembers(); !stringSlicesEqual(got, []string{"member-2"}) {
		t.Fatalf("removed members = %v, want member-2", got)
	}

	checkRemoved, err := ExecuteEncryptedSpaceMemberCheck(t.Context(), sdk.TypedStepRequest[*contracts.MemberCheckConfig, *contracts.MemberCheckInput]{
		Config: &contracts.MemberCheckConfig{},
		Input: &contracts.MemberCheckInput{
			State:    updateResult.Output.GetState(),
			MemberId: "member-2",
		},
	})
	if err != nil {
		t.Fatalf("member check removed: %v", err)
	}
	if checkRemoved.Output.GetMemberAllowed() {
		t.Fatal("removed member check allowed member-2")
	}
	if !checkRemoved.Output.GetMemberRemoved() {
		t.Fatal("removed member check did not mark member-2 removed")
	}

	checkActive, err := ExecuteEncryptedSpaceMemberCheck(t.Context(), sdk.TypedStepRequest[*contracts.MemberCheckConfig, *contracts.MemberCheckInput]{
		Config: &contracts.MemberCheckConfig{},
		Input: &contracts.MemberCheckInput{
			State:    updateResult.Output.GetState(),
			MemberId: "member-1",
		},
	})
	if err != nil {
		t.Fatalf("member check active: %v", err)
	}
	if !checkActive.Output.GetMemberAllowed() || checkActive.Output.GetMemberRemoved() {
		t.Fatalf("active member check = allowed %v removed %v, want allowed true removed false", checkActive.Output.GetMemberAllowed(), checkActive.Output.GetMemberRemoved())
	}
}

func TestStateLifecycleStepsRejectMalformedSnapshots(t *testing.T) {
	_, err := ExecuteEncryptedSpaceStateUpdate(t.Context(), sdk.TypedStepRequest[*contracts.StateUpdateConfig, *contracts.StateUpdateInput]{
		Config: &contracts.StateUpdateConfig{},
		Input: &contracts.StateUpdateInput{
			State: &contracts.SpaceState{
				SpaceId:         "space-1",
				KeyEpoch:        1,
				MembershipEpoch: 1,
				Members:         []string{"member-1"},
				RemovedMembers:  []string{"member-1"},
			},
			MemberId: "member-2",
			Action:   "add",
		},
	})
	if err == nil {
		t.Fatal("expected malformed snapshot error")
	}
	if errors.Is(err, sdk.ErrTypedContractNotHandled) {
		t.Fatalf("state update returned contract error instead of validation error: %v", err)
	}

	initResult, err := ExecuteEncryptedSpaceStateInit(t.Context(), sdk.TypedStepRequest[*contracts.StateInitConfig, *contracts.StateInitInput]{
		Config: &contracts.StateInitConfig{},
		Input: &contracts.StateInitInput{
			SpaceId: "space-1",
			Members: []string{
				"member-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("state init: %v", err)
	}
	_, err = ExecuteEncryptedSpaceStateUpdate(t.Context(), sdk.TypedStepRequest[*contracts.StateUpdateConfig, *contracts.StateUpdateInput]{
		Config: &contracts.StateUpdateConfig{},
		Input: &contracts.StateUpdateInput{
			State:  initResult.Output.GetState(),
			Action: "add",
		},
	})
	if err == nil {
		t.Fatal("expected empty member id update error")
	}
}

func TestStateLifecycleOutputsDoNotExposeOperationMaterial(t *testing.T) {
	result, err := ExecuteEncryptedSpaceStateInit(t.Context(), sdk.TypedStepRequest[*contracts.StateInitConfig, *contracts.StateInitInput]{
		Config: &contracts.StateInitConfig{},
		Input: &contracts.StateInitInput{
			SpaceId: "space-1",
			Members: []string{
				"member-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("state init: %v", err)
	}
	state := result.Output.GetState()
	if state.GetSpaceId() == "" || len(state.GetMembers()) != 1 {
		t.Fatalf("unexpected state output: %#v", state)
	}
}

func stringSlicesEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
