package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/GoCodeAlone/encrypted-spaces-go/epochs"
	"github.com/GoCodeAlone/encrypted-spaces-go/operationlog"
	"github.com/GoCodeAlone/encrypted-spaces-go/verification"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func ExecuteEncryptedSpaceAppend(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.AppendConfig, *contracts.AppendInput],
) (*sdk.TypedStepResult[*contracts.AppendOutput], error) {
	if req.Input == nil || req.Input.GetOperation() == nil {
		return nil, fmt.Errorf("encrypted space append: operation is required")
	}
	if !req.Config.GetAllowFakeVerifier() {
		return nil, fmt.Errorf("encrypted space append: fake verifier must be explicitly allowed")
	}
	log, err := operationlog.NewLog(operationlog.LogOptions{Retention: retentionFromContract(req.Config.GetRetention())})
	if err != nil {
		return nil, fmt.Errorf("encrypted space append: log: %w", err)
	}
	commitment, err := log.Append(operationFromContract(req.Input.GetOperation()))
	if err != nil {
		return nil, fmt.Errorf("encrypted space append: %w", err)
	}
	report, err := verification.NewFakeVerifier().VerifyOperation(verification.FakeProof{
		OperationID: commitment.OperationID,
		Digest:      commitment.Digest,
		Proof:       []byte("fake-proof"),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space append: fake verifier: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.AppendOutput]{
		Output: &contracts.AppendOutput{
			Commitment:   commitmentToContract(commitment),
			Verification: reportToContract(report),
		},
	}, nil
}

func ExecuteEncryptedSpaceFastForward(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.FastForwardConfig, *contracts.FastForwardInput],
) (*sdk.TypedStepResult[*contracts.FastForwardOutput], error) {
	if req.Input == nil || req.Input.GetCommitment() == nil || req.Input.GetCheckpoint() == nil {
		return nil, fmt.Errorf("encrypted space fast-forward: commitment and checkpoint are required")
	}
	commitment := req.Input.GetCommitment()
	checkpoint := req.Input.GetCheckpoint()
	accepted := checkpoint.GetSpaceId() == commitment.GetSpaceId() &&
		checkpoint.GetOperationId() == commitment.GetOperationId() &&
		checkpoint.GetDigest() == commitment.GetDigest() &&
		checkpoint.GetThroughSequence() == commitment.GetSequence() &&
		checkpoint.GetKeyEpoch() == commitment.GetKeyEpoch() &&
		checkpoint.GetMembershipEpoch() == commitment.GetMembershipEpoch()
	if !accepted {
		return nil, fmt.Errorf("encrypted space fast-forward: checkpoint mismatch")
	}
	return &sdk.TypedStepResult[*contracts.FastForwardOutput]{
		Output: &contracts.FastForwardOutput{Accepted: true, Status: "accepted"},
	}, nil
}

func ExecuteEncryptedSpaceEpochRotate(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.EpochRotateConfig, *contracts.EpochRotateInput],
) (*sdk.TypedStepResult[*contracts.EpochRotateOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space epoch rotate: input is required")
	}
	state := epochs.NewSpaceState(operationlog.SpaceID(req.Input.GetSpaceId()), nil)
	for state.KeyEpoch() < operationlog.KeyEpoch(req.Input.GetCurrentKeyEpoch()) {
		state.RotateKeyEpoch("hydrate")
	}
	keyEpoch := state.RotateKeyEpoch(req.Input.GetReason())
	return &sdk.TypedStepResult[*contracts.EpochRotateOutput]{
		Output: &contracts.EpochRotateOutput{KeyEpoch: uint64(keyEpoch), Reason: req.Input.GetReason()},
	}, nil
}

func ExecuteEncryptedSpaceMemberUpdate(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.MemberUpdateConfig, *contracts.MemberUpdateInput],
) (*sdk.TypedStepResult[*contracts.MemberUpdateOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space member update: input is required")
	}
	members := make([]operationlog.MemberID, 0, len(req.Input.GetMembers()))
	for _, member := range req.Input.GetMembers() {
		members = append(members, operationlog.MemberID(member))
	}
	state := epochs.NewSpaceState(operationlog.SpaceID(req.Input.GetSpaceId()), members)
	action := epochs.MemberAction(req.Input.GetAction())
	membershipEpoch, err := state.ApplyMemberUpdate(epochs.MemberUpdate{
		MemberID: operationlog.MemberID(req.Input.GetMemberId()),
		Action:   action,
		Reason:   req.Input.GetReason(),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space member update: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.MemberUpdateOutput]{
		Output: &contracts.MemberUpdateOutput{
			MembershipEpoch: uint64(membershipEpoch),
			MemberAllowed:   state.AllowsMember(operationlog.MemberID(req.Input.GetMemberId())),
			Action:          req.Input.GetAction(),
		},
	}, nil
}

func retentionFromContract(policy *contracts.RetentionPolicy) operationlog.RetentionPolicy {
	if policy == nil {
		return operationlog.RetentionPolicy{}
	}
	return operationlog.RetentionPolicy{
		MaxOperations:      int(policy.GetMaxOperations()),
		MinKeyEpoch:        operationlog.KeyEpoch(policy.GetMinKeyEpoch()),
		MinMembershipEpoch: operationlog.MembershipEpoch(policy.GetMinMembershipEpoch()),
	}
}

func operationFromContract(operation *contracts.EncryptedOperation) operationlog.EncryptedOperation {
	return operationlog.EncryptedOperation{
		SpaceID:         operationlog.SpaceID(operation.GetSpaceId()),
		MemberID:        operationlog.MemberID(operation.GetMemberId()),
		DeviceID:        operationlog.DeviceID(operation.GetDeviceId()),
		OperationID:     operationlog.OperationID(operation.GetOperationId()),
		KeyEpoch:        operationlog.KeyEpoch(operation.GetKeyEpoch()),
		MembershipEpoch: operationlog.MembershipEpoch(operation.GetMembershipEpoch()),
		Ciphertext:      append([]byte(nil), operation.GetCiphertext()...),
		Nonce:           append([]byte(nil), operation.GetNonce()...),
		AssociatedData:  append([]byte(nil), operation.GetAssociatedData()...),
		CreatedAt:       time.Unix(0, operation.GetCreatedAtUnixNano()).UTC(),
	}
}

func commitmentToContract(commitment operationlog.OperationCommitment) *contracts.OperationCommitment {
	return &contracts.OperationCommitment{
		SpaceId:         string(commitment.SpaceID),
		OperationId:     string(commitment.OperationID),
		Sequence:        commitment.Sequence,
		Digest:          commitment.Digest,
		KeyEpoch:        uint64(commitment.KeyEpoch),
		MembershipEpoch: uint64(commitment.MembershipEpoch),
		CiphertextSize:  uint64(commitment.CiphertextSize),
	}
}

func reportToContract(report verification.Report) *contracts.VerificationReport {
	return &contracts.VerificationReport{
		OperationId:     string(report.OperationID),
		Digest:          report.Digest,
		Accepted:        report.Accepted,
		ProductionReady: report.ProductionReady,
		Mode:            report.Mode,
	}
}
