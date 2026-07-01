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

func TestVectorReportStepReturnsPerDomainCoverage(t *testing.T) {
	result, err := ExecuteEncryptedSpaceVectorReport(context.Background(), sdk.TypedStepRequest[*contracts.VectorReportConfig, *contracts.VectorReportInput]{
		Config: &contracts.VectorReportConfig{},
		Input:  &contracts.VectorReportInput{},
	})
	if err != nil {
		t.Fatalf("VectorReport: %v", err)
	}
	output := result.Output
	if output.GetUpstreamTag() == "" {
		t.Fatal("upstream tag is empty")
	}
	if output.GetProductionEquivalent() {
		t.Fatal("vector report claimed production equivalence with deferred proof domains")
	}
	if output.GetStatus() != "deferred" {
		t.Fatalf("status = %q, want deferred", output.GetStatus())
	}

	rows := map[string]*contracts.VectorCoverageRow{}
	for _, row := range output.GetRows() {
		rows[row.GetDomain()] = row
		switch row.GetStatus() {
		case "vector-backed", "deterministic-only", "deferred":
		default:
			t.Fatalf("%s status = %q, want known coverage status", row.GetDomain(), row.GetStatus())
		}
	}
	for _, domain := range []string{"zkgroup", "zkcredential", "poksho", "keytrans", "message-backup", "svr-svrb"} {
		if rows[domain] == nil {
			t.Fatalf("missing coverage row %s", domain)
		}
	}
	if rows["zkgroup"].GetStatus() != "vector-backed" || rows["zkgroup"].GetVector() == "" {
		t.Fatalf("zkgroup row = %#v, want vector-backed with vector", rows["zkgroup"])
	}
	if rows["message-backup"].GetStatus() != "deferred" || rows["message-backup"].GetReason() == "" {
		t.Fatalf("message-backup row = %#v, want deferred with reason", rows["message-backup"])
	}
	assertStringSet(t, output.GetDeferredDomains(), []string{"message-backup", "svr-svrb"})
	assertStringSet(t, output.GetNonVectorDomains(), []string{"message-backup", "svr-svrb"})
}

func TestVectorReportStepFiltersRequiredDomains(t *testing.T) {
	result, err := ExecuteEncryptedSpaceVectorReport(context.Background(), sdk.TypedStepRequest[*contracts.VectorReportConfig, *contracts.VectorReportInput]{
		Config: &contracts.VectorReportConfig{RequiredDomains: []string{"zkgroup", "poksho"}},
		Input:  &contracts.VectorReportInput{},
	})
	if err != nil {
		t.Fatalf("VectorReport filtered: %v", err)
	}
	if !result.Output.GetProductionEquivalent() {
		t.Fatal("filtered vector-backed domains should be production equivalent")
	}
	if len(result.Output.GetRows()) != 2 {
		t.Fatalf("rows = %d, want 2", len(result.Output.GetRows()))
	}
	if len(result.Output.GetDeferredDomains()) != 0 {
		t.Fatalf("deferred domains = %v, want none", result.Output.GetDeferredDomains())
	}
	if len(result.Output.GetNonVectorDomains()) != 0 {
		t.Fatalf("non-vector domains = %v, want none", result.Output.GetNonVectorDomains())
	}
}

func TestVectorReportStepRefusesProductionEquivalenceClaimWithDeferredDomains(t *testing.T) {
	_, err := ExecuteEncryptedSpaceVectorReport(context.Background(), sdk.TypedStepRequest[*contracts.VectorReportConfig, *contracts.VectorReportInput]{
		Config: &contracts.VectorReportConfig{
			RequiredDomains:              []string{"zkgroup", "message-backup"},
			RequireProductionEquivalence: true,
		},
		Input: &contracts.VectorReportInput{},
	})
	if err == nil {
		t.Fatal("expected production-equivalence gate to reject deferred domain")
	}
}
