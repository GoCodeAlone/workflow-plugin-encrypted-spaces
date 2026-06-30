package internal

import (
	"context"
	"testing"
	"time"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func TestOperationSteps(t *testing.T) {
	appendResult, err := ExecuteEncryptedSpaceAppend(context.Background(), sdk.TypedStepRequest[*contracts.AppendConfig, *contracts.AppendInput]{
		Config: &contracts.AppendConfig{
			Retention:         &contracts.RetentionPolicy{MaxOperations: 10},
			AllowFakeVerifier: true,
		},
		Input: &contracts.AppendInput{Operation: testContractOperation("op-1")},
	})
	if err != nil {
		t.Fatalf("ExecuteEncryptedSpaceAppend: %v", err)
	}
	if appendResult.Output.GetCommitment().GetOperationId() != "op-1" {
		t.Fatalf("operation id = %q, want op-1", appendResult.Output.GetCommitment().GetOperationId())
	}
	if appendResult.Output.GetCommitment().GetDigest() == "" {
		t.Fatal("append returned empty digest")
	}
	if appendResult.Output.GetVerification().GetProductionReady() {
		t.Fatal("fake append verification reported production ready")
	}

	checkpoint := &contracts.FastForwardCheckpoint{
		SpaceId:         "space-1",
		ThroughSequence: appendResult.Output.GetCommitment().GetSequence(),
		OperationId:     appendResult.Output.GetCommitment().GetOperationId(),
		Digest:          appendResult.Output.GetCommitment().GetDigest(),
		KeyEpoch:        appendResult.Output.GetCommitment().GetKeyEpoch(),
		MembershipEpoch: appendResult.Output.GetCommitment().GetMembershipEpoch(),
	}
	fastForward, err := ExecuteEncryptedSpaceFastForward(context.Background(), sdk.TypedStepRequest[*contracts.FastForwardConfig, *contracts.FastForwardInput]{
		Config: &contracts.FastForwardConfig{},
		Input: &contracts.FastForwardInput{
			Commitment: appendResult.Output.GetCommitment(),
			Checkpoint: checkpoint,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteEncryptedSpaceFastForward: %v", err)
	}
	if !fastForward.Output.GetAccepted() {
		t.Fatal("fast-forward checkpoint was not accepted")
	}
}

func TestEpochAndMemberSteps(t *testing.T) {
	rotated, err := ExecuteEncryptedSpaceEpochRotate(context.Background(), sdk.TypedStepRequest[*contracts.EpochRotateConfig, *contracts.EpochRotateInput]{
		Config: &contracts.EpochRotateConfig{},
		Input: &contracts.EpochRotateInput{
			SpaceId:         "space-1",
			CurrentKeyEpoch: 4,
			Reason:          "scheduled",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteEncryptedSpaceEpochRotate: %v", err)
	}
	if rotated.Output.GetKeyEpoch() != 5 {
		t.Fatalf("key epoch = %d, want 5", rotated.Output.GetKeyEpoch())
	}

	updated, err := ExecuteEncryptedSpaceMemberUpdate(context.Background(), sdk.TypedStepRequest[*contracts.MemberUpdateConfig, *contracts.MemberUpdateInput]{
		Config: &contracts.MemberUpdateConfig{},
		Input: &contracts.MemberUpdateInput{
			SpaceId:  "space-1",
			Members:  []string{"member-1", "member-2"},
			MemberId: "member-2",
			Action:   "remove",
			Reason:   "access revoked",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteEncryptedSpaceMemberUpdate: %v", err)
	}
	if updated.Output.GetMembershipEpoch() != 2 {
		t.Fatalf("membership epoch = %d, want 2", updated.Output.GetMembershipEpoch())
	}
	if updated.Output.GetMemberAllowed() {
		t.Fatal("removed member is still allowed")
	}
}

func testContractOperation(id string) *contracts.EncryptedOperation {
	return &contracts.EncryptedOperation{
		SpaceId:           "space-1",
		MemberId:          "member-1",
		DeviceId:          "device-1",
		OperationId:       id,
		KeyEpoch:          1,
		MembershipEpoch:   1,
		Ciphertext:        []byte("ciphertext-" + id),
		Nonce:             []byte("nonce-" + id),
		AssociatedData:    []byte("aad-" + id),
		CreatedAtUnixNano: time.Date(2026, 6, 30, 20, 0, 0, 0, time.UTC).UnixNano(),
	}
}
