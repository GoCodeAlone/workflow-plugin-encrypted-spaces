package internal

import (
	"context"
	"testing"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func TestProofSteps(t *testing.T) {
	membershipDigest := "sha256:2f99cb90ee710be078aaf1b8cb9a22942c10f5965e5e39c1607a930fd6df7874"
	membership, err := ExecuteEncryptedSpaceVerifyMembership(context.Background(), sdk.TypedStepRequest[*contracts.VerifyMembershipConfig, *contracts.VerifyMembershipInput]{
		Config: &contracts.VerifyMembershipConfig{},
		Input: &contracts.VerifyMembershipInput{
			GroupId:      "space-1",
			MemberId:     "member-1",
			Issuer:       "issuer-1",
			ExpiresAt:    1893456000,
			ProofDigest:  membershipDigest,
			UpstreamPath: "java/shared/java/org/signal/libsignal/zkgroup/groups",
		},
	})
	if err != nil {
		t.Fatalf("VerifyMembership: %v", err)
	}
	if !membership.Output.GetReport().GetProductionReady() {
		t.Fatal("membership report is not production ready")
	}
	if got := membership.Output.GetReport().GetOperationId(); got != "space-1" {
		t.Fatalf("membership operation id = %q, want space-1", got)
	}
	if got := membership.Output.GetReport().GetDigest(); got != membershipDigest {
		t.Fatalf("membership digest = %q, want %s", got, membershipDigest)
	}

	operationDigest := "sha256:03362ba67e599e1ae3b3b34ef2734245fd784ba2948e6ee8835ba06d1019b3ff"
	operation, err := ExecuteEncryptedSpaceVerifyOperation(context.Background(), sdk.TypedStepRequest[*contracts.VerifyOperationConfig, *contracts.VerifyOperationInput]{
		Config: &contracts.VerifyOperationConfig{},
		Input: &contracts.VerifyOperationInput{
			TranscriptId: "transcript-1",
			StatementId:  "statement-1",
			WitnessHash:  "sha256:operation",
			ProofDigest:  operationDigest,
			UpstreamPath: "rust/poksho/src/proof.rs",
		},
	})
	if err != nil {
		t.Fatalf("VerifyOperation: %v", err)
	}
	if !operation.Output.GetReport().GetAccepted() {
		t.Fatal("operation proof was not accepted")
	}
	if got := operation.Output.GetReport().GetOperationId(); got != "transcript-1" {
		t.Fatalf("operation id = %q, want transcript-1", got)
	}
	if got := operation.Output.GetReport().GetDigest(); got != operationDigest {
		t.Fatalf("operation digest = %q, want %s", got, operationDigest)
	}

	checkpointDigest := "sha256:479338417f33b12df048fbe2180f58638636b2618d90ac6f807ed436ff881d8c"
	checkpoint, err := ExecuteEncryptedSpaceVerifyCheckpoint(context.Background(), sdk.TypedStepRequest[*contracts.VerifyCheckpointConfig, *contracts.VerifyCheckpointInput]{
		Config: &contracts.VerifyCheckpointConfig{},
		Input: &contracts.VerifyCheckpointInput{
			CheckpointId: "checkpoint-1",
			TreeHead:     "tree-head-1",
			TreeSize:     42,
			ProofDigest:  checkpointDigest,
			UpstreamPath: "rust/keytrans/src/verify.rs",
		},
	})
	if err != nil {
		t.Fatalf("VerifyCheckpoint: %v", err)
	}
	if !checkpoint.Output.GetReport().GetProductionReady() {
		t.Fatal("checkpoint report is not production ready")
	}
	if got := checkpoint.Output.GetReport().GetOperationId(); got != "checkpoint-1" {
		t.Fatalf("checkpoint operation id = %q, want checkpoint-1", got)
	}
	if got := checkpoint.Output.GetReport().GetDigest(); got != checkpointDigest {
		t.Fatalf("checkpoint digest = %q, want %s", got, checkpointDigest)
	}
}

func TestProofStepsRejectFakeProductionProof(t *testing.T) {
	_, err := ExecuteEncryptedSpaceVerifyOperation(context.Background(), sdk.TypedStepRequest[*contracts.VerifyOperationConfig, *contracts.VerifyOperationInput]{
		Config: &contracts.VerifyOperationConfig{},
		Input: &contracts.VerifyOperationInput{
			TranscriptId: "fake",
			StatementId:  "fake",
			WitnessHash:  "fake",
			ProofDigest:  "fake-proof",
			UpstreamPath: "fake",
		},
	})
	if err == nil {
		t.Fatal("expected fake production proof rejection")
	}
}
