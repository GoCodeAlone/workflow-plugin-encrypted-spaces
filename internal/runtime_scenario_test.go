package internal

import (
	"context"
	"testing"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func TestEncryptedSpaceRuntimeScenario(t *testing.T) {
	ctx := context.Background()

	memberUpdate, err := ExecuteEncryptedSpaceMemberUpdate(ctx, sdk.TypedStepRequest[*contracts.MemberUpdateConfig, *contracts.MemberUpdateInput]{
		Config: &contracts.MemberUpdateConfig{},
		Input: &contracts.MemberUpdateInput{
			SpaceId:  "space-1",
			Members:  []string{"member-1", "member-2"},
			MemberId: "member-3",
			Action:   "add",
			Reason:   "room membership sync",
		},
	})
	if err != nil {
		t.Fatalf("member update: %v", err)
	}
	if !memberUpdate.Output.GetMemberAllowed() {
		t.Fatal("room-derived member was not allowed")
	}

	appendResult, err := ExecuteEncryptedSpaceAppend(ctx, sdk.TypedStepRequest[*contracts.AppendConfig, *contracts.AppendInput]{
		Config: &contracts.AppendConfig{
			Retention:         &contracts.RetentionPolicy{MaxOperations: 10},
			AllowFakeVerifier: true,
		},
		Input: &contracts.AppendInput{Operation: testContractOperation("scenario-op-1")},
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	commitment := appendResult.Output.GetCommitment()

	eventPayload := map[string]any{
		"operation_id": commitment.GetOperationId(),
		"key_epoch":    commitment.GetKeyEpoch(),
		"result":       "appended",
	}
	auditMetadata := map[string]any{
		"operation_id":    commitment.GetOperationId(),
		"key_epoch":       commitment.GetKeyEpoch(),
		"ciphertext_size": commitment.GetCiphertextSize(),
		"result":          "appended",
	}

	if eventPayload["operation_id"] == "" {
		t.Fatal("event payload missing operation id")
	}
	for _, forbidden := range []string{"plaintext", "ciphertext", "key", "secret", "nonce"} {
		if _, ok := auditMetadata[forbidden]; ok {
			t.Fatalf("audit metadata leaked %s", forbidden)
		}
	}
	if auditMetadata["ciphertext_size"] != uint64(len(testContractOperation("scenario-op-1").GetCiphertext())) {
		t.Fatalf("ciphertext_size = %v, want ciphertext length", auditMetadata["ciphertext_size"])
	}

	vectorReport, err := ExecuteEncryptedSpaceVectorReport(ctx, sdk.TypedStepRequest[*contracts.VectorReportConfig, *contracts.VectorReportInput]{
		Config: &contracts.VectorReportConfig{
			RequiredDomains: []string{"zkgroup", "message-backup"},
		},
		Input: &contracts.VectorReportInput{},
	})
	if err != nil {
		t.Fatalf("vector report: %v", err)
	}
	if vectorReport.Output.GetProductionEquivalent() {
		t.Fatal("runtime scenario claimed production equivalence with message-backup deferred")
	}
	if got := vectorReport.Output.GetStatus(); got != "deferred" {
		t.Fatalf("runtime scenario vector status = %q, want deferred", got)
	}
}
