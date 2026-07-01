package internal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/GoCodeAlone/encrypted-spaces-go/operationlog"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func TestAppendVerifiedAcceptsVectorBackedProof(t *testing.T) {
	operation := testContractOperation("verified-op")
	commitment := commitmentForOperation(t, operation)

	result, err := ExecuteEncryptedSpaceAppendVerified(t.Context(), sdk.TypedStepRequest[*contracts.AppendVerifiedConfig, *contracts.AppendVerifiedInput]{
		Config: &contracts.AppendVerifiedConfig{Policy: &contracts.ProofPolicyConfig{RequireVectorBacked: true}},
		Input: &contracts.AppendVerifiedInput{
			Operation:          operation,
			ExpectedCommitment: commitment,
			Membership:         vectorMembershipProof(),
			Checkpoint:         vectorCheckpointProof(0),
		},
	})
	if err != nil {
		t.Fatalf("AppendVerified: %v", err)
	}
	if result.Output.GetCommitment().GetDigest() != commitment.GetDigest() {
		t.Fatalf("digest = %q, want %q", result.Output.GetCommitment().GetDigest(), commitment.GetDigest())
	}
	if len(result.Output.GetReports()) != 3 {
		t.Fatalf("reports = %d, want 3", len(result.Output.GetReports()))
	}
	if result.Output.GetEvidence().GetDigest() != commitment.GetDigest() {
		t.Fatalf("evidence digest = %q, want %q", result.Output.GetEvidence().GetDigest(), commitment.GetDigest())
	}
	assertAuditPayload(t, result.Output.GetAuditEventJson(), "encrypted_space.append_verified")
}

func TestAppendVerifiedRejectsTamperedProof(t *testing.T) {
	operation := testContractOperation("verified-op")
	commitment := commitmentForOperation(t, operation)
	tampered := operation
	tampered.Ciphertext = []byte("tampered ciphertext")

	_, err := ExecuteEncryptedSpaceAppendVerified(t.Context(), sdk.TypedStepRequest[*contracts.AppendVerifiedConfig, *contracts.AppendVerifiedInput]{
		Config: &contracts.AppendVerifiedConfig{Policy: &contracts.ProofPolicyConfig{RequireVectorBacked: true}},
		Input: &contracts.AppendVerifiedInput{
			Operation:          tampered,
			ExpectedCommitment: commitment,
			Membership:         vectorMembershipProof(),
			Checkpoint:         vectorCheckpointProof(0),
		},
	})
	if err == nil {
		t.Fatal("expected tampered operation rejection")
	}
}

func TestProofEvidenceRedactsPlaintextAndKeyMaterial(t *testing.T) {
	operation := testContractOperation("evidence-op")
	operation.Ciphertext = []byte("plaintext operation body and key material")
	operation.Nonce = []byte("secret nonce key material")
	operation.AssociatedData = []byte("associated key material")
	commitment := commitmentForOperation(t, operation)

	result, err := ExecuteEncryptedSpaceProofEvidence(t.Context(), sdk.TypedStepRequest[*contracts.ProofEvidenceConfig, *contracts.ProofEvidenceInput]{
		Config: &contracts.ProofEvidenceConfig{},
		Input: &contracts.ProofEvidenceInput{
			Commitment: commitment,
			Reports: []*contracts.ProofReport{{
				Domain:          "zkgroup.membership",
				Accepted:        true,
				ProductionReady: true,
				UpstreamPath:    "java/shared/java/org/signal/libsignal/zkgroup/groups",
			}},
		},
	})
	if err != nil {
		t.Fatalf("ProofEvidence: %v", err)
	}
	for _, forbidden := range []string{"plaintext operation body", "secret nonce", "associated key material", "key material"} {
		if strings.Contains(result.Output.GetJson(), forbidden) {
			t.Fatalf("evidence leaked %q: %s", forbidden, result.Output.GetJson())
		}
	}
	if !strings.Contains(result.Output.GetJson(), commitment.GetDigest()) {
		t.Fatalf("evidence missing digest %q: %s", commitment.GetDigest(), result.Output.GetJson())
	}
	assertAuditPayload(t, result.Output.GetAuditEventJson(), "encrypted_space.proof_evidence")
}

func commitmentForOperation(t *testing.T, operation *contracts.EncryptedOperation) *contracts.OperationCommitment {
	t.Helper()
	log, err := operationlog.NewLog(operationlog.LogOptions{})
	if err != nil {
		t.Fatalf("NewLog: %v", err)
	}
	commitment, err := log.Append(operationFromContract(operation))
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	return commitmentToContract(commitment)
}

func vectorMembershipProof() *contracts.MembershipProof {
	return &contracts.MembershipProof{
		GroupId:      "space-1",
		MemberId:     "member-1",
		Issuer:       "issuer-1",
		ExpiresAt:    1893456000,
		ProofDigest:  "sha256:2f99cb90ee710be078aaf1b8cb9a22942c10f5965e5e39c1607a930fd6df7874",
		UpstreamPath: "java/shared/java/org/signal/libsignal/zkgroup/groups",
	}
}

func vectorCheckpointProof(previousTreeSize uint64) *contracts.CheckpointProof {
	return &contracts.CheckpointProof{
		CheckpointId:     "checkpoint-1",
		TreeHead:         "tree-head-1",
		TreeSize:         42,
		ProofDigest:      "sha256:479338417f33b12df048fbe2180f58638636b2618d90ac6f807ed436ff881d8c",
		UpstreamPath:     "rust/keytrans/src/verify.rs",
		PreviousTreeSize: previousTreeSize,
	}
}

func assertAuditPayload(t *testing.T, raw, wantEvent string) {
	t.Helper()
	if raw == "" {
		t.Fatal("audit payload is empty")
	}
	var payload struct {
		EventType string `json:"event_type"`
		Plugin    string `json:"plugin"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("audit payload is not JSON: %v", err)
	}
	if payload.EventType != wantEvent {
		t.Fatalf("event_type = %q, want %q", payload.EventType, wantEvent)
	}
	if payload.Plugin != "workflow-plugin-encrypted-spaces" {
		t.Fatalf("plugin = %q, want workflow-plugin-encrypted-spaces", payload.Plugin)
	}
	if _, err := time.Parse(time.RFC3339, payload.Timestamp); err != nil {
		t.Fatalf("timestamp = %q, want RFC3339: %v", payload.Timestamp, err)
	}
}
