package internal

import (
	"context"
	"fmt"

	"github.com/GoCodeAlone/encrypted-spaces-go/keytrans"
	"github.com/GoCodeAlone/encrypted-spaces-go/poksho"
	"github.com/GoCodeAlone/encrypted-spaces-go/verification"
	"github.com/GoCodeAlone/encrypted-spaces-go/zkgroup"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func ExecuteEncryptedSpaceVerifyMembership(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.VerifyMembershipConfig, *contracts.VerifyMembershipInput],
) (*sdk.TypedStepResult[*contracts.VerifyMembershipOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space verify membership: input is required")
	}
	report, err := zkgroup.VerifyMembershipCredential(zkgroup.MembershipCredential{
		GroupID:      req.Input.GetGroupId(),
		MemberID:     req.Input.GetMemberId(),
		Issuer:       req.Input.GetIssuer(),
		ExpiresAt:    req.Input.GetExpiresAt(),
		ProofDigest:  req.Input.GetProofDigest(),
		UpstreamPath: req.Input.GetUpstreamPath(),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space verify membership: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.VerifyMembershipOutput]{
		Output: &contracts.VerifyMembershipOutput{Report: proofReport(req.Input.GetGroupId(), req.Input.GetProofDigest(), report.Accepted, report.ProductionReady)},
	}, nil
}

func ExecuteEncryptedSpaceVerifyOperation(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.VerifyOperationConfig, *contracts.VerifyOperationInput],
) (*sdk.TypedStepResult[*contracts.VerifyOperationOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space verify operation: input is required")
	}
	if req.Input.GetProofDigest() == "fake-proof" {
		return nil, fmt.Errorf("encrypted space verify operation: fake proof cannot be marked production")
	}
	report, err := poksho.VerifyProofTranscript(poksho.ProofTranscript{
		TranscriptID: req.Input.GetTranscriptId(),
		StatementID:  req.Input.GetStatementId(),
		WitnessHash:  req.Input.GetWitnessHash(),
		ProofDigest:  req.Input.GetProofDigest(),
		UpstreamPath: req.Input.GetUpstreamPath(),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space verify operation: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.VerifyOperationOutput]{
		Output: &contracts.VerifyOperationOutput{Report: proofReport(req.Input.GetTranscriptId(), req.Input.GetProofDigest(), report.Accepted, report.ProductionReady)},
	}, nil
}

func ExecuteEncryptedSpaceVerifyCheckpoint(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.VerifyCheckpointConfig, *contracts.VerifyCheckpointInput],
) (*sdk.TypedStepResult[*contracts.VerifyCheckpointOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space verify checkpoint: input is required")
	}
	report, err := keytrans.VerifyCheckpoint(keytrans.Checkpoint{
		CheckpointID: req.Input.GetCheckpointId(),
		TreeHead:     req.Input.GetTreeHead(),
		TreeSize:     req.Input.GetTreeSize(),
		ProofDigest:  req.Input.GetProofDigest(),
		UpstreamPath: req.Input.GetUpstreamPath(),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space verify checkpoint: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.VerifyCheckpointOutput]{
		Output: &contracts.VerifyCheckpointOutput{Report: proofReport(req.Input.GetCheckpointId(), req.Input.GetProofDigest(), report.Accepted, report.ProductionReady)},
	}, nil
}

func ExecuteEncryptedSpaceVectorReport(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.VectorReportConfig, *contracts.VectorReportInput],
) (*sdk.TypedStepResult[*contracts.VectorReportOutput], error) {
	report := verification.ProofCoverageReport()
	required := requiredDomainSet(req.Config.GetRequiredDomains())
	hasRequired := len(required) > 0
	output := &contracts.VectorReportOutput{
		UpstreamTag:          report.UpstreamTag,
		ProductionEquivalent: true,
		Rows:                 make([]*contracts.VectorCoverageRow, 0, len(report.Rows)),
		DeferredDomains:      []string{},
		Status:               "production-equivalent",
		NonVectorDomains:     []string{},
	}

	for _, row := range report.Rows {
		if !isProofCoverageDomain(row.Domain) {
			continue
		}
		if hasRequired {
			if _, ok := required[row.Domain]; !ok {
				continue
			}
			delete(required, row.Domain)
		}
		contractRow := &contracts.VectorCoverageRow{
			Domain:            row.Domain,
			Status:            row.Status,
			Vector:            row.Vector,
			Reason:            row.Reason,
			Notes:             row.Notes,
			NextUpstreamInput: row.NextUpstreamInput,
		}
		output.Rows = append(output.Rows, contractRow)
		if row.Status != "vector-backed" {
			output.ProductionEquivalent = false
			output.NonVectorDomains = append(output.NonVectorDomains, row.Domain)
			if row.Status == "deferred" {
				output.DeferredDomains = append(output.DeferredDomains, row.Domain)
			}
		}
	}
	for domain := range required {
		return nil, fmt.Errorf("encrypted space vector report: required domain %q not found", domain)
	}
	if !output.ProductionEquivalent {
		output.Status = "deferred"
	}
	if req.Config.GetRequireProductionEquivalence() && !output.ProductionEquivalent {
		return nil, fmt.Errorf("encrypted space vector report: production equivalence requires vector-backed domains, non-vector-backed domains: %v", output.NonVectorDomains)
	}
	return &sdk.TypedStepResult[*contracts.VectorReportOutput]{Output: output}, nil
}

func proofReport(operationID, proofDigest string, accepted, productionReady bool) *contracts.VerificationReport {
	return &contracts.VerificationReport{
		OperationId:     operationID,
		Digest:          proofDigest,
		Accepted:        accepted,
		ProductionReady: productionReady,
		Mode:            "production",
	}
}

func requiredDomainSet(domains []string) map[string]struct{} {
	if len(domains) == 0 {
		return nil
	}
	required := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		if domain == "" {
			continue
		}
		required[domain] = struct{}{}
	}
	return required
}

func isProofCoverageDomain(domain string) bool {
	switch domain {
	case "zkgroup", "zkcredential", "poksho", "keytrans", "message-backup", "svr-svrb":
		return true
	default:
		return false
	}
}
