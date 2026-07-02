package internal

import (
	"context"
	"fmt"
	"slices"

	"github.com/GoCodeAlone/encrypted-spaces-go/epochs"
	"github.com/GoCodeAlone/encrypted-spaces-go/operationlog"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func ExecuteEncryptedSpaceStateInit(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.StateInitConfig, *contracts.StateInitInput],
) (*sdk.TypedStepResult[*contracts.StateInitOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space state init: input is required")
	}
	keyEpoch := req.Input.GetKeyEpoch()
	if keyEpoch == 0 {
		keyEpoch = 1
	}
	membershipEpoch := req.Input.GetMembershipEpoch()
	if membershipEpoch == 0 {
		membershipEpoch = 1
	}
	snapshot := epochs.SpaceSnapshot{
		SpaceID:         operationlog.SpaceID(req.Input.GetSpaceId()),
		KeyEpoch:        operationlog.KeyEpoch(keyEpoch),
		MembershipEpoch: operationlog.MembershipEpoch(membershipEpoch),
		Members:         memberIDsFromStrings(req.Input.GetMembers()),
	}
	state, err := epochs.NewSpaceStateFromSnapshot(snapshot)
	if err != nil {
		return nil, fmt.Errorf("encrypted space state init: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.StateInitOutput]{
		Output: &contracts.StateInitOutput{State: spaceStateToContract(state.Snapshot())},
	}, nil
}

func ExecuteEncryptedSpaceStateUpdate(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.StateUpdateConfig, *contracts.StateUpdateInput],
) (*sdk.TypedStepResult[*contracts.StateUpdateOutput], error) {
	if req.Input == nil || req.Input.GetState() == nil {
		return nil, fmt.Errorf("encrypted space state update: state is required")
	}
	if req.Input.GetMemberId() == "" {
		return nil, fmt.Errorf("encrypted space state update: member id is required")
	}
	state, err := stateFromContract(req.Input.GetState())
	if err != nil {
		return nil, fmt.Errorf("encrypted space state update: %w", err)
	}
	membershipEpoch, err := state.ApplyMemberUpdate(epochs.MemberUpdate{
		MemberID: operationlog.MemberID(req.Input.GetMemberId()),
		Action:   epochs.MemberAction(req.Input.GetAction()),
		Reason:   req.Input.GetReason(),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space state update: %w", err)
	}
	memberAllowed := state.AllowsMember(operationlog.MemberID(req.Input.GetMemberId()))
	return &sdk.TypedStepResult[*contracts.StateUpdateOutput]{
		Output: &contracts.StateUpdateOutput{
			State:           spaceStateToContract(state.Snapshot()),
			MembershipEpoch: uint64(membershipEpoch),
			MemberAllowed:   memberAllowed,
			Action:          req.Input.GetAction(),
		},
	}, nil
}

func ExecuteEncryptedSpaceMemberCheck(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.MemberCheckConfig, *contracts.MemberCheckInput],
) (*sdk.TypedStepResult[*contracts.MemberCheckOutput], error) {
	if req.Input == nil || req.Input.GetState() == nil {
		return nil, fmt.Errorf("encrypted space member check: state is required")
	}
	if req.Input.GetMemberId() == "" {
		return nil, fmt.Errorf("encrypted space member check: member id is required")
	}
	state, err := stateFromContract(req.Input.GetState())
	if err != nil {
		return nil, fmt.Errorf("encrypted space member check: %w", err)
	}
	memberID := operationlog.MemberID(req.Input.GetMemberId())
	snapshot := state.Snapshot()
	memberAllowed := state.AllowsMember(memberID)
	memberRemoved := slices.Contains(snapshot.RemovedMembers, memberID)
	membershipStatus := "denied"
	switch {
	case memberAllowed:
		membershipStatus = "allowed"
	case memberRemoved:
		membershipStatus = "removed"
	}
	return &sdk.TypedStepResult[*contracts.MemberCheckOutput]{
		Output: &contracts.MemberCheckOutput{
			MemberAllowed:    memberAllowed,
			MemberRemoved:    memberRemoved,
			KeyEpoch:         uint64(snapshot.KeyEpoch),
			MembershipEpoch:  uint64(snapshot.MembershipEpoch),
			MembershipStatus: membershipStatus,
		},
	}, nil
}

func stateFromContract(state *contracts.SpaceState) (*epochs.SpaceState, error) {
	if state == nil {
		return nil, fmt.Errorf("state is required")
	}
	return epochs.NewSpaceStateFromSnapshot(epochs.SpaceSnapshot{
		SpaceID:         operationlog.SpaceID(state.GetSpaceId()),
		KeyEpoch:        operationlog.KeyEpoch(state.GetKeyEpoch()),
		MembershipEpoch: operationlog.MembershipEpoch(state.GetMembershipEpoch()),
		Members:         memberIDsFromStrings(state.GetMembers()),
		RemovedMembers:  memberIDsFromStrings(state.GetRemovedMembers()),
	})
}

func spaceStateToContract(snapshot epochs.SpaceSnapshot) *contracts.SpaceState {
	return &contracts.SpaceState{
		SpaceId:         string(snapshot.SpaceID),
		KeyEpoch:        uint64(snapshot.KeyEpoch),
		MembershipEpoch: uint64(snapshot.MembershipEpoch),
		Members:         stringsFromMemberIDs(snapshot.Members),
		RemovedMembers:  stringsFromMemberIDs(snapshot.RemovedMembers),
	}
}

func memberIDsFromStrings(members []string) []operationlog.MemberID {
	ids := make([]operationlog.MemberID, 0, len(members))
	for _, member := range members {
		ids = append(ids, operationlog.MemberID(member))
	}
	return ids
}

func stringsFromMemberIDs(members []operationlog.MemberID) []string {
	values := make([]string, 0, len(members))
	for _, member := range members {
		values = append(values, string(member))
	}
	return values
}
