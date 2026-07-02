package internal

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
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

func TestEncryptedSpaceFileStateRuntimeScenario(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spaces.json")
	first, err := newEncryptedSpaceStateStoreModuleFromConfig("runtime-file-state", &contracts.StateStoreConfig{
		Backend:             "file",
		StoragePath:         path,
		AllowFileStateStore: true,
	})
	if err != nil {
		t.Fatalf("new file state store: %v", err)
	}
	if err := first.Start(t.Context()); err != nil {
		t.Fatalf("start file state store: %v", err)
	}

	state := initializedStateForTest(t, "runtime-space", "member-a")
	update, err := ExecuteEncryptedSpaceStateUpdate(t.Context(), sdk.TypedStepRequest[*contracts.StateUpdateConfig, *contracts.StateUpdateInput]{
		Input: &contracts.StateUpdateInput{
			State:    state,
			MemberId: "member-a",
			Action:   "remove",
			Reason:   "runtime scenario removal",
		},
	})
	if err != nil {
		t.Fatalf("remove member: %v", err)
	}
	save, err := ExecuteEncryptedSpaceStateSave(t.Context(), sdk.TypedStepRequest[*contracts.StateSaveConfig, *contracts.StateSaveInput]{
		Config: &contracts.StateSaveConfig{StateStore: "runtime-file-state"},
		Input:  &contracts.StateSaveInput{State: update.Output.GetState()},
	})
	if err != nil {
		t.Fatalf("save removed state: %v", err)
	}
	if err := first.Stop(t.Context()); err != nil {
		t.Fatalf("stop first store: %v", err)
	}

	second, err := newEncryptedSpaceStateStoreModuleFromConfig("runtime-file-state", &contracts.StateStoreConfig{
		Backend:             "file",
		StoragePath:         path,
		AllowFileStateStore: true,
	})
	if err != nil {
		t.Fatalf("new restarted store: %v", err)
	}
	if err := second.Start(t.Context()); err != nil {
		t.Fatalf("start restarted store: %v", err)
	}
	t.Cleanup(func() {
		_ = second.Stop(t.Context())
	})
	load, err := ExecuteEncryptedSpaceStateLoad(t.Context(), sdk.TypedStepRequest[*contracts.StateLoadConfig, *contracts.StateLoadInput]{
		Config: &contracts.StateLoadConfig{StateStore: "runtime-file-state"},
		Input:  &contracts.StateLoadInput{SpaceId: "runtime-space"},
	})
	if err != nil {
		t.Fatalf("load restarted state: %v", err)
	}
	check, err := ExecuteEncryptedSpaceMemberCheck(t.Context(), sdk.TypedStepRequest[*contracts.MemberCheckConfig, *contracts.MemberCheckInput]{
		Input: &contracts.MemberCheckInput{
			State:    load.Output.GetState(),
			MemberId: "member-a",
		},
	})
	if err != nil {
		t.Fatalf("check restarted removed member: %v", err)
	}
	if check.Output.GetMemberAllowed() || !check.Output.GetMemberRemoved() {
		t.Fatalf("removed member allowed=%v removed=%v, want allowed=false removed=true", check.Output.GetMemberAllowed(), check.Output.GetMemberRemoved())
	}

	raw, err := json.Marshal([]any{save.Output, load.Output, check.Output})
	if err != nil {
		t.Fatalf("marshal runtime outputs: %v", err)
	}
	outputs := strings.ToLower(string(raw))
	for _, forbidden := range []string{"plaintext", "proof-secret", "private-key", "key material"} {
		if strings.Contains(outputs, forbidden) {
			t.Fatalf("runtime outputs leaked forbidden term %q: %s", forbidden, outputs)
		}
	}
}
