package internal

import (
	"context"
	"testing"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func TestProofSteps(t *testing.T) {
	membership, err := ExecuteEncryptedSpaceVerifyMembership(context.Background(), sdk.TypedStepRequest[*contracts.VerifyMembershipConfig, *contracts.VerifyMembershipInput]{
		Config: &contracts.VerifyMembershipConfig{},
		Input: &contracts.VerifyMembershipInput{
			GroupId:      "space-1",
			MemberId:     "member-1",
			Issuer:       "issuer-1",
			ExpiresAt:    1893456000,
			ProofDigest:  "sha256:2f99cb90ee710be078aaf1b8cb9a22942c10f5965e5e39c1607a930fd6df7874",
			UpstreamPath: "java/shared/java/org/signal/libsignal/zkgroup/groups",
		},
	})
	if err != nil {
		t.Fatalf("VerifyMembership: %v", err)
	}
	if !membership.Output.GetReport().GetProductionReady() {
		t.Fatal("membership report is not production ready")
	}

	operation, err := ExecuteEncryptedSpaceVerifyOperation(context.Background(), sdk.TypedStepRequest[*contracts.VerifyOperationConfig, *contracts.VerifyOperationInput]{
		Config: &contracts.VerifyOperationConfig{},
		Input: &contracts.VerifyOperationInput{
			TranscriptId: "transcript-1",
			StatementId:  "statement-1",
			WitnessHash:  "sha256:operation",
			ProofDigest:  "sha256:03362ba67e599e1ae3b3b34ef2734245fd784ba2948e6ee8835ba06d1019b3ff",
			UpstreamPath: "rust/poksho/src/proof.rs",
		},
	})
	if err != nil {
		t.Fatalf("VerifyOperation: %v", err)
	}
	if !operation.Output.GetReport().GetAccepted() {
		t.Fatal("operation proof was not accepted")
	}

	checkpoint, err := ExecuteEncryptedSpaceVerifyCheckpoint(context.Background(), sdk.TypedStepRequest[*contracts.VerifyCheckpointConfig, *contracts.VerifyCheckpointInput]{
		Config: &contracts.VerifyCheckpointConfig{},
		Input: &contracts.VerifyCheckpointInput{
			CheckpointId: "checkpoint-1",
			TreeHead:     "tree-head-1",
			TreeSize:     42,
			ProofDigest:  "sha256:479338417f33b12df048fbe2180f58638636b2618d90ac6f807ed436ff881d8c",
			UpstreamPath: "rust/keytrans/src/verify.rs",
		},
	})
	if err != nil {
		t.Fatalf("VerifyCheckpoint: %v", err)
	}
	if !checkpoint.Output.GetReport().GetProductionReady() {
		t.Fatal("checkpoint report is not production ready")
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
